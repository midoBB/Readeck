// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bookmarks

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"io"
	"time"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/blowfish" //nolint:staticcheck

	"codeberg.org/readeck/readeck/configs"
	"codeberg.org/readeck/readeck/pkg/base58"
)

// EncryptID returns an 160-bit base58 encoded encrypted ID and timestamp.
// This uses Encrypt-then-MAC authentication, with 2 keys.
//
// Initial message is:
//
// -----------------------------------
// | 32-bit         | 32-bit         |
// | timestamp      | ID             |
// -----------------------------------
//
// Finale message is:
//
// ------------------------------------------------------------------------------
// | 64-bit ciphertext               | 64-bit MAC                      | 32-bit |
// |                                 |                                 | salt   |
// ------------------------------------------------------------------------------
//
// This returns a base58 encoded string.
func EncryptID(id uint64, expires time.Time) (string, error) {
	packed := make([]byte, 8)
	binary.LittleEndian.PutUint32(packed[:4], uint32(id))

	// Timestamp is expressed in minutes after rounding.
	// This gives us plenty of time in a 32-bit unsigned integer.
	binary.LittleEndian.PutUint32(packed[4:], uint32(expires.Round(time.Minute).Unix()/60))

	// salt and keys
	salt := make([]byte, 4)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	// Get keys
	k1, _ := configs.Keys.Expand("bookmark_share"+string(salt), 32)
	k2, _ := configs.Keys.Expand("bookmark_share_mac"+string(salt), 32)

	// Encrypt
	cipher, err := blowfish.NewCipher(k1)
	if err != nil {
		return "", err
	}
	msg := make([]byte, 20)
	cipher.Encrypt(msg, packed)

	// MAC
	h, err := blake2b.New(8, k2)
	if err != nil {
		return "", err
	}
	h.Write(msg[0:8])

	copy(msg[8:], h.Sum(nil))
	copy(msg[16:], salt)

	return base58.EncodeToString(msg), nil
}

// DecryptID deciphers a base58 encoded encrypted value
// into a timestamp and a unsigned integer.
func DecryptID(text string) (time.Time, uint64, error) {
	// Load the base64 encoded value. It must be exactly 16 bytes.
	msg, err := base58.DecodeString(text)
	if err != nil {
		return time.Time{}, 0, err
	}

	if len(msg) != 20 {
		return time.Time{}, 0, errors.New("invalid size")
	}

	// Get mac and salt
	salt := msg[16:]
	k2, _ := configs.Keys.Expand("bookmark_share_mac"+string(salt), 32)

	// Verify MAC
	h, err := blake2b.New(8, k2)
	if err != nil {
		return time.Time{}, 0, err
	}
	h.Write(msg[0:8])
	if subtle.ConstantTimeCompare(h.Sum(nil), msg[8:16]) != 1 {
		return time.Time{}, 0, errors.New("invalid data")
	}

	// Decrypt content
	k1, _ := configs.Keys.Expand("bookmark_share"+string(salt), 32)
	cipher, err := blowfish.NewCipher(k1)
	if err != nil {
		return time.Time{}, 0, err
	}
	packed := make([]byte, 8)
	cipher.Decrypt(packed, msg)

	ts := binary.LittleEndian.Uint32(packed[4:])
	return time.Unix(int64(ts)*60, 0), uint64(binary.LittleEndian.Uint32(packed[:4])), nil
}

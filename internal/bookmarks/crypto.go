// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bookmarks

import (
	"crypto/aes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"time"

	"codeberg.org/readeck/readeck/configs"
)

func encryptID(id uint64, expires time.Time) (string, error) {
	// Pack the expiry timestamp and the bookmark ID in a 128bit packet
	buf := make([]byte, 16)
	now := uint64(expires.Unix())
	binary.LittleEndian.PutUint64(buf[0:], now)
	binary.LittleEndian.PutUint64(buf[8:], uint64(id))

	// Encrypt the packet. There's not need for complex IV since
	// it's the right size to encrypt the initial packet.
	cipher, err := aes.NewCipher(configs.CookieBlockKey())
	if err != nil {
		return "", err
	}
	res := make([]byte, 16)
	cipher.Encrypt(res, buf)

	// Return the base64 encoded encrypted value
	return base64.RawURLEncoding.EncodeToString(res), nil
}

func decryptID(value string) (time.Time, uint64, error) {
	// Load the base64 encoded value. It must be exactly 16 bytes.
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return time.Time{}, 0, err
	}
	if len(data) != 16 {
		return time.Time{}, 0, errors.New("invalid size")
	}

	// Decrypt this value into a packet (ts + id)
	cipher, err := aes.NewCipher(configs.CookieBlockKey())
	if err != nil {
		return time.Time{}, 0, err
	}
	packed := make([]byte, 16)
	cipher.Decrypt(packed, data)

	ts := time.Unix(int64(binary.LittleEndian.Uint64(packed[0:])), 0)
	id := binary.LittleEndian.Uint64(packed[8:])

	return ts, id, nil
}

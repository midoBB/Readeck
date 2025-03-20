// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tokens

import (
	"crypto/subtle"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/blake2b"

	"codeberg.org/readeck/readeck/configs"
	"codeberg.org/readeck/readeck/pkg/base58"
)

const macSize = 19 // Token hash size (152bit)

// EncodeToken returns a base58 encoded token and its mac.
// The whole token is 280-bit long.
func EncodeToken(uid string) (string, error) {
	id, err := base58.DecodeUUID(uid)
	if err != nil {
		return "", err
	}

	h, err := blake2b.New(macSize, configs.Keys.TokenKey())
	if err != nil {
		return "", err
	}
	h.Write(id[:])

	res := make([]byte, len(id)+macSize)
	copy(res, id[:])
	copy(res[len(id):], h.Sum(nil))

	return base58.EncodeToString(res), nil
}

// DecodeToken returns a token ID, using the signed, base58 encoded
// token. It checks its length and mac.
func DecodeToken(token string) (string, error) {
	msg, err := base58.DecodeString(token)
	if err != nil {
		return "", err
	}
	if len(msg) != 16+macSize {
		return "", errors.New("invalid token size")
	}

	id, mac := msg[:16], msg[16:]

	h, err := blake2b.New(macSize, configs.Keys.TokenKey())
	if err != nil {
		return "", err
	}
	h.Write(id)

	if subtle.ConstantTimeCompare(mac, h.Sum(nil)) != 1 {
		return "", errors.New("invalid token")
	}

	return base58.EncodeUUID(uuid.UUID(id)), nil
}

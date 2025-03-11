// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package base58

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ErrInvalidLenght is returned on UUID decoding errors.
var ErrInvalidLenght = errors.New("invalid length")

// NewUUID returns a base58 encoded UUIDv4.
func NewUUID() string {
	return EncodeUUID(uuid.New())
}

// EncodeUUID encodes an UUID to a base58 string.
func EncodeUUID(u uuid.UUID) string {
	return EncodeToString(u[:])
}

// DecodeUUID decodes a string to an UUID.
func DecodeUUID(s string) (u uuid.UUID, err error) {
	var b []byte
	if b, err = DecodeString(s); err != nil {
		return
	}

	if len(b) != 16 {
		err = fmt.Errorf("%w (wants 16, got %d)", ErrInvalidLenght, len(b))
		return
	}

	return uuid.UUID(b), nil
}

// SPDX-FileCopyrightText: © 2013-2015 The btcsuite developers
//
// SPDX-License-Identifier: ISC

// Package base58 implements base58 encoding
package base58

import (
	"errors"
	"fmt"
)

// An Encoding is a radix 58 encoding/decoding scheme, defined by a
// 58-character alphabet.
type Encoding struct {
	encode    [58]byte
	decodeMap [256]uint8
}

const (
	decodeMapInitialize = "" +
		"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
		"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
		"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
		"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
		"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
		"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
		"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
		"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
		"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
		"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
		"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
		"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
		"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
		"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
		"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff" +
		"\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff"
	invalidIndex = '\xff'
)

// ErrInvalidChar is returned on decoding errors.
var ErrInvalidChar = errors.New("invalid base58 string")

// NewEncoding returns a new [Encoding] defined by the given alphabet,
// which must be a 58-bit string that contains unique byte values and does
// not contain CR or LF characters.
func NewEncoding(encoder string) *Encoding {
	if len(encoder) != 58 {
		panic("encoding alphabet is not 58-bytes long")
	}

	e := new(Encoding)
	copy(e.encode[:], encoder)
	copy(e.decodeMap[:], decodeMapInitialize)

	for i := range len(encoder) {
		switch {
		case encoder[i] == '\n' || encoder[i] == '\r':
			panic("encoding alphabet contains newline character")
		case e.decodeMap[encoder[i]] != invalidIndex:
			panic("encoding alphabet includes duplicate symbols")
		}
		e.decodeMap[encoder[i]] = uint8(i)

	}

	return e
}

// StdEncoding is an [Encoding] with the Bitcoin alphabet.
var StdEncoding = NewEncoding("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

// EncodeToString is a shortcut to [Encoding.EncodeToString] on [StdEncoding].
func EncodeToString(src []byte) string {
	return StdEncoding.EncodeToString(src)
}

// DecodeString is a shortcut to [Encoding.DecodeString] on [StdEncoding].
func DecodeString(s string) ([]byte, error) {
	return StdEncoding.DecodeString(s)
}

// EncodeToString returns the base58 encoding of src.
func (enc *Encoding) EncodeToString(src []byte) string {
	// Since the conversion is from base256 to base58, the max possible number
	// of bytes of output per input byte is log_58(256) ~= 1.37.  Thus, the max
	// total output size is ceil(len(input) * 137/100).  Rather than worrying
	// about the ceiling, just add one even if it isn't needed since the final
	// output is truncated to the right size at the end.
	output := make([]byte, (len(src)*137/100)+1)

	// Encode to base58 in reverse order to avoid extra calculations to
	// determine the final output size in favor of just keeping track while
	// iterating.
	var index int
	for _, r := range src {
		// Multiply each byte in the output by 256 and encode to base58 while
		// propagating the carry.
		val := uint32(r)
		for i, b := range output[:index] {
			val += uint32(b) << 8
			output[i] = byte(val % 58)
			val /= 58
		}
		for ; val > 0; val /= 58 {
			output[index] = byte(val % 58)
			index++
		}
	}

	// Replace the calculated remainders with their corresponding base58 digit.
	for i, b := range output[:index] {
		output[i] = enc.encode[b]
	}

	// Account for the leading zeros in the input.  They are appended since the
	// encoding is happening in reverse order.
	for _, r := range src {
		if r != 0 {
			break
		}

		output[index] = enc.encode[0]
		index++
	}

	// Truncate the output buffer to the actual number of encoded bytes and
	// reverse it since it was calculated in reverse order.
	output = output[:index:index]
	for i := range index / 2 {
		output[i], output[index-1-i] = output[index-1-i], output[i]
	}

	return string(output)
}

// DecodeString returns the bytes represented by the base58 string s.
func (enc *Encoding) DecodeString(s string) ([]byte, error) {
	if len(s) == 0 {
		return []byte{}, nil
	}

	// The max possible output size is when a base58 encoding consists of
	// nothing but the alphabet character at index 0 which would result in the
	// same number of bytes as the number of input chars.
	output := make([]byte, len(s))

	// Encode to base256 in reverse order to avoid extra calculations to
	// determine the final output size in favor of just keeping track while
	// iterating.
	var index int
	for _, r := range []byte(s) {
		// Invalid base58 character.
		val := uint32(enc.decodeMap[r])
		if val == 255 {
			return nil, fmt.Errorf("%w (got %q)", ErrInvalidChar, r)
		}

		// Multiply each byte in the output by 58 and encode to base256 while
		// propagating the carry.
		for i, b := range output[:index] {
			val += uint32(b) * 58
			output[i] = byte(val)
			val >>= 8
		}
		for ; val > 0; val >>= 8 {
			output[index] = byte(val)
			index++
		}
	}

	// Account for the leading zeros in the input.  They are appended since the
	// encoding is happening in reverse order.
	for _, r := range []byte(s) {
		if r != enc.encode[0] {
			break
		}

		output[index] = 0
		index++
	}

	// Truncate the output buffer to the actual number of decoded bytes and
	// reverse it since it was calculated in reverse order.
	output = output[:index:index]
	for i := range index / 2 {
		output[i], output[index-1-i] = output[index-1-i], output[i]
	}

	return output, nil
}

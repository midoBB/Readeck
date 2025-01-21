// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package forms

import (
	"fmt"
	"strconv"
	"time"

	"github.com/araddon/dateparse"
)

// nilText is a text null value.
// In an URL of form value, it would be a field with %EF%BC%80 value.
// On an HTML field, it's simply &#xff00.
const nilText = "\uff00"

// ValueFlags holds the flags a value can get.
type ValueFlags int8

const (
	// IsOk must be true when a field was successfully decoded.
	IsOk ValueFlags = 1 << iota
	// IsBound is the flag for a bound value.
	IsBound
	// IsNil is the flag for a nil value.
	IsNil
	// IsEmpty is the flag for an empty value.
	IsEmpty
)

// IsOk returns true when [IsOk] is present in the flags.
func (f ValueFlags) IsOk() bool {
	return f&IsOk > 0
}

// IsBound returns true when [IsBound] is present in the flags.
func (f ValueFlags) IsBound() bool {
	return f&IsBound > 0
}

// IsNil returns true when [IsNil] is present in the flags.
func (f ValueFlags) IsNil() bool {
	return f&IsNil > 0
}

// IsEmpty returns true when [IsEmpty] is present in the flags.
func (f ValueFlags) IsEmpty() bool {
	return f&IsEmpty > 0
}

// Value is the value holder.
type Value[T any] struct {
	F ValueFlags
	V T
}

// NewValue returns a new, empty [Value].
func NewValue[T any]() (value Value[T]) {
	value.F = IsEmpty
	return
}

func (v *Value[T]) set(value T) {
	v.F = IsOk
	v.V = value
}

// Decoder describes a typed value decoder.
type Decoder[T any] interface {
	DecodeAny(data any) Value[T]
	DecodeText(text string) Value[T]
}

type valueStringer[T any] interface {
	ValueString(v T) string
}

type valueDecoder[T any] struct {
	decodeAny  func(data any) Value[T]
	decodeText func(text string) Value[T]
	stringer   func(T) string
}

func (d valueDecoder[T]) DecodeAny(data any) Value[T] {
	return d.decodeAny(data)
}

func (d valueDecoder[T]) DecodeText(text string) Value[T] {
	return d.decodeText(text)
}

func (d valueDecoder[T]) ValueString(v T) string {
	if d.stringer != nil {
		return d.stringer(v)
	}
	return fmt.Sprintf("%v", v)
}

// NewValueDecoder creates a [Decoder] with a decoder functions pair.
func NewValueDecoder[T any](
	decodeAny func(data any) Value[T],
	decodeText func(text string) Value[T],
	stringer func(T) string,
) Decoder[T] {
	return valueDecoder[T]{
		decodeAny:  decodeAny,
		decodeText: decodeText,
		stringer:   stringer,
	}
}

// DecodeString is a [Value] decoder for strings.
var DecodeString = NewValueDecoder(
	func(data any) Value[string] {
		value := NewValue[string]()

		if data == nil {
			value.F |= IsNil
			return value
		}

		if V, ok := data.(string); ok {
			value.F |= IsOk
			if V != "" {
				value.set(V)
			}
		}

		return value
	},
	func(text string) Value[string] {
		value := NewValue[string]()
		if text != "" {
			value.set(text)
		} else {
			value.F |= IsOk
		}
		return value
	},
	func(v string) string {
		return v
	},
)

// DecodeBoolean is a [Value] decoder for booleans.
var DecodeBoolean = NewValueDecoder(
	func(data any) Value[bool] {
		value := NewValue[bool]()

		if data == nil {
			value.F |= IsNil
			return value
		}

		if V, ok := data.(bool); ok {
			value.set(V)
		}

		return value
	},
	func(text string) Value[bool] {
		value := NewValue[bool]()

		switch text {
		case "":
			value.F |= IsNil
			return value
		case "on":
			value.set(true)
		default:
			if v, err := strconv.ParseBool(text); err == nil {
				value.set(v)
			}
		}

		return value
	},
	strconv.FormatBool,
)

// DecodeInt is a [Value] decoder for integers.
var DecodeInt = NewValueDecoder(
	func(data any) Value[int] {
		value := NewValue[int]()

		if data == nil {
			value.F |= IsNil
			return value
		}

		switch V := data.(type) {
		case int:
			value.set(V)
		case float64:
			if V == float64(int(V)) {
				value.set(int(V))
			}
		}

		return value
	},
	func(text string) Value[int] {
		value := NewValue[int]()

		if len(text) == 0 {
			value.F |= IsNil
		}
		v, err := strconv.ParseInt(text, 10, 0)
		if err != nil {
			return value
		}
		value.set(int(v))

		return value
	},
	strconv.Itoa,
)

// DecodeTime is a [Value] decoder for [time.Time] values.
var DecodeTime = NewValueDecoder(
	func(data any) Value[time.Time] {
		value := NewValue[time.Time]()

		if data == nil {
			value.F |= IsNil
			return value
		}

		switch V := data.(type) {
		case string:
			if len(V) == 0 {
				value.F |= IsNil
				return value
			}
			if v, err := dateparse.ParseAny(V); err == nil {
				value.set(v)
			}
		case time.Time:
			value.set(V)
		}

		if value.V.IsZero() {
			value.F |= IsEmpty
		}

		return value
	},
	func(text string) Value[time.Time] {
		value := NewValue[time.Time]()

		if len(text) == 0 {
			value.F |= IsNil
			return value
		}

		if v, err := dateparse.ParseAny(text); err == nil {
			value.set(v)
		}

		if value.V.IsZero() {
			value.F |= IsEmpty
		}

		return value
	},
	func(v time.Time) string {
		return v.Format("2006-01-02")
	},
)

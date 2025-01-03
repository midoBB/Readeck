// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package forms_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"codeberg.org/readeck/readeck/pkg/forms/v2"
)

func TestValueFlags(t *testing.T) {
	tests := []struct {
		f       forms.ValueFlags
		isOk    bool
		isBound bool
		isNil   bool
		isEmpty bool
	}{
		{forms.ValueFlags(0), false, false, false, false},
		{forms.IsNil, false, false, true, false},
		{forms.IsOk | forms.IsBound | forms.IsNil | forms.IsEmpty, true, true, true, true},
		{(forms.IsBound | forms.IsNil) &^ forms.IsBound, false, false, true, false},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			assert := require.New(t)
			assert.Equal(test.isOk, test.f.IsOk())
			assert.Equal(test.isBound, test.f.IsBound())
			assert.Equal(test.isNil, test.f.IsNil())
			assert.Equal(test.isEmpty, test.f.IsEmpty())
		})
	}
}

type anyValueTest[T any] struct {
	value    any
	expected forms.Value[T]
}

func runAnyDecoder[T any](decoder forms.Decoder[T], tests []anyValueTest[T]) func(t *testing.T) {
	return func(t *testing.T) {
		for i, test := range tests {
			t.Run(strconv.Itoa(i+1), func(t *testing.T) {
				v := decoder.DecodeAny(test.value)
				require.Exactly(t, test.expected, v)
			})
		}
	}
}

type textValueTest[T any] struct {
	text     string
	expected forms.Value[T]
}

func runTextDecoder[T any](decoder forms.Decoder[T], tests []textValueTest[T]) func(t *testing.T) {
	return func(t *testing.T) {
		for i, test := range tests {
			t.Run(strconv.Itoa(i+1), func(t *testing.T) {
				v := decoder.DecodeText(test.text)
				require.Exactly(t, test.expected, v)
			})
		}
	}
}

func TestDecoder(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		t.Run("any", runAnyDecoder(forms.DecodeString, []anyValueTest[string]{
			{nil, forms.Value[string]{
				F: forms.IsNil | forms.IsEmpty,
			}},
			{"", forms.Value[string]{
				F: forms.IsOk | forms.IsEmpty,
			}},
			{"abc", forms.Value[string]{
				V: "abc",
				F: forms.IsOk,
			}},
			{123, forms.Value[string]{
				F: forms.IsEmpty,
			}},
		}))

		t.Run("text", runTextDecoder(forms.DecodeString, []textValueTest[string]{
			{"", forms.Value[string]{
				F: forms.IsEmpty | forms.IsOk,
			}},
			{"abc", forms.Value[string]{
				V: "abc",
				F: forms.IsOk,
			}},
		}))
	})

	t.Run("boolean", func(t *testing.T) {
		t.Run("any", runAnyDecoder(forms.DecodeBoolean, []anyValueTest[bool]{
			{nil, forms.Value[bool]{
				F: forms.IsNil | forms.IsEmpty,
			}},
			{true, forms.Value[bool]{
				V: true,
				F: forms.IsOk,
			}},
			{false, forms.Value[bool]{
				V: false,
				F: forms.IsOk,
			}},
			{"abc", forms.Value[bool]{
				V: false,
				F: forms.IsEmpty,
			}},
		}))

		t.Run("text", runTextDecoder(forms.DecodeBoolean, []textValueTest[bool]{
			{"", forms.Value[bool]{
				F: forms.IsNil | forms.IsEmpty,
			}},
			{"on", forms.Value[bool]{
				V: true,
				F: forms.IsOk,
			}},
			{"f", forms.Value[bool]{
				V: false,
				F: forms.IsOk,
			}},
			{"t", forms.Value[bool]{
				V: true,
				F: forms.IsOk,
			}},
			{"0", forms.Value[bool]{
				V: false,
				F: forms.IsOk,
			}},
			{"1", forms.Value[bool]{
				V: true,
				F: forms.IsOk,
			}},
			{"abc", forms.Value[bool]{
				F: forms.IsEmpty,
			}},
		}))
	})

	t.Run("int", func(t *testing.T) {
		t.Run("any", runAnyDecoder(forms.DecodeInt, []anyValueTest[int]{
			{nil, forms.Value[int]{
				F: forms.IsNil | forms.IsEmpty,
			}},
			{10, forms.Value[int]{
				V: 10,
				F: forms.IsOk,
			}},
			{float64(123.0), forms.Value[int]{
				V: 123,
				F: forms.IsOk,
			}},
			{"abc", forms.Value[int]{
				F: forms.IsEmpty,
			}},
		}))

		t.Run("text", runTextDecoder(forms.DecodeInt, []textValueTest[int]{
			{"", forms.Value[int]{
				F: forms.IsNil | forms.IsEmpty,
			}},
			{"10", forms.Value[int]{
				V: 10,
				F: forms.IsOk,
			}},
			{"-5", forms.Value[int]{
				V: -5,
				F: forms.IsOk,
			}},
			{"abc", forms.Value[int]{
				F: forms.IsEmpty,
			}},
		}))
	})

	t.Run("time.Time", func(t *testing.T) {
		d1, _ := time.Parse(time.DateOnly, "2020-01-30")
		d2, _ := time.Parse(time.DateTime, "2020-01-30 14:24:06")

		t.Run("any", runAnyDecoder(forms.DecodeTime, []anyValueTest[time.Time]{
			{nil, forms.Value[time.Time]{
				F: forms.IsNil | forms.IsEmpty,
			}},
			{time.Time{}, forms.Value[time.Time]{
				V: time.Time{},
				F: forms.IsOk | forms.IsEmpty,
			}},
			{"", forms.Value[time.Time]{
				F: forms.IsNil | forms.IsEmpty,
			}},
			{d1, forms.Value[time.Time]{
				V: d1,
				F: forms.IsOk,
			}},
			{"2020-01-30 14:24:06", forms.Value[time.Time]{
				V: d2,
				F: forms.IsOk,
			}},
			{"abcd", forms.Value[time.Time]{
				F: forms.IsEmpty,
			}},
		}))

		t.Run("text", runTextDecoder(forms.DecodeTime, []textValueTest[time.Time]{
			{"", forms.Value[time.Time]{
				F: forms.IsNil | forms.IsEmpty,
			}},
			{time.Time{}.Format(time.RFC3339), forms.Value[time.Time]{
				F: forms.IsOk | forms.IsEmpty,
			}},
			{"2020-01-30", forms.Value[time.Time]{
				V: d1,
				F: forms.IsOk,
			}},
			{"2020-01-30 14:24:06", forms.Value[time.Time]{
				V: d2,
				F: forms.IsOk,
			}},
			{"abcd", forms.Value[time.Time]{
				F: forms.IsEmpty,
			}},
		}))
	})
}

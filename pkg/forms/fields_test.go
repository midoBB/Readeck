// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package forms_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"codeberg.org/readeck/readeck/pkg/forms"
)

type fieldTypeTest struct {
	data  string
	isNil bool
	value interface{}
	str   string
	err   error
}

func testField(t *testing.T, test fieldTypeTest, f forms.Field, decoder func([]byte) error) {
	assert := require.New(t)
	err := decoder([]byte(test.data))
	if test.err == nil {
		assert.NoError(err)
	} else {
		assert.Error(err)
		assert.EqualError(err, test.err.Error())
	}
	assert.True(f.IsBound(), "field is bound")
	assert.Equal(test.isNil, f.IsNil(), "null field")
	assert.Equal(test.value, f.Value(), "field value")
	assert.Equal(test.str, f.String(), "field string")
}

func TestTextField(t *testing.T) {
	assert := require.New(t)
	var field interface{} = forms.NewTextField("test")
	f, ok := field.(forms.Field)

	assert.True(ok)
	assert.Equal("test", f.Name())
	assert.False(f.IsBound())
	assert.True(f.IsNil())
	assert.Nil(f.Value())

	f.Set("value")
	assert.False(f.IsBound())
	assert.False(f.IsNil())
	assert.Equal("value", f.Value())

	f.Set(nil)
	assert.False(f.IsBound())
	assert.True(f.IsNil())
	assert.Nil(f.Value())

	t.Run("bind json", func(t *testing.T) {
		tests := []fieldTypeTest{
			{
				data:  `""`,
				isNil: false,
				value: "",
				str:   "",
			},
			{
				data:  `"test"`,
				isNil: false,
				value: "test",
				str:   "test",
			},
			{
				data:  "null",
				isNil: true,
				value: nil,
				str:   "",
			},
			{
				data:  "1234",
				isNil: false,
				value: "",
				str:   "",
				err:   forms.ErrInvalidType,
			},
		}

		field := forms.NewTextField("test")
		for i, test := range tests {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				testField(t, test, field, field.UnmarshalJSON)
			})
		}
	})

	t.Run("bind param", func(t *testing.T) {
		tests := []fieldTypeTest{
			{
				data:  "",
				isNil: false,
				value: "",
				str:   "",
			},
			{
				data:  "\x00",
				isNil: true,
				value: nil,
				str:   "",
			},
			{
				data:  "foo",
				isNil: false,
				value: "foo",
				str:   "foo",
			},
			{
				data:  "bar",
				isNil: false,
				value: "bar",
				str:   "bar",
			},
		}

		field := forms.NewTextField("test")
		for i, test := range tests {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				testField(t, test, field, field.UnmarshalText)
			})
		}
	})
}

func TestBooleanField(t *testing.T) {
	assert := require.New(t)
	var field interface{} = forms.NewBooleanField("test")
	f, ok := field.(forms.Field)

	assert.True(ok)
	assert.Equal("test", f.Name())
	assert.False(f.IsBound())
	assert.True(f.IsNil())
	assert.Nil(f.Value())
	assert.Equal("", f.String())

	f.Set(true)
	assert.False(f.IsBound())
	assert.False(f.IsNil())
	assert.Exactly(true, f.Value())
	assert.Equal("true", f.String())

	f.Set(nil)
	assert.False(f.IsBound())
	assert.True(f.IsNil())
	assert.Nil(f.Value())
	assert.Equal("", f.String())

	t.Run("bind json", func(t *testing.T) {
		tests := []fieldTypeTest{
			{
				data:  "true",
				isNil: false,
				value: true,
				str:   "true",
			},
			{
				data:  "false",
				isNil: false,
				value: false,
				str:   "false",
			},
			{
				data:  "null",
				isNil: true,
				value: nil,
				str:   "",
			},
			{
				data:  "1234",
				isNil: false,
				value: false,
				str:   "false",
				err:   forms.ErrInvalidType,
			},
		}

		field := forms.NewBooleanField("test")
		for i, test := range tests {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				testField(t, test, field, field.UnmarshalJSON)
			})
		}
	})

	t.Run("bind param", func(t *testing.T) {
		tests := []fieldTypeTest{
			{
				data:  "",
				isNil: true,
				value: nil,
				str:   "",
			},
			{
				data:  "on",
				isNil: false,
				value: true,
				str:   "true",
			},
			{
				data:  "f",
				isNil: false,
				value: false,
				str:   "false",
			},
			{
				data:  "t",
				isNil: false,
				value: true,
				str:   "true",
			},
			{
				data:  "whatever",
				isNil: false,
				value: false,
				str:   "false",
				err:   forms.ErrInvalidValue,
			},
		}

		field := forms.NewBooleanField("test")
		for i, test := range tests {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				testField(t, test, field, field.UnmarshalText)
			})
		}
	})
}

func TestIntegerField(t *testing.T) {
	assert := require.New(t)
	var field interface{} = forms.NewIntegerField("test")
	f, ok := field.(forms.Field)

	assert.True(ok)
	assert.Equal("test", f.Name())
	assert.False(f.IsBound())
	assert.True(f.IsNil())
	assert.Nil(f.Value())
	assert.Equal("", f.String())

	f.Set(10)
	assert.False(f.IsBound())
	assert.False(f.IsNil())
	assert.Equal(10, f.Value())
	assert.Equal("10", f.String())

	f.Set(nil)
	assert.False(f.IsBound())
	assert.True(f.IsNil())
	assert.Nil(f.Value())
	assert.Equal("", f.String())

	assert.False(f.Set("abc"))

	t.Run("bind json", func(t *testing.T) {
		tests := []fieldTypeTest{
			{
				data:  "10",
				isNil: false,
				value: 10,
				str:   "10",
			},
			{
				data:  "-5",
				isNil: false,
				value: -5,
				str:   "-5",
			},
			{
				data:  "102.5",
				isNil: true,
				value: nil,
				str:   "",
				err:   forms.ErrInvalidType,
			},
			{
				data:  "null",
				isNil: true,
				value: nil,
				str:   "",
			},
			{
				data:  "[123]",
				isNil: true,
				value: nil,
				str:   "",
				err:   forms.ErrInvalidType,
			},
			{
				data:  `"abcd"`,
				isNil: true,
				value: nil,
				str:   "",
				err:   forms.ErrInvalidType,
			},
		}

		field := forms.NewIntegerField("test")
		for i, test := range tests {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				testField(t, test, field, field.UnmarshalJSON)
			})
		}
	})

	t.Run("bind param", func(t *testing.T) {
		tests := []fieldTypeTest{
			{
				data:  "10",
				isNil: false,
				value: 10,
				str:   "10",
			},
			{
				data:  "-5",
				isNil: false,
				value: -5,
				str:   "-5",
			},
			{
				data:  "102.5",
				isNil: true,
				value: nil,
				str:   "",
				err:   forms.ErrInvalidType,
			},
			{
				data:  "\x00",
				isNil: true,
				value: nil,
				str:   "",
			},
			{
				data:  "whatever",
				isNil: true,
				value: nil,
				str:   "",
				err:   forms.ErrInvalidType,
			},
		}

		field := forms.NewIntegerField("test")
		for i, test := range tests {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				testField(t, test, field, field.UnmarshalText)
			})
		}
	})
}

func TestDatetimeField(t *testing.T) {
	assert := require.New(t)
	var field interface{} = forms.NewDatetimeField("test")
	f, ok := field.(forms.Field)
	d, _ := time.Parse("2006-01-02", "2020-01-30")

	assert.True(ok)
	assert.Equal("test", f.Name())
	assert.False(f.IsBound())
	assert.True(f.IsNil())
	assert.Nil(f.Value())
	assert.Equal("", f.String())

	f.Set(nil)
	assert.False(f.IsBound())
	assert.True(f.IsNil())
	assert.Nil(f.Value())
	assert.Equal("", f.String())

	f.Set(time.Time{})
	assert.False(f.IsBound())
	assert.True(f.IsNil())
	assert.Nil(f.Value())
	assert.Equal("", f.String())

	f.Set(d)
	assert.False(f.IsBound())
	assert.False(f.IsNil())
	assert.Equal(d, f.Value().(time.Time))
	assert.Equal("2020-01-30", f.String())

	f.Set(&d)
	assert.False(f.IsBound())
	assert.False(f.IsNil())
	assert.Equal(d, f.Value().(time.Time))
	assert.Equal("2020-01-30", f.String())

	t.Run("bind json", func(t *testing.T) {
		tests := []fieldTypeTest{
			{
				data:  `""`,
				isNil: true,
				value: nil,
				str:   "",
			},
			{
				data:  `"2020-01-30"`,
				isNil: false,
				value: d,
				str:   "2020-01-30",
			},
			{
				data:  "null",
				isNil: true,
				value: nil,
				str:   "",
			},
			{
				data:  `"blaaa"`,
				isNil: true,
				value: nil,
				str:   "",
				err:   errors.New("invalid datetime format"),
			},
		}

		field := forms.NewDatetimeField("test")
		for i, test := range tests {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				testField(t, test, field, field.UnmarshalJSON)
			})
		}
	})

	t.Run("bind param", func(t *testing.T) {
		tests := []fieldTypeTest{
			{
				data:  "",
				isNil: true,
				value: nil,
				str:   "",
			},
			{
				data:  "2020-01-30",
				isNil: false,
				value: d,
				str:   "2020-01-30",
			},
			{
				data:  "\x00",
				isNil: true,
				value: nil,
				str:   "",
			},
			{
				data:  "blaaa",
				isNil: true,
				value: nil,
				str:   "",
				err:   errors.New("invalid datetime format"),
			},
		}

		field := forms.NewDatetimeField("test")
		for i, test := range tests {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				testField(t, test, field, field.UnmarshalText)
			})
		}
	})
}

func TestMultipartField(t *testing.T) {
	t.Run("multipart", func(t *testing.T) {
		assert := require.New(t)

		txt := `
--foo
Content-Disposition: form-data; name="data"; filename="blob"
Content-Type: application/octet-stream

test value
abc
--foo--
		`
		mr := multipart.NewReader(strings.NewReader(txt), "foo")
		f, err := mr.ReadForm(2 << 20)
		assert.NoError(err)
		part := f.File["data"][0]

		field := forms.NewFileField("data")
		assert.True(field.Set(part))

		r, err := field.Open()
		assert.NoError(err)
		defer assert.NoError(r.Close())

		value, err := io.ReadAll(r)
		assert.NoError(err)
		assert.Equal("test value\nabc", string(value))
	})

	t.Run("reader", func(t *testing.T) {
		assert := require.New(t)
		txt := "test value\nabc"

		field := forms.NewFileField("data")
		assert.True(field.Set(strings.NewReader(txt)))

		r, err := field.Open()
		assert.NoError(err)
		defer assert.NoError(r.Close())

		value, err := io.ReadAll(r)
		assert.NoError(err)
		assert.Equal("test value\nabc", string(value))
	})

	t.Run("unset", func(t *testing.T) {
		assert := require.New(t)
		field := forms.NewFileField("data")

		r, err := field.Open()
		assert.ErrorIs(err, forms.ErrInvalidValue)
		assert.Nil(r)
	})

	t.Run("invalid", func(t *testing.T) {
		assert := require.New(t)
		field := forms.NewFileField("data")
		assert.False(field.Set(0))

		r, err := field.Open()
		assert.ErrorIs(err, forms.ErrInvalidValue)
		assert.Nil(r)
	})
}

func TestListField(t *testing.T) {
	assert := require.New(t)
	var field interface{} = forms.NewListField("test",
		func(n string) forms.Field {
			return forms.NewIntegerField(n)
		},
		forms.DefaultListConverter)
	f, ok := field.(forms.Field)

	assert.True(ok)
	assert.Equal("test", f.Name())
	assert.False(f.IsBound())
	assert.True(f.IsNil())
	assert.Nil(f.Value())
	assert.Equal("", f.String())

	f.Set(nil)
	assert.False(f.IsBound())
	assert.True(f.IsNil())
	assert.Nil(f.Value())
	assert.Equal("", f.String())

	assert.True(f.Set([]int{1, 2, 3, 4}))
	assert.False(f.IsBound())
	assert.False(f.IsNil())
	assert.Equal([]interface{}{1, 2, 3, 4}, f.Value())

	assert.False(f.Set("value"))
	assert.False(f.IsBound())
	assert.True(f.IsNil())
	assert.Nil(f.Value())

	assert.False(f.Set([]bool{true, false}))
	assert.False(f.IsBound())
	assert.True(f.IsNil())
	assert.Nil(f.Value())

	assert.True(f.Set([]interface{}{1, 2, 3, 4}))
	assert.False(f.IsBound())
	assert.False(f.IsNil())
	assert.Equal([]interface{}{1, 2, 3, 4}, f.Value())

	assert.False(f.Set([]interface{}{"a", "b"}))
	assert.False(f.IsBound())
	assert.True(f.IsNil())
	assert.Nil(f.Value())

	t.Run("bind param", func(t *testing.T) {
		tests := []fieldTypeTest{
			{
				data:  "test=",
				isNil: true,
				value: nil,
				str:   "",
			},
			{
				data:  "test=%00",
				isNil: true,
				value: nil,
				str:   "",
			},
			{
				data:  "test=1&test=2",
				isNil: false,
				value: []int{1, 2},
				str:   "[1,2]",
			},
			{
				data:  "test=2&test=3&test=abc",
				isNil: false,
				value: []int{2, 3},
				str:   "[2,3]",
				err:   forms.ErrInvalidType,
			},
			{
				data:  "test=%00&test=8&test=10",
				isNil: false,
				value: []int{8, 10},
				str:   "[8,10]",
			},
			{
				data:  "test=blaaa",
				isNil: true,
				value: nil,
				str:   "",
				err:   forms.ErrInvalidType,
			},
			{
				data:  "test=200",
				isNil: false,
				value: []int{200},
				str:   "[200]",
				err:   errors.New("must be lower or equal than 100"),
			},
		}

		field := forms.NewListField("test", func(n string) forms.Field {
			return forms.NewIntegerField(n, forms.Lte(100))
		}, func(values []forms.Field) interface{} {
			res := make([]int, len(values))
			for i, x := range values {
				res[i] = x.Value().(int)
			}
			return res
		})
		for i, test := range tests {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				// We must reset the field on each test
				field.Set(nil)

				testField(t, test, field, func(b []byte) error {
					// It's what forms.UnmarshalValues would do.
					values, _ := url.ParseQuery(string(b))
					errs := forms.Errors{}
					for _, x := range values[field.Name()] {
						err := field.UnmarshalText([]byte(x))
						if err != nil {
							errs = append(errs, err)
						}
					}
					if len(errs) > 0 {
						return errs
					}
					return nil
				})
			})
		}
	})

	t.Run("bind json", func(t *testing.T) {
		tests := []fieldTypeTest{
			{
				data:  "null",
				isNil: true,
				value: nil,
				str:   "",
			},
			{
				data:  "[]",
				isNil: true,
				value: nil,
				str:   "",
			},
			{
				data:  "[1, 2]",
				isNil: false,
				value: []int{1, 2},
				str:   "[1,2]",
			},
			{
				data:  `[2,3,"abc"]`,
				isNil: true,
				value: nil,
				str:   "",
				err:   forms.ErrInvalidType,
			},
			{
				data:  "[null, 8, 10]",
				isNil: false,
				value: []int{8, 10},
				str:   "[8,10]",
			},
			{
				data:  `["bla"]`,
				isNil: true,
				value: nil,
				str:   "",
				err:   forms.ErrInvalidType,
			},
			{
				data:  "[200]",
				isNil: false,
				value: []int{200},
				str:   "[200]",
				err:   errors.New("must be lower or equal than 100"),
			},
		}

		field := forms.NewListField("test", func(n string) forms.Field {
			return forms.NewIntegerField(n, forms.Lte(100))
		}, func(values []forms.Field) interface{} {
			res := make([]int, len(values))
			for i, x := range values {
				res[i] = x.Value().(int)
			}
			return res
		})
		for i, test := range tests {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				// fmt.Printf("++++++ %#v\n", test.data)
				// We must reset the field on each test
				field.Set(nil)
				// err := json.Unmarshal([]byte(test.data), &field)
				// if err != nil {
				// 	println("!!!!!", err.Error())
				// }
				// fmt.Printf("$$$$$ %#v\n", field.Value())

				testField(t, test, field, func(b []byte) error {
					return json.Unmarshal(b, field)
				})
			})
		}
	})
}

func TestChoiceListField(t *testing.T) {
	assert := require.New(t)
	var field interface{} = forms.NewListField("test",
		func(n string) forms.Field {
			return forms.NewTextField(n)
		},
		forms.DefaultListConverter)
	f, ok := field.(forms.Field)

	assert.True(ok)

	f.(*forms.ListField).SetChoices(forms.Choices{
		{"a", "A"},
		{"b", "B"},
		{"c", "C"},
	})

	assert.Equal("test", f.Name())
	assert.False(f.IsBound())
	assert.True(f.IsNil())
	assert.Nil(f.Value())
	assert.Equal("", f.String())

	assert.True(f.Set([]string{"a", "b"}))
	assert.False(f.IsBound())
	assert.False(f.IsNil())
	assert.Equal([]any{"a", "b"}, f.Value())

	assert.False(f.Set([]int{1, 2}))
	assert.False(f.IsBound())
	assert.True(f.IsNil())
	assert.Nil(f.Value())

	assert.True(f.Set([]string{"a", "f"}))
	assert.EqualError(forms.ValidateField(f, f.Validators()...), "f is not a valid value")
}

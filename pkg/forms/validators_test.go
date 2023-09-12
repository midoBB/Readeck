// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package forms_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"codeberg.org/readeck/readeck/pkg/forms"
)

type fieldValidatorTest struct {
	f      forms.Field
	value  interface{}
	expect interface{}
	errors []string
}

func testValidator(t *testing.T, test fieldValidatorTest) {
	test.f.UnmarshalJSON([]byte("null"))
	test.f.Set(test.value)
	errors := forms.ValidateField(test.f, test.f.Validators()...)
	assert.Len(t, errors, len(test.errors))
	if len(test.errors) > 0 {
		for i, e := range errors {
			assert.EqualError(t, e, test.errors[i])
		}
	} else {
		assert.Equal(t, test.expect, test.f.Value())
	}
}

func TestTrim(t *testing.T) {
	tests := []fieldValidatorTest{
		// Trim
		{
			f:      forms.NewTextField("test", forms.Trim),
			value:  " 1234  ",
			expect: "1234",
		},
		{
			f:      forms.NewTextField("test", forms.Trim),
			value:  "\t1234  \r",
			expect: "1234",
		},
		{
			f:      forms.NewTextField("test", forms.Trim),
			value:  nil,
			expect: nil,
		},
		{
			f:      forms.NewBooleanField("test", forms.Trim),
			value:  true,
			expect: true,
		},
		// Required
		{
			f:      forms.NewTextField("test", forms.Required),
			value:  "abc",
			expect: "abc",
		},
		{
			f:      forms.NewTextField("test", forms.Required),
			value:  "",
			expect: "",
			errors: []string{"field is required"},
		},
		{
			f:      forms.NewTextField("test", forms.Required),
			value:  nil,
			expect: nil,
			errors: []string{"field is required"},
		},
		{
			f:      forms.NewBooleanField("test", forms.Required),
			value:  nil,
			expect: nil,
			errors: []string{"field is required"},
		},
		{
			f:      forms.NewBooleanField("test", forms.Required),
			value:  true,
			expect: true,
		},
		{
			f:      forms.NewBooleanField("test", forms.Required),
			value:  false,
			expect: false,
		},
		// Trim + required
		{
			f:      forms.NewTextField("test", forms.Trim, forms.Required),
			value:  "   \t \r ",
			expect: "",
			errors: []string{"field is required"},
		},
		// RequiredOrNil
		{
			f:      forms.NewTextField("test", forms.RequiredOrNil),
			value:  "",
			expect: "",
			errors: []string{"field is required"},
		},
		{
			f:      forms.NewTextField("test", forms.RequiredOrNil),
			value:  nil,
			expect: nil,
		},
		{
			f:      forms.NewBooleanField("test", forms.RequiredOrNil),
			value:  nil,
			expect: nil,
		},
		{
			f:      forms.NewBooleanField("test", forms.RequiredOrNil),
			value:  false,
			expect: false,
		},
		// IsEmail
		{
			f:      forms.NewTextField("test", forms.IsEmail),
			value:  "test@example.org",
			expect: "test@example.org",
		},
		{
			f:      forms.NewTextField("test", forms.IsEmail),
			value:  "test@example@.org",
			expect: "test@example@.org",
			errors: []string{"not a valid email address"},
		},
		{
			f:      forms.NewTextField("test", forms.IsEmail),
			value:  "@test",
			expect: "@test",
			errors: []string{"not a valid email address"},
		},
		{
			f:      forms.NewTextField("test", forms.IsEmail),
			value:  "test@",
			expect: "test@",
			errors: []string{"not a valid email address"},
		},
		{
			f:      forms.NewTextField("test", forms.IsEmail),
			value:  "foo",
			expect: "foo",
			errors: []string{"not a valid email address"},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			testValidator(t, test)
		})
	}
}

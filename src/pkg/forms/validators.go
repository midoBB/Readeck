// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package forms

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

var (
	ErrRequired     = errors.New("field is required")         // ErrRequired is a required field
	ErrInvalidEmail = errors.New("not a valid email address") // ErrInvalidEmail is an invalid e-mail address
	ErrInvalidURL   = errors.New("invalid URL")               // ErrInvalidURL is an invalid URL
)

// FieldValidator is a function that validates and/or alters a field.
type FieldValidator func(f Field) error

// ValidateField performs the field validation and/or alteration.
func ValidateField(f Field, validators ...FieldValidator) Errors {
	res := Errors{}

	for _, validator := range validators {
		if err := validator(f); err != nil {
			res = append(res, err)
		}
	}

	return res
}

// Chain applies multiple validators but stops at the first error
func Chain(validators ...FieldValidator) FieldValidator {
	return func(f Field) error {
		for _, v := range validators {
			if err := v(f); err != nil {
				return err
			}
		}
		return nil
	}
}

// Trim return a validator that trims spaces from the value when it's a string.
func Trim(f Field) error {
	if f.IsNil() {
		return nil
	}

	if v, ok := f.Value().(string); ok {
		f.Set(strings.TrimSpace(v))
	}

	return nil
}

// Required check that the field is not null or empty.
func Required(f Field) error {
	if !f.IsBound() || f.IsNil() || f.String() == "" {
		return ErrRequired
	}
	return nil
}

// RequiredOrNil checks that the field is not empty if it's not null.
func RequiredOrNil(f Field) error {
	if !f.IsNil() && f.String() == "" {
		return ErrRequired
	}
	return nil
}

// StringValidator is a helper function that returns a validator from a
// simple function and an error message.
func StringValidator(validator func(v string) bool, err error) FieldValidator {
	return func(f Field) error {
		if f.IsNil() {
			return nil
		}

		v, ok := f.Value().(string)
		if !ok {
			return ErrInvalidType
		}

		if !validator(v) {
			return err
		}

		return nil
	}
}

// IsEmail performs a rough check of the email address. That is, it
// only checks for the presence of "@", only once and in the string.
var IsEmail = StringValidator(func(v string) bool {
	return strings.Count(v, "@") == 1 && !(strings.HasPrefix(v, "@") || strings.HasSuffix(v, "@"))
}, ErrInvalidEmail)

// IsValidURL checks that the input value is a valid URL and matches the given
// schemes.
func IsValidURL(schemes ...string) FieldValidator {
	return StringValidator(func(v string) bool {
		u, err := url.Parse(v)
		if err != nil {
			return false
		}

		if !slices.Contains(schemes, u.Scheme) {
			return false
		}
		return u.Hostname() != ""
	}, ErrInvalidURL)
}

// Gte returns a integer validator that checks if a value is greater or equal than a parameter.
func Gte(value int) FieldValidator {
	return func(f Field) error {
		if v, ok := f.Value().(int); ok {
			if v < value {
				return fmt.Errorf("must be greater or equal than %d", value)
			}
		}
		return nil
	}
}

// Lte returns a integer validator that checks if a value is lower or equal than a parameter.
func Lte(value int) FieldValidator {
	return func(f Field) error {
		if v, ok := f.Value().(int); ok {
			if v > value {
				return fmt.Errorf("must be lower or equal than %d", value)
			}
		}
		return nil
	}
}

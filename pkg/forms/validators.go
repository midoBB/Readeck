package forms

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
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
		return errors.New("field is required")
	}
	return nil
}

// RequiredOrNil checks that the field is not empty if it's not null.
func RequiredOrNil(f Field) error {
	if !f.IsNil() && f.String() == "" {
		return errors.New("field is required")
	}
	return nil
}

// StringValidator is a helper function that returns a validator from a
// simple function and an error message.
func StringValidator(validator func(v string) bool, message string) FieldValidator {
	return func(f Field) error {
		if f.IsNil() {
			return nil
		}

		v, ok := f.Value().(string)
		if !ok {
			return ErrInvalidType
		}

		if !validator(v) {
			return errors.New(message)
		}

		return nil
	}
}

// IsEmail performs a rough check of the email address. That is, it
// only checks for the presence of "@", only once and in the string.
var IsEmail = StringValidator(func(v string) bool {
	return strings.Count(v, "@") == 1 && !(strings.HasPrefix(v, "@") || strings.HasSuffix(v, "@"))
}, "not a valid email address")

// IsValidURL checks that the input value is a valid URL and matches the given
// schemes.
func IsValidURL(schemes ...string) FieldValidator {
	validSchemes := map[string]bool{}
	for _, k := range schemes {
		validSchemes[k] = true
	}
	return StringValidator(func(v string) bool {
		u, err := url.Parse(v)
		if err != nil {
			return false
		}

		return validSchemes[u.Scheme]
	}, "invalid URL")
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

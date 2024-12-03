// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package forms

import (
	"errors"
	"fmt"
)

// Translator describes a type that implements a translation method.
type Translator interface {
	Gettext(string, ...interface{}) string
	Pgettext(ctx, str string, vars ...interface{}) string
}

// FormError is a form's or field's error that contains an error message
// and arguments.
type FormError struct {
	err  error
	args []interface{}
}

// Error returns the untranslated error.
func (p FormError) Error() string {
	return fmt.Sprintf(p.err.Error(), p.args...)
}

// Is implements error identification.
func (p FormError) Is(err error) bool {
	return errors.Unwrap(err) == p.err
}

// Unwrap implements error unwrap.
func (p FormError) Unwrap() error {
	return p.err
}

// Translate returns the translated error using
// the given translator.
func (p FormError) Translate(tr Translator) string {
	return tr.Gettext(p.err.Error(), p.args...)
}

// newError returns a new FormError.
func newError(msg string, args ...interface{}) FormError {
	return FormError{errors.New(msg), args}
}

// Gettext is an alias for newError so it can be picked up by a locales extractor.
var Gettext = newError

// localizedError associates an error with a translator.
// If the error implements a Translate(translator) method
// it will be used to translate the error when it's added
// to the form's error list.
type localizedError struct {
	err error
	tr  Translator
}

func (le localizedError) Unwrap() error {
	return le.err
}

func (le localizedError) Error() string {
	if err, ok := le.err.(interface{ Translate(Translator) string }); ok && le.tr != nil {
		return err.Translate(le.tr)
	}

	return le.err.Error()
}

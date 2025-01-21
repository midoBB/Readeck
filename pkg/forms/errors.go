// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package forms

import (
	"encoding/json"
	"strings"
)

// Errors is an error list.
type Errors []error

func (e Errors) Error() string {
	return e.String()
}

func (e Errors) String() string {
	if len(e) == 0 {
		return ""
	}

	res := make([]string, len(e))
	for i, x := range e {
		res[i] = x.Error()
	}
	return strings.Join(res, ", ")
}

// MarshalJSON implements [json.Marshaler].
func (e Errors) MarshalJSON() ([]byte, error) {
	if len(e) == 0 {
		return json.Marshal(nil)
	}

	res := make([]string, len(e))
	for i := range e {
		res[i] = e[i].Error()
	}

	return json.Marshal(res)
}

type fatalError struct {
	err error
}

// FatalError returns an error that has the effect to stop any
// subsequent validation.
func FatalError(err error) error {
	return &fatalError{err}
}

func (e *fatalError) Error() string {
	return e.err.Error()
}

func (e *fatalError) Err() error {
	return e.err
}

// unwrapErrors returns an [error] list, unwrapping any
// error that implements Unwrap (like any [errors.Join]'s result).
func unwrapErrors(errs ...error) (res []error) {
	for _, err := range errs {
		if err == nil {
			continue
		}
		if err, ok := err.(interface{ Unwrap() []error }); ok {
			res = append(res, err.Unwrap()...)
			continue
		}
		res = append(res, err)
	}
	return
}

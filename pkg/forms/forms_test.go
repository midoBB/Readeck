// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package forms_test

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/kinbiko/jsonassert"
	"github.com/stretchr/testify/require"

	"codeberg.org/readeck/readeck/pkg/forms"
)

type formTest struct {
	data   string
	result string
}

type formMarshaler interface {
	forms.Binder
	MarshalJSON() ([]byte, error)
}

func testForm(t *testing.T, test formTest, f formMarshaler) {
	require.True(t, f.IsBound())
	data, err := f.MarshalJSON()
	if err != nil {
		panic(err)
	}
	jsonassert.New(t).Assertf(string(data), test.result)
	if t.Failed() {
		t.Errorf("Received JSON: %s\n", string(data))
		t.FailNow()
	}
}

type simpleForm struct {
	*forms.Form
}

func newSimpleForm() *simpleForm {
	return &simpleForm{forms.Must(
		forms.NewTextField("name"),
		forms.NewIntegerField("id", forms.Required),
	)}
}

type customValidationForm struct {
	*forms.Form
}

func newCustomValidationForm() *customValidationForm {
	return &customValidationForm{forms.Must(
		forms.NewTextField("name"),
	)}
}

func (f *customValidationForm) Validate() {
	if f.Get("name").String() == "nope" {
		f.AddErrors("name", errors.New("forbidden value"))
	}
}

func TestSimpleForm(t *testing.T) {
	t.Run("from json", func(t *testing.T) {
		tests := []formTest{
			{
				data: `{"name": "test", "id": 2}`,
				result: `{
					"is_valid": true,
					"errors": null,
					"fields": {
						"id": {
							"is_null": false,
							"is_bound": true,
							"value": 2,
							"errors": null
						},
						"name": {
							"is_null": false,
							"is_bound": true,
							"value": "test",
							"errors": null
						}
					}
				}`,
			},
			{
				data: "",
				result: `{
					"is_valid": false,
					"errors": [
						"Invalid input data"
					],
					"fields": {
						"id": {
							"is_null": true,
							"is_bound": false,
							"value": null,
							"errors": [
								"field is required"
							]
						},
						"name": {
							"is_null": true,
							"is_bound": false,
							"value": null,
							"errors": null
						}
					}
				}`,
			},
			{
				data: `{"name": 123}`,
				result: `{
					"is_valid": false,
					"errors": null,
					"fields": {
						"id": {
							"is_null": true,
							"is_bound": false,
							"value": null,
							"errors": [
								"field is required"
							]
						},
						"name": {
							"is_null": false,
							"is_bound": true,
							"value": "",
							"errors": [
								"invalid type"
							]
						}
					}
				}`,
			},
		}

		for i, test := range tests {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				f := newSimpleForm()
				require.False(t, f.IsBound())

				r := strings.NewReader(test.data)
				forms.UnmarshalJSON(f, r)
				testForm(t, test, f)
			})
		}
	})

	t.Run("from url encoded", func(t *testing.T) {
		tests := []formTest{
			{
				data: `name=test&id=2`,
				result: `{
					"is_valid": true,
					"errors": null,
					"fields": {
						"id": {
							"is_null": false,
							"is_bound": true,
							"value": 2,
							"errors": null
						},
						"name": {
							"is_null": false,
							"is_bound": true,
							"value": "test",
							"errors": null
						}
					}
				}`,
			},
			{
				data: "",
				result: `{
					"is_valid": false,
					"errors": null,
					"fields": {
						"id": {
							"is_null": true,
							"is_bound": false,
							"value": null,
							"errors": [
								"field is required"
							]
						},
						"name": {
							"is_null": true,
							"is_bound": false,
							"value": null,
							"errors": null
						}
					}
				}`,
			},
			{
				data: `name=123`,
				result: `{
					"is_valid": false,
					"errors": null,
					"fields": {
						"id": {
							"is_null": true,
							"is_bound": false,
							"value": null,
							"errors": [
								"field is required"
							]
						},
						"name": {
							"is_null": false,
							"is_bound": true,
							"value": "123",
							"errors": null
						}
					}
				}`,
			},
		}

		for i, test := range tests {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				f := newSimpleForm()
				require.False(t, f.IsBound())

				values, err := url.ParseQuery(test.data)
				require.NoError(t, err)
				forms.UnmarshalValues(f, values)
				testForm(t, test, f)
			})
		}
	})
}

func TestCustomValidation(t *testing.T) {
	t.Run("from json", func(t *testing.T) {
		tests := []formTest{
			{
				data: `{"name": "test"}`,
				result: `{
					"is_valid": true,
					"errors": null,
					"fields": {
						"name": {
							"is_null": false,
							"is_bound": true,
							"value": "test",
							"errors": null
						}
					}
				}`,
			},
			{
				data: `{"name": "nope"}`,
				result: `{
					"is_valid": false,
					"errors": null,
					"fields": {
						"name": {
							"is_null": false,
							"is_bound": true,
							"value": "nope",
							"errors": [
								"forbidden value"
							]
						}
					}
				}`,
			},
		}

		for i, test := range tests {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				f := newCustomValidationForm()
				require.False(t, f.IsBound())

				r := strings.NewReader(test.data)
				forms.UnmarshalJSON(f, r)
				testForm(t, test, f)
			})
		}
	})
}

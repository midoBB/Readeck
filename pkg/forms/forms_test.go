// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package forms_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
	jsonassert.New(t).Assertf(string(data), test.result) //nolint:govet
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

func TestMultipart(t *testing.T) {
	body := `
--foo
Content-Disposition: form-data; name="data"; filename="blob"
Content-Type: application/octet-stream

test value
abc
--foo
Content-Disposition: form-data; name="test"

123
--foo--
		`

	tests := []struct {
		formFn func() *forms.Form
		testFn func(*require.Assertions, *forms.Form)
	}{
		{
			func() *forms.Form {
				return forms.Must(
					forms.NewIntegerField("test"),
					forms.NewFileField("data", forms.Required),
				)
			},
			func(assert *require.Assertions, f *forms.Form) {
				assert.True(f.IsBound())
				assert.True(f.IsValid())
				assert.Equal(123, f.Get("test").Value())

				r, err := f.Get("data").Field.(*forms.FileField).Open()
				assert.NoError(err)
				data, err := io.ReadAll(r)
				assert.NoError(err)
				assert.Equal([]byte("test value\nabc"), data)
			},
		},
		{
			func() *forms.Form {
				return forms.Must(
					forms.NewFileField("data2", forms.Required),
				)
			},
			func(assert *require.Assertions, f *forms.Form) {
				assert.True(f.IsBound())
				assert.False(f.IsValid())
				assert.EqualError(errors.Join(f.Get("data2").Errors), "field is required")
			},
		},
		{
			func() *forms.Form {
				return forms.Must(
					forms.NewFileField("data2"),
				)
			},
			func(assert *require.Assertions, f *forms.Form) {
				assert.True(f.IsBound())
				assert.True(f.IsValid())
				assert.True(f.Get("data2").IsNil())
			},
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			req := &http.Request{
				Method: http.MethodPost,
				Header: http.Header{
					"Content-Type": []string{"multipart/form-data; boundary=foo"},
				},
			}
			req.Body = io.NopCloser(strings.NewReader(body))
			form := test.formFn()
			forms.BindMultipart(form, req)

			assert := require.New(t)
			test.testFn(assert, form)
		})
	}
}

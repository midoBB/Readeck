// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// Package forms provides helpers and functions to create and validate forms.
package forms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

// ErrUnexpected is an error that can be used during custom validation
// or form actions.
var ErrUnexpected = Gettext("an unexpected error has occurred")

// Errors is an error list.
type Errors []error

type ctxTranslatorKey struct{}

// Binder describes the basic needed method for a form that can be bound
// from JSON or URL values.
type Binder interface {
	Fields() []*FormField
	Get(string) *FormField
	AddErrors(string, ...error)
	IsBound() bool
	Bind()
	IsValid() bool
}

// Localized describes a form that can receive a translator
// so messages and errors can be translated.
type Localized interface {
	SetLocale(Translator)
}

// AnyBinder describes a form that provides its own binding method for unknown content-type.
// One can use it to bind from multipart data, plain text, etc.
type AnyBinder interface {
	BindAny(contentType string, r *http.Request)
}

// Validator describes a form that implements a custom validation.
type Validator interface {
	Validate()
}

// Form is a list of fields.
type Form struct {
	isBound  bool
	fields   []Field
	fieldMap map[string]int
	context  context.Context
	errors   map[string]Errors
}

// FormField is a Field with its errors.
type FormField struct {
	Field
	Errors Errors
}

// New returns a new Form instance.
func New(fields ...Field) (*Form, error) {
	f := &Form{
		fields:   make([]Field, len(fields)),
		fieldMap: map[string]int{},
		errors:   map[string]Errors{},
		context:  context.Background(),
	}

	for i, field := range fields {
		name := field.Name()
		if name == "" {
			return nil, fmt.Errorf("Unamed field")
		}
		if _, exists := f.fieldMap[field.Name()]; exists {
			return nil, fmt.Errorf(`Field "%s" already defined`, field.Name())
		}
		f.fields[i] = field
		f.fieldMap[field.Name()] = i
	}

	return f, nil
}

// Must returns a new Form instance and panics if there was any error.
func Must(fields ...Field) *Form {
	f, err := New(fields...)
	if err != nil {
		panic(err)
	}
	return f
}

// SetLocale sets the form current locale.
func (f *Form) SetLocale(tr Translator) {
	if tr != nil {
		f.context = context.WithValue(f.context, ctxTranslatorKey{}, tr)
	}
}

// Fields returns the form's field list.
func (f *Form) Fields() []*FormField {
	res := make([]*FormField, len(f.fields))
	for i, field := range f.fields {
		res[i] = &FormField{Field: field, Errors: f.errors[field.Name()]}
	}
	return res
}

// Get returns a field by its name, or nil when it doesn't exist.
func (f *Form) Get(name string) *FormField {
	i, ok := f.fieldMap[name]
	if !ok {
		return nil
	}
	return &FormField{
		Field:  f.fields[i],
		Errors: f.errors[f.fields[i].Name()],
	}
}

// AddErrors adds an error to the form.
func (f *Form) AddErrors(name string, errorList ...error) {
	if len(errorList) == 0 {
		return
	}
	if _, ok := f.errors[name]; !ok {
		f.errors[name] = []error{}
	}

	tr, _ := f.context.Value(ctxTranslatorKey{}).(Translator)
	for _, err := range errorList {
		f.errors[name] = append(f.errors[name], localizedError{err: err, tr: tr})
	}
}

// IsValid returns true if the form has no error.
func (f *Form) IsValid() bool {
	return len(f.errors) == 0
}

// Errors returns the form's non-field error list.
func (f *Form) Errors() Errors {
	return f.errors[""]
}

// AllErrors returns the form's error map (including field errors).
func (f *Form) AllErrors() map[string]Errors {
	return f.errors
}

// IsBound returns true if the form has been bound to input data.
func (f *Form) IsBound() bool {
	return f.isBound
}

// Bind set the form as bound. As it's called before validation, it can be
// used to set default values, when needs be.
func (f *Form) Bind() {
	f.isBound = true
}

// Context returns the form current context.
func (f *Form) Context() context.Context {
	if f.context != nil {
		return f.context
	}
	return context.Background()
}

// SetContext set the new form's context.
func (f *Form) SetContext(ctx context.Context) *Form {
	if ctx == nil {
		panic("nil context")
	}
	f.context = ctx
	return f
}

// FieldMap returns a map of all fields.
func (f *Form) FieldMap() map[string]*FormField {
	res := map[string]*FormField{}
	for _, field := range f.Fields() {
		res[field.Name()] = field
	}
	return res
}

// MarshalJSON returns the JSON serialization of a form.
func (f *Form) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		IsValid bool                  `json:"is_valid"`
		Errors  Errors                `json:"errors"`
		Fields  map[string]*FormField `json:"fields"`
	}{
		IsValid: f.IsValid(),
		Errors:  f.Errors(),
		Fields:  f.FieldMap(),
	})
}

// MarshalJSON returns the JSON serialization of a field.
func (f *FormField) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		IsNil   bool        `json:"is_null"`
		IsBound bool        `json:"is_bound"`
		Value   interface{} `json:"value"`
		Errors  Errors      `json:"errors"`
	}{
		IsNil:   f.IsNil(),
		IsBound: f.IsBound(),
		Value:   f.Value(),
		Errors:  f.Errors,
	})
}

// Choices returns the choice list of a field, if the wrapped field
// implements the FieldChoices interface.
func (f *FormField) Choices() Choices {
	if v, ok := f.Field.(FieldChoices); ok {
		return v.Choices()
	}
	return nil
}

// MarshalJSON returns the JSON serialization of an error list.
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

func (e Errors) Error() string {
	if len(e) == 0 {
		return ""
	}

	res := make([]string, len(e))
	for i, x := range e {
		res[i] = x.Error()
	}
	return strings.Join(res, ", ")
}

// UnmarshalJSON decodes JSON values into the form.
// It does so by decoding first the input value into a map
// of raw values. Then each register field that's present in
// the resulting map is decoded.
func UnmarshalJSON(f Binder, r io.Reader) {
	if f.IsBound() {
		f.AddErrors("", errors.New("Form is already bound"))
		return
	}

	f.Bind()

	values := map[string]json.RawMessage{}
	if err := json.NewDecoder(r).Decode(&values); err != nil {
		f.AddErrors("", errors.New("Invalid input data"))
	}
	for _, field := range f.Fields() {
		data, exists := values[field.Name()]
		if !exists {
			continue
		}
		err := field.UnmarshalJSON(data)
		if err != nil {
			f.AddErrors(field.Name(), err)
		}
	}
	Validate(f)
}

// UnmarshalValues decodes url encoded values into the form.
// It decodes every item into a map of values that are passed
// to each field's UnmarshalText method.
func UnmarshalValues(f Binder, values url.Values) {
	if f.IsBound() {
		f.AddErrors("", errors.New("Form is already bound"))
		return
	}

	f.Bind()

	for _, field := range f.Fields() {

		v, exists := values[field.Name()]
		if !exists {
			continue
		}

		// Always empty the field before proceeding
		field.Set(nil)
		for _, x := range v {
			if err := field.UnmarshalText([]byte(x)); err != nil {
				f.AddErrors(field.Name(), err)
			}
		}
	}
	Validate(f)
}

// Bind loads and validates the data using the method tied
// to the request's content-type header.
func Bind(f Binder, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("content-type"))
	if err != nil {
		f.AddErrors("", errors.New("Invalid content-type"))
		return
	}

	switch mediaType {
	case "application/json", "text/json":
		defer r.Body.Close() //nolint:errcheck
		UnmarshalJSON(f, r.Body)
	case "application/x-www-form-urlencoded":
		if err := r.ParseForm(); err != nil {
			f.AddErrors("", errors.New("Invalid input"))
		}
		UnmarshalValues(f, r.PostForm)
	default:
		if f, ok := f.(AnyBinder); ok {
			f.BindAny(mediaType, r)
			break
		}
		f.AddErrors("", errors.New("Unknown content-type"))
	}
}

// BindMultipart loads and validates the data from the request's body,
// including "multipart/form-data".
// It defaults to calling [Bind] for other content types.
func BindMultipart(f Binder, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("content-type"))
	if err != nil {
		f.AddErrors("", errors.New("Invalid content-type"))
		return
	}

	if mediaType != "multipart/form-data" {
		Bind(f, r)
		return
	}

	if f.IsBound() {
		f.AddErrors("", errors.New("Form is already bound"))
		return
	}

	// Always finish with loading the regular values
	defer func() {
		// Bind regular fields (this validates the form as well)
		UnmarshalValues(f, r.PostForm)
	}()

	if r.MultipartForm == nil {
		if err := r.ParseMultipartForm(16 << 20); err != nil {
			f.AddErrors("", errors.New("Error loading data"))
			return
		}
	}

	// Bind the file fields
	for name, headers := range r.MultipartForm.File {
		if len(headers) == 0 {
			continue
		}
		field := f.Get(name)
		if field == nil {
			continue
		}
		if _, ok := field.Field.(*FileField); ok {
			if !field.Set(headers[len(headers)-1]) {
				f.AddErrors(field.Name(), ErrInvalidValue)
			}
		}
	}
}

// Validate performs all the fields validation and, if the form is a
// FormValidator, the form.Validate() method.
func Validate(input interface{}) {
	// Validate the fields
	if f, ok := input.(Binder); ok {
		for _, field := range f.Fields() {
			errors := ValidateField(field, field.Validators()...)
			if len(errors) > 0 {
				f.AddErrors(field.Name(), errors...)
			}
		}
	}
	// Validate the form itself
	if f, ok := input.(Validator); ok {
		f.Validate()
	}
}

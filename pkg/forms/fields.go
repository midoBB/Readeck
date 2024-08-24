// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package forms

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
)

// NilValue is a text null value. In an URL of form value, it would
// be a field with %00 value. (name=%00).
var NilValue = []byte{0}

// Field describes a form field.
type Field interface {
	Name() string
	IsBound() bool
	IsNil() bool
	Set(value interface{}) bool
	UnmarshalJSON([]byte) error
	UnmarshalText([]byte) error
	Value() interface{}
	String() string
	Validators() []FieldValidator
	SetValidators(...FieldValidator)
}

// FieldChoices describes a field that can return a list
// of possible values.
type FieldChoices interface {
	Choices() Choices
}

var (
	// ErrInvalidType is the error for invalid type.
	ErrInvalidType = errors.New("invalid type")
	// ErrInvalidValue is the for invalid value.
	ErrInvalidValue = errors.New("invalid value")
)

// fieldConstructor is a function returning a field with a given name.
type fieldConstructor func(string) Field

// fieldConverter is a function that returns a value from a list of fields.
type fieldConverter func([]Field) interface{}

// BaseField is a basic field that holds the field's name and its
// bound and nil state.
type BaseField struct {
	name       string
	isBound    bool
	isNil      bool
	validators []FieldValidator
}

// NewBaseField returns a new BaseField instance that is considered
// null. Until it's set or bound, it will stay that way.
func NewBaseField(name string, validators ...FieldValidator) *BaseField {
	return &BaseField{
		name:       name,
		isNil:      true,
		isBound:    false,
		validators: validators,
	}
}

// SetNil sets the nil state of the field, based on the passed value.
func (f *BaseField) SetNil(value interface{}) {
	f.isNil = value == nil
}

// SetBind marks the field as bound.
func (f *BaseField) SetBind() {
	f.isBound = true
}

// Name returns the field's name.
func (f *BaseField) Name() string {
	return f.name
}

// IsBound returns true if the field is bound.
func (f *BaseField) IsBound() bool {
	return f.isBound
}

// IsNil returns true if the field's value is null.
func (f *BaseField) IsNil() bool {
	return f.isNil
}

// Validators returns the field's validator list.
func (f *BaseField) Validators() []FieldValidator {
	return f.validators
}

// SetValidators sets new validators for a field.
func (f *BaseField) SetValidators(validators ...FieldValidator) {
	f.validators = validators
}

/* Text field
   --------------------------------------------------------------- */

// TextField is a field with a string value.
type TextField struct {
	*BaseField
	value string
}

// NewTextField returns a TextField instance.
func NewTextField(name string, validators ...FieldValidator) *TextField {
	return &TextField{BaseField: NewBaseField(name, validators...)}
}

// Set sets the field's value.
func (f *TextField) Set(value interface{}) bool {
	f.BaseField.SetNil(value)
	if f.IsNil() {
		f.value = ""
		return true
	}
	var ok bool
	f.value, ok = value.(string)
	return ok
}

// UnmarshalJSON decodes the input value into a string or a nil value.
func (f *TextField) UnmarshalJSON(data []byte) error {
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	f.BaseField.SetNil(value)
	f.BaseField.SetBind()
	if ok := f.Set(value); !ok {
		return ErrInvalidType
	}
	return nil
}

// UnmarshalText decodes the input text value into a string or nil value.
func (f *TextField) UnmarshalText(text []byte) error {
	if bytes.Equal(text, NilValue) {
		f.BaseField.SetBind()
		f.Set(nil)
		return nil
	}

	f.BaseField.SetBind()
	f.Set(string(text))

	return nil
}

// Value returns the field's actuall value.
func (f *TextField) Value() interface{} {
	if f.IsNil() {
		return nil
	}
	return f.value
}

// String returns the field's string value.
func (f *TextField) String() string {
	return f.value
}

/* Boolean field
   --------------------------------------------------------------- */

// BooleanField is a boolean field (true/false).
type BooleanField struct {
	*BaseField
	value bool
}

// NewBooleanField return a BooleanField instance.
func NewBooleanField(name string, validators ...FieldValidator) *BooleanField {
	return &BooleanField{BaseField: NewBaseField(name, validators...)}
}

// Set sets the field's value.
func (f *BooleanField) Set(value interface{}) bool {
	f.BaseField.SetNil(value)
	if f.IsNil() {
		f.value = false
		return true
	}
	var ok bool
	f.value, ok = value.(bool)
	return ok
}

// UnmarshalJSON decodes the input value into a string or a nil value.
func (f *BooleanField) UnmarshalJSON(data []byte) error {
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	f.BaseField.SetNil(value)
	f.BaseField.SetBind()
	if ok := f.Set(value); !ok {
		return ErrInvalidType
	}
	return nil
}

// UnmarshalText decodes the input text value into a string or nil value.
func (f *BooleanField) UnmarshalText(text []byte) error {
	f.BaseField.SetBind()
	if len(text) == 0 {
		f.Set(nil)
		return nil
	}

	var value bool
	var err error
	t := string(text)

	switch t {
	case "":
		value = false
	case "on":
		value = true
	default:
		value, err = strconv.ParseBool(t)
	}

	if err != nil {
		f.Set(false)
		return ErrInvalidValue
	}

	f.Set(value)
	return nil
}

// Value returns the field's actuall value.
func (f *BooleanField) Value() interface{} {
	if f.IsNil() {
		return nil
	}
	return f.value
}

// String returns the field's string value.
func (f *BooleanField) String() string {
	if f.IsNil() {
		return ""
	}
	if f.value {
		return "true"
	}
	return "false"
}

/* Integer field
   --------------------------------------------------------------- */

// IntegerField is an integer field.
type IntegerField struct {
	*BaseField
	value int
}

// NewIntegerField returns a IntegerField instance.
func NewIntegerField(name string, validators ...FieldValidator) *IntegerField {
	return &IntegerField{
		BaseField: NewBaseField(name, validators...),
		value:     0,
	}
}

// Set sets the field's value.
func (f *IntegerField) Set(value interface{}) bool {
	f.BaseField.SetNil(value)
	if f.IsNil() {
		f.value = 0
		return true
	}

	switch v := value.(type) {
	case int:
		f.value = v
		return true
	case float64:
		if v == float64(int(v)) {
			f.value = int(v)
			return true
		}
	}

	f.SetNil(nil)
	return false
}

// UnmarshalJSON decodes the input value into a string or a nil value.
func (f *IntegerField) UnmarshalJSON(data []byte) error {
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	f.BaseField.SetNil(value)
	f.BaseField.SetBind()
	if value == nil {
		return nil
	}

	if ok := f.Set(value); !ok {
		return ErrInvalidType
	}
	return nil
}

// UnmarshalText decodes the input text value into a string or nil value.
func (f *IntegerField) UnmarshalText(text []byte) error {
	if bytes.Equal(text, NilValue) {
		f.BaseField.SetBind()
		f.Set(nil)
		return nil
	}

	f.BaseField.SetBind()

	v, err := strconv.ParseInt(string(text), 10, 0)
	if err != nil {
		f.Set(nil)
		return ErrInvalidType
	}

	if ok := f.Set(int(v)); !ok {
		return ErrInvalidType
	}

	return nil
}

// Value returns the field's actuall value.
func (f *IntegerField) Value() interface{} {
	if f.IsNil() {
		return nil
	}
	return f.value
}

// String returns the field's string value.
func (f *IntegerField) String() string {
	if f.IsNil() {
		return ""
	}
	return fmt.Sprint(f.value)
}

/* Time field
   --------------------------------------------------------------- */

// DatetimeField is a datetime field.
type DatetimeField struct {
	*BaseField
	value time.Time
}

// NewDatetimeField return a DatetimeField instance.
func NewDatetimeField(name string, validators ...FieldValidator) *DatetimeField {
	return &DatetimeField{
		BaseField: NewBaseField(name, validators...),
		value:     time.Time{},
	}
}

// Set sets the field's value.
func (f *DatetimeField) Set(value interface{}) bool {
	var t *time.Time
	switch v := value.(type) {
	case nil:
		t = nil
	case time.Time:
		t = &v
	case *time.Time:
		t = v
	default:
		return false
	}

	if t == nil || t.IsZero() {
		f.SetNil(nil)
		return true
	}
	f.SetNil(struct{}{})
	f.value = *t

	return true
}

// UnmarshalJSON decodes the input value into a string or a nil value.
func (f *DatetimeField) UnmarshalJSON(data []byte) error {
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	f.BaseField.SetNil(value)
	f.BaseField.SetBind()
	switch v := value.(type) {
	case string:
		return f.decodeTime(v)
	case nil:
		f.Set(nil)
		return nil
	}
	return ErrInvalidType
}

// UnmarshalText decodes the input text value into a string or nil value.
func (f *DatetimeField) UnmarshalText(text []byte) error {
	if bytes.Equal(text, NilValue) {
		f.BaseField.SetBind()
		f.Set(nil)
		return nil
	}

	f.BaseField.SetBind()
	return f.decodeTime(string(text))
}

func (f *DatetimeField) decodeTime(text string) error {
	if text == "" {
		f.SetNil(nil)
		return nil
	}
	v, err := dateparse.ParseAny(text)
	if err != nil {
		f.Set(time.Time{})
		return errors.New("invalid datetime format")
	}
	if !f.Set(v) {
		return ErrInvalidType
	}
	return nil
}

// Value returns the field's actuall value.
func (f *DatetimeField) Value() interface{} {
	if f.IsNil() {
		return nil
	}
	return f.value
}

// String returns the field's string value.
func (f *DatetimeField) String() string {
	if f.IsNil() {
		return ""
	}

	return f.value.Format("2006-01-02")
}

/* File field
   --------------------------------------------------------------- */

// FileOpener describes an opener interface. Its [Open] function must return an [io.ReadCloser].
type FileOpener interface {
	Open() (io.ReadCloser, error)
}

// multipartFileOpener is a [FileOpener] implementation wrapping [multipart.FileHeader].
type multipartFileOpener struct {
	*multipart.FileHeader
}

// Open implements [FileOpener].
func (o *multipartFileOpener) Open() (io.ReadCloser, error) {
	return o.FileHeader.Open()
}

// readerOpener is a [FileOpener] implementation wrapping an [io.Reader].
type readerOpener struct {
	io.Reader
}

// Open implements [FileOpener].
func (o *readerOpener) Open() (io.ReadCloser, error) {
	if r, ok := o.Reader.(io.ReadCloser); ok {
		return r, nil
	}

	return io.NopCloser(o.Reader), nil
}

// FileField is a file field. It receives a [FileOpener] as a value and thus, wont't be deserialize
// by classic unmarshal functions but it can be open with its [FileField.Open] function.
type FileField struct {
	*BaseField
	value FileOpener
}

// NewFileField returns a new [FileField] instance.
func NewFileField(name string, validators ...FieldValidator) *FileField {
	return &FileField{
		BaseField: NewBaseField(name, validators...),
	}
}

// Set sets the field value. It can receive a [*multipart.FileHeader] or a [io.Reader].
// The value is set to a [FileOpener].
func (f *FileField) Set(value interface{}) bool {
	f.BaseField.SetNil(value)
	if f.IsNil() {
		f.value = nil
		f.SetBind()
		return true
	}

	switch t := value.(type) {
	case *multipart.FileHeader:
		f.value = &multipartFileOpener{t}
	case io.Reader:
		f.value = &readerOpener{t}
	}

	if f.value != nil {
		f.SetBind()
		return true
	}

	f.SetNil(nil)
	return false
}

// Value returns nil if the value's null or "(file reader)" when there's a [readerOpener] attached.
func (f *FileField) Value() interface{} {
	if f.value == nil {
		return nil
	}
	return "(file reader)"
}

// String always returns an empty string.
func (f *FileField) String() string {
	if f.IsNil() {
		return ""
	}
	return "\u0000"
}

// UnmarshalJSON is not implemented.
func (f *FileField) UnmarshalJSON(_ []byte) error {
	return nil
}

// UnmarshalText is not implemented.
func (f *FileField) UnmarshalText(_ []byte) error {
	return nil
}

// Open implements [FileOpener] on the field.
func (f *FileField) Open() (io.ReadCloser, error) {
	if f.value == nil {
		return nil, ErrInvalidValue
	}
	return f.value.Open()
}

// Choices is a list of valid values (value and name).
type Choices [][2]string

// ChoiceField is a text field with a limited possible values.
type ChoiceField struct {
	Field
	choices Choices
}

// NewChoiceField returns a ChoiceField instance.
func NewChoiceField(name string, choices Choices, validators ...FieldValidator) *ChoiceField {
	f := &ChoiceField{
		choices: choices,
	}
	f.Field = NewTextField(name, append([]FieldValidator{f.Validate}, validators...)...)
	return f
}

// Choices returns the list of possible values.
func (f *ChoiceField) Choices() Choices {
	return f.choices
}

// Validate performs the field's validation.
func (f *ChoiceField) Validate(_ Field) error {
	choices := make([]string, len(f.choices))
	for i, v := range f.choices {
		choices[i] = v[0]
		if v[0] == f.String() {
			return nil
		}
	}

	return fmt.Errorf("must be one of %s", strings.Join(choices, ", "))
}

// ListField is a field that implement decoding and encoding of
// values in lists.
type ListField struct {
	*BaseField
	constructor fieldConstructor
	converter   fieldConverter
	value       []Field
	choices     Choices
}

// DefaultListConverter is a default fieldConverter that
// simply returns a list of interface{} items.
func DefaultListConverter(values []Field) interface{} {
	res := make([]interface{}, len(values))
	for i, x := range values {
		res[i] = x.Value()
	}
	return res
}

// NewListField return a ListField instance. It needs a constructor and a converter.
// Validators at this stage are only applied to the whole field.
// If you need a validator for each received value, it must come with the field returned
// by a custom constructor.
func NewListField(name string, constructor fieldConstructor, converter fieldConverter, validators ...FieldValidator) Field {
	f := &ListField{
		constructor: constructor,
		converter:   converter,
		value:       []Field{},
	}
	f.BaseField = NewBaseField(name, append([]FieldValidator{f.Validate}, validators...)...)
	return f
}

// SetChoices sets the field choice list.
func (f *ListField) SetChoices(choices Choices) {
	f.choices = choices
}

// Choices returns the field choice list.
func (f *ListField) Choices() Choices {
	return f.choices
}

// InChoices returns true if value is in the field choice list.
func (f *ListField) InChoices(value string) bool {
	for _, x := range f.value {
		if value == x.String() {
			return true
		}
	}
	return false
}

// Validate performs the field validation.
func (f *ListField) Validate(_ Field) error {
	if len(f.choices) == 0 {
		return nil
	}

	choices := map[string]struct{}{}
	for _, c := range f.choices {
		choices[c[0]] = struct{}{}
	}

	for _, v := range f.value {
		if _, ok := choices[v.String()]; !ok {
			return fmt.Errorf("%s is not a valid value", v.String())
		}
	}

	return nil
}

// Set sets the field's value.
func (f *ListField) Set(value interface{}) bool {
	f.BaseField.SetNil(value)
	if f.IsNil() {
		f.value = []Field{}
		return true
	}

	var values []interface{}

	// Some shortcuts for common type.
	// This let us set concrete values without having to
	// convert it first.
	switch v := value.(type) {
	case []interface{}:
		values = v
	case []bool:
		values = make([]interface{}, len(v))
		for i, x := range v {
			values[i] = x
		}
	case []float64:
		values = make([]interface{}, len(v))
		for i, x := range v {
			values[i] = x
		}
	case []int:
		values = make([]interface{}, len(v))
		for i, x := range v {
			values[i] = x
		}
	case []string:
		values = make([]interface{}, len(v))
		for i, x := range v {
			values[i] = x
		}
	}

	res := false
	defer func() {
		if !res {
			f.SetNil(nil)
			f.value = []Field{}
		}
	}()

	if values == nil {
		return res
	}

	f.value = make([]Field, len(values))
	for i, x := range values {
		f.value[i] = f.constructor(strconv.Itoa(i))
		if !f.value[i].Set(x) {
			return res
		}
	}

	res = true
	return res
}

// UnmarshalJSON decodes the input value into a string or a nil value.
func (f *ListField) UnmarshalJSON(data []byte) error {
	f.BaseField.SetBind()

	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	f.BaseField.SetBind()
	if value == nil {
		f.BaseField.SetNil(nil)
		return nil
	}

	v, ok := value.([]interface{})
	if !ok {
		return ErrInvalidType
	}

	// To match the behavior of URL encoded values,
	// an empty list leads to nil.
	if len(v) == 0 {
		f.BaseField.SetNil(nil)
		return nil
	}

	// Discard null values in the list
	values := []interface{}{}
	for _, x := range v {
		if x == nil {
			continue
		}
		values = append(values, x)
	}

	if ok = f.Set(values); !ok {
		return ErrInvalidType
	}

	// We must now validate each field
	for _, field := range f.value {
		if e := ValidateField(field, field.Validators()...); len(e) > 0 {
			return e
		}
	}

	return nil
}

// UnmarshalText decodes the input text value and appends it to the
// values when it's valid.
func (f *ListField) UnmarshalText(text []byte) error {
	f.BaseField.SetBind()

	if len(text) == 0 {
		return nil
	}

	var err error
	idx := len(f.value)

	// Initialize the field
	field := f.constructor(strconv.Itoa(idx))
	err = field.UnmarshalText(text)
	if err != nil {
		return err
	}

	// No need to go further with a nil value
	if field.IsNil() {
		return nil
	}

	// Validation will add new errors to the field but
	// could also change its value, so we don't return
	// there and will append the value in any case.
	if e := ValidateField(field, field.Validators()...); len(e) > 0 {
		err = e
	}

	// We can now safely append the value in the field.
	f.SetNil(1)
	f.value = append(f.value, field)
	return err
}

// Value returns the field's actuall value.
func (f *ListField) Value() interface{} {
	if f.value == nil || f.IsNil() || f.value == nil {
		return nil
	}

	return f.converter(f.value)
}

// String returns the field's string value.
func (f *ListField) String() string {
	if f.IsNil() {
		return ""
	}
	b, _ := json.Marshal(f.Value())
	return string(b)
}

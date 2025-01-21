// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package forms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrInvalidValue is the error for invalid value.
var ErrInvalidValue = errors.New("invalid value")

// FieldFlags is a field's flag list.
type FieldFlags int8

const (
	// ValidatedField indicates a field has been validated.
	ValidatedField FieldFlags = 1 << iota
)

type marshalledField struct {
	IsNil   bool   `json:"is_null"`
	IsBound bool   `json:"is_bound"`
	Value   any    `json:"value"`
	Errors  Errors `json:"errors"`
}

// Field describes a form field.
type Field interface {
	fmt.Stringer
	json.Unmarshaler
	UnmarshalValues([]string) error
	Contexter

	Name() string
	IsBound() bool
	IsEmpty() bool
	IsNil() bool
	Value() any
	Set(value any)

	IsValid() bool

	AddErrors(...error)
	Errors() Errors
}

// TypedField exposes a field's typed value.
type TypedField[T any] interface {
	V() T
}

// GetForm returns the form that's attached to a field.
// It's only available when the field has been added to a form
// using [New] of [Must].
func GetForm(f Field) Binder {
	if form, ok := f.Context().Value(ctxFormKey).(Binder); ok {
		return form
	}
	return nil
}

// BaseField is a generic base field that holds a value of the
// given type and implements [Field].
// It's the common building block for a specialized field.
type BaseField[T any] struct {
	flags   FieldFlags
	value   Value[T]
	name    string
	context context.Context
	decoder Decoder[T]
	errors  Errors
}

// NewBaseField returns a new BaseField instance that is considered
// null. Until it's set or bound, it will stay that way.
func NewBaseField[T any](name string, decoder Decoder[T], options ...any) *BaseField[T] {
	res := &BaseField[T]{
		name: name,
		value: Value[T]{
			F: IsNil | IsEmpty,
		},
		decoder: decoder,
		context: context.Background(),
	}

	applyFieldOptions[T](res, options...)
	return res
}

// Name returns the field's name.
func (f BaseField[T]) Name() string {
	return f.name
}

// Context returns the field's context.
func (f BaseField[T]) Context() context.Context {
	return f.context
}

// SetContext sets the field's context.
func (f *BaseField[T]) SetContext(ctx context.Context) {
	f.context = ctx
}

// IsBound returns true if the field is bound.
func (f BaseField[T]) IsBound() bool {
	return f.value.F.IsBound()
}

// IsNil returns true if the field's value is null.
func (f BaseField[T]) IsNil() bool {
	return f.value.F.IsNil()
}

// IsEmpty returns true if the field's value is empty.
func (f BaseField[T]) IsEmpty() bool {
	return f.value.F.IsEmpty()
}

// Value implements the [Field] interface and returns the field's value
// with a type "any". It returns nil when the field is nil.
func (f BaseField[T]) Value() any {
	if f.value.F.IsNil() {
		return nil
	}
	return f.value.V
}

// V implement the [TypedField] interface and returns the field's value
// with its intended type.
func (f BaseField[T]) V() T {
	return f.value.V
}

// IsValid returns true when the field is valid.
// It performs the validation on its first call.
func (f *BaseField[T]) IsValid() bool {
	if f.flags&ValidatedField > 0 {
		return len(f.errors) == 0
	}

	defer func() {
		f.flags |= ValidatedField
	}()
	if len(f.errors) == 0 {
		f.AddErrors(ApplyValidators[T](f, f.value.V, GetValidators(f)...)...)
	}
	return len(f.errors) == 0
}

// Set sets the field's value. The value v can be a pointer or a concrete
// value. A nil value, sets the field as nil with an zeroed value.
func (f *BaseField[T]) Set(value any) {
	if x, ok := value.(*T); ok && x != nil {
		f.value = f.decoder.DecodeAny(*x)
		return
	}

	// Then try a concrete value.
	f.value = f.decoder.DecodeAny(value)
}

// UnmarshalJSON implements [json.Unmarshaler] for the field.
// In case of decoding error, the value is zeroed and an [ErrInvalidValue] is returned.
func (f *BaseField[T]) UnmarshalJSON(data []byte) error {
	var errs Errors
	var decoded any

	defer func() {
		f.postBinding(errs)
	}()

	if err := json.Unmarshal(data, &decoded); err != nil {
		f.Set(nil)
		errs = Errors{ErrInvalidValue}
		return errs[0]
	}

	if decoded == nil {
		f.Set(nil)
		return nil
	}

	f.value = f.decoder.DecodeAny(
		f.preBinding(decoded),
	)

	if !f.value.F.IsNil() && !f.value.F.IsOk() {
		errs = Errors{ErrInvalidValue}
		return errs[0]
	}

	return nil
}

// UnmarshalValues decodes a list of values using the provided
// field's [Decoder].
// In this case, it only decodes the first value.
func (f *BaseField[T]) UnmarshalValues(values []string) error {
	var errs Errors
	defer func() {
		f.postBinding(errs)
	}()

	if len(values) == 0 {
		return nil
	}

	if values[0] == nilText {
		f.Set(nil)
		return nil
	}

	x := f.preBinding(values[0])
	if x, ok := x.(string); ok {
		f.value = f.decoder.DecodeText(x)
	} else {
		f.value.F |= IsNil
	}

	if !f.value.F.IsNil() && !f.value.F.IsOk() {
		errs = Errors{ErrInvalidValue}
		return errs[0]
	}

	return nil
}

func (f *BaseField[T]) preBinding(data any) any {
	for _, v := range GetCleaners(f) {
		data = v.Clean(data)
	}

	return data
}

func (f *BaseField[T]) postBinding(err Errors) {
	if err != nil {
		f.AddErrors(err...)
		return
	}
	f.value.F |= IsOk | IsBound
}

// Errors return the field's [Errors].
func (f *BaseField[T]) Errors() Errors {
	return f.errors
}

// AddErrors add errors to the field.
func (f *BaseField[T]) AddErrors(errs ...error) {
	if len(errs) == 0 {
		return
	}

	tr := GetTranslator(f.context)
	for _, err := range unwrapErrors(errs...) {
		if err == nil {
			continue
		}

		// an error that implements Err() (like [fatalError]) is unwrapped
		// so we can access its real error with localization, if any.
		if x, ok := err.(interface{ Err() error }); ok {
			err = x.Err()
		}

		if _, ok := err.(localizedError); !ok {
			err = localizedError{err: err, tr: tr}
		}

		f.errors = append(f.errors, err)
	}

	if len(f.errors) > 0 {
		f.value.F &^= IsOk
	}
}

// String implement [fmt.Stringer].
func (f *BaseField[T]) String() string {
	if f.IsNil() || f.IsEmpty() {
		return ""
	}

	if d, ok := f.decoder.(valueStringer[T]); ok {
		return d.ValueString(f.value.V)
	}

	return fmt.Sprintf("%v", f.value.V)
}

// MarshalJSON implements [json.Marshaler] for the field.
func (f BaseField[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(marshalledField{
		IsNil:   f.IsNil(),
		IsBound: f.IsBound(),
		Value:   f.V(),
		Errors:  f.Errors(),
	})
}

/* Text field
   --------------------------------------------------------------- */

// TextField is a field that holds a [string] value.
type TextField struct {
	*BaseField[string]
}

// NewTextField return a new [TextField] instance.
func NewTextField(name string, options ...any) *TextField {
	return &TextField{
		NewBaseField(name, DecodeString, options...),
	}
}

func (f TextField) String() string {
	return f.value.V
}

// Choices returns the field's [ValueChoices].
func (f TextField) Choices() ValueChoices[string] {
	return GetChoices[string](f)
}

/* Boolean field
   --------------------------------------------------------------- */

// BooleanField is a field that holds a [bool] value.
type BooleanField struct {
	*BaseField[bool]
}

// NewBooleanField return a new [BooleanField] instance.
func NewBooleanField(name string, options ...any) *BooleanField {
	return &BooleanField{
		NewBaseField(name, DecodeBoolean, options...),
	}
}

/* Integer field
   --------------------------------------------------------------- */

// IntegerField is a field that holds an [int] value.
type IntegerField struct {
	*BaseField[int]
}

// NewIntegerField returns a IntegerField instance.
func NewIntegerField(name string, options ...any) *IntegerField {
	return &IntegerField{
		NewBaseField(name, DecodeInt, options...),
	}
}

// Choices returns the field's [ValueChoices].
func (f IntegerField) Choices() ValueChoices[int] {
	return GetChoices[int](f)
}

/* Datetime field
   --------------------------------------------------------------- */

// DatetimeField is a field that holds a [time.Time] value.
type DatetimeField struct {
	*BaseField[time.Time]
}

// NewDatetimeField return a [DatetimeField] instance.
func NewDatetimeField(name string, options ...any) *DatetimeField {
	return &DatetimeField{
		NewBaseField(name, DecodeTime, options...),
	}
}

/* List field
   --------------------------------------------------------------- */

// ListField is a field wrapping a [BaseField] with a list of given type.
type ListField[T any] struct {
	*BaseField[[]T]
	decoder Decoder[T]
}

// NewListField returns a new instance of [ListField].
func NewListField[T any](name string, decoder Decoder[T], options ...any) *ListField[T] {
	res := &ListField[T]{
		BaseField: NewBaseField[[]T](name, nil),
		decoder:   decoder,
	}

	// We don't pass the options to the [BaseField].
	applyFieldOptions[T](res, options...)

	return res
}

// Set sets the field's value. The value v can be a pointer or a concrete
// value. A nil value, sets the field as nil with an zeroed value.
func (f *ListField[T]) Set(value any) {
	var items []T
	f.value.F = ValueFlags(0)

	switch t := value.(type) {
	case *[]T:
		items = *t
	case []T:
		items = t
	case []any:
		for _, v := range t {
			if v, ok := v.(T); ok {
				items = append(items, v)
			}
		}
	default:
		f.value.F |= IsNil | IsEmpty
		return
	}

	f.value = NewValue[[]T]()
	if len(items) == 0 {
		return
	}
	f.value.F &^= IsEmpty
	for _, item := range items {
		v := f.decoder.DecodeAny(item)
		if v.F.IsOk() && !v.F.IsNil() {
			f.value.V = append(f.value.V, v.V)
		}
	}
}

// IsValid returns true when the field is valid.
// It performs the validation on its first call.
func (f *ListField[T]) IsValid() bool {
	if f.flags&ValidatedField > 0 {
		return len(f.errors) == 0
	}

	defer func() {
		f.flags |= ValidatedField
	}()
	if len(f.errors) == 0 {
		f.AddErrors(ApplyValidators[T](f, f.value.V, GetValidators(f)...)...)
	}
	return len(f.errors) == 0
}

// UnmarshalJSON implements [json.Unmarshaler] for the field.
// In case of decoding error, the value is zeroed and an [ErrInvalidValue] is returned.
func (f *ListField[T]) UnmarshalJSON(data []byte) error {
	var errs Errors
	var decoded any

	defer func() {
		if len(f.value.V) == 0 {
			f.value.F |= IsEmpty
		}
		f.postBinding(errs)
	}()

	if err := json.Unmarshal(data, &decoded); err != nil {
		f.Set(nil)
		errs = Errors{ErrInvalidValue}
		return errs[0]
	}

	if decoded == nil {
		f.Set(nil)
		return nil
	}

	items, ok := decoded.([]any)
	if !ok {
		errs = Errors{ErrInvalidValue}
		return errs[0]
	}

	f.value = Value[[]T]{}
	if len(items) == 0 {
		return nil
	}
	for _, item := range items {
		v := f.decoder.DecodeAny(
			f.preBinding(item),
		)
		if v.F.IsNil() {
			continue
		}
		if !v.F.IsOk() {
			f.value.V = []T(nil)
			f.value.F |= IsEmpty
			errs = Errors{ErrInvalidValue}
			return errs[0]
		}

		f.value.V = append(f.value.V, v.V)
	}

	return nil
}

// UnmarshalValues decodes a list of values using the provided
// field's [Decoder] on each value.
func (f *ListField[T]) UnmarshalValues(values []string) error {
	var errs Errors
	defer func() {
		if len(f.value.V) == 0 {
			f.value.F |= IsEmpty
		}
		f.postBinding(errs)
	}()

	if len(values) == 0 {
		f.Set(nil)
		return nil
	}

	f.value = Value[[]T]{}
	for _, s := range values {
		if s == nilText {
			continue
		}
		x := f.preBinding(s)
		if _, ok := x.(string); !ok {
			continue
		}
		v := f.decoder.DecodeText(x.(string))
		if v.F.IsNil() {
			continue
		}
		if !v.F.IsOk() {
			f.value.V = []T(nil)
			f.value.F |= IsEmpty
			errs = Errors{ErrInvalidValue}
			return errs[0]
		}

		f.value.V = append(f.value.V, v.V)
	}

	return nil
}

func (f *ListField[T]) String() string {
	if f.IsNil() || f.IsEmpty() {
		return ""
	}

	if d, ok := f.decoder.(valueStringer[T]); ok {
		res := make([]string, len(f.value.V))
		for i := range f.value.V {
			res[i] = d.ValueString(f.value.V[i])
		}
		return strings.Join(res, ", ")
	}

	return fmt.Sprintf("%v", f.value.V)
}

// TextListField is a field that holds a list of [string] values.
type TextListField struct {
	*ListField[string]
}

// NewTextListField returns a new [TextListField] instance.
func NewTextListField(name string, options ...any) *TextListField {
	return &TextListField{
		NewListField(name, DecodeString, options...),
	}
}

// Choices returns the field's [ValueChoices].
func (f TextListField) Choices() ValueChoices[string] {
	return GetChoices[string](f)
}

// IntegerListField is a field that holds a list of [int] values.
type IntegerListField struct {
	*ListField[int]
}

// NewIntegerListField return new [IntegerListField] instance.
func NewIntegerListField(name string, options ...any) *IntegerListField {
	return &IntegerListField{
		NewListField(name, DecodeInt, options...),
	}
}

// Choices returns the field's [ValueChoices].
func (f IntegerListField) Choices() ValueChoices[int] {
	return GetChoices[int](f)
}

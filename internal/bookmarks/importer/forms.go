// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package importer

import (
	"io"
	"net/http"
	"strings"

	"codeberg.org/readeck/readeck/pkg/forms"
)

// OptionForm is the form containing common import options.
type OptionForm struct {
	*forms.Form
}

// NewOptionForm returns a new OptionForm.
func NewOptionForm() *OptionForm {
	f := &OptionForm{
		forms.Must(
			forms.NewTextField("label", forms.Trim),
			forms.NewBooleanField("ignore_duplicates", forms.RequiredOrNil),
		),
	}
	f.Get("ignore_duplicates").Set(true)
	return f
}

// BindAny implements forms.AnyBinder.
func (f *OptionForm) BindAny(contentType string, r *http.Request) {
	if contentType == "multipart/form-data" {
		err := r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			return
		}
		f.Get("label").Set(r.FormValue("label"))
		for _, v := range r.Form["ignore_duplicates"] {
			_ = f.Get("ignore_duplicates").UnmarshalText([]byte(v))
		}
	}
}

// multipartForm wraps a form and implements forms.AnyBinder
// to handler multipart and text/* content types.
type multipartForm struct {
	*forms.Form
}

func newMultipartForm() *multipartForm {
	return &multipartForm{
		forms.Must(
			newReaderField("data"),
		),
	}
}

func (f *multipartForm) BindAny(contentType string, r *http.Request) {
	// With a multipart content, we'll extract the first "data" field.
	if contentType == "multipart/form-data" {
		found := false
		if r.MultipartForm != nil {
			for _, part := range r.MultipartForm.File["data"] {
				found = true
				reader, err := part.Open()
				if err != nil {
					f.AddErrors("data", err)
					return
				}

				f.Get("data").Set(reader)
				break //nolint
			}
		} else {
			mr, err := r.MultipartReader()
			if err != nil {
				f.AddErrors("data", err)
				return
			}

			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					f.AddErrors("data", err)
					return
				}
				if p.FormName() != "data" {
					continue
				}

				found = true
				f.Get("data").Set(p)
				break
			}
		}

		if !found {
			f.AddErrors("data", forms.Gettext("field is required"))
		}
		return
	}

	// On regular text/* content-type, simply use the request's body.
	if !strings.HasPrefix(contentType, "text/") {
		f.AddErrors("", forms.Gettext("Invalid Content-Type"))
		return
	}

	f.Get("data").Set(r.Body)
}

func (f *multipartForm) dataReader() io.ReadCloser {
	reader := f.Get("data").Field.(*readerField).value
	if reader, ok := reader.(io.ReadCloser); ok {
		return reader
	}

	return io.NopCloser(reader)
}

type readerField struct {
	*forms.BaseField
	value io.Reader
}

func newReaderField(name string, validators ...forms.FieldValidator) *readerField {
	return &readerField{
		BaseField: forms.NewBaseField(name, validators...),
		value:     nil,
	}
}

func (f *readerField) Set(value interface{}) bool {
	f.BaseField.SetNil(value)
	if f.IsNil() {
		f.value = nil
		f.SetBind()
		return true
	}

	if v, ok := value.(io.Reader); ok {
		f.value = v
		f.SetBind()
		return true
	}

	f.SetNil(nil)
	return false
}

func (f *readerField) Value() interface{} {
	if f.value == nil {
		return nil
	}
	return "(io reader)"
}

func (f *readerField) String() string {
	return ""
}

func (f *readerField) UnmarshalJSON(_ []byte) error {
	return nil
}

func (f *readerField) UnmarshalText(_ []byte) error {
	return nil
}

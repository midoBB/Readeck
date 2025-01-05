// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package forms

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/textproto"
	"strings"
)

// FileOpener describes an opener interface. Its [Open] function must return an [io.ReadCloser].
type FileOpener interface {
	Open() (io.ReadCloser, error)
	Filename() string
	Size() int64
	Header() textproto.MIMEHeader
}

// HeaderReader describes an interface than can open multipart content.
type HeaderReader interface {
	UnmarshalFiles([]*multipart.FileHeader) error
}

// MultipartFileOpener is a [FileOpener] implementation wrapping [multipart.FileHeader].
type MultipartFileOpener struct {
	*multipart.FileHeader
}

// Open implements [FileOpener].
func (o *MultipartFileOpener) Open() (io.ReadCloser, error) {
	return o.FileHeader.Open()
}

// Filename implements [FileOpener].
func (o *MultipartFileOpener) Filename() string {
	return o.FileHeader.Filename
}

// Size implements [FileOpener].
func (o *MultipartFileOpener) Size() int64 {
	return o.FileHeader.Size
}

// Header implements [FileOpener].
func (o *MultipartFileOpener) Header() textproto.MIMEHeader {
	return o.FileHeader.Header
}

// MarshalJSON implement [json.Marshaler].
func (o *MultipartFileOpener) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"name":         o.Filename(),
		"size":         o.Size(),
		"content-type": o.Header().Get("content-type"),
	})
}

// StringOpener is a [FileOpener] implementation using a string.
type StringOpener string

// Open implements [FileOpener].
func (o StringOpener) Open() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(string(o))), nil
}

// Filename implements [FileOpener].
func (o StringOpener) Filename() string {
	return ""
}

// Size implements [FileOpener].
func (o StringOpener) Size() int64 {
	return int64(len(o))
}

// Header implements [FileOpener].
func (o StringOpener) Header() textproto.MIMEHeader {
	return textproto.MIMEHeader{
		"Content-Type": []string{"text/plain"},
	}
}

// MarshalJSON implement [json.Marshaler].
func (o StringOpener) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"name":         o.Filename(),
		"size":         o.Size(),
		"content-type": o.Header().Get("content-type"),
	})
}

// DecodeFileHeader is a [Value] decoder for [FileOpener].
// It can decode [*multipart.FileHeader] and string values.
var DecodeFileHeader = NewValueDecoder(
	func(data any) Value[FileOpener] {
		value := NewValue[FileOpener]()

		if data == nil {
			value.F |= IsNil
			return value
		}

		switch t := data.(type) {
		case FileOpener:
			value.set(t)
		case *multipart.FileHeader:
			value.set(&MultipartFileOpener{t})
		case string:
			value.set(StringOpener(t))
		}

		if value.V != nil && value.V.Size() == 0 {
			value.F |= IsEmpty
		}

		return value
	},
	func(text string) Value[FileOpener] {
		value := NewValue[FileOpener]()
		value.set(StringOpener(text))
		if value.V.Size() == 0 {
			value.F |= IsEmpty
		}
		return value
	},
	func(_ FileOpener) string {
		return "-file-"
	},
)

// FileField is a field that holds a [FileOpener] value.
type FileField struct {
	*BaseField[FileOpener]
}

// NewFileField returns a new [FileField] instance.
func NewFileField(name string, options ...any) *FileField {
	return &FileField{
		NewBaseField(name, DecodeFileHeader, options...),
	}
}

// UnmarshalFiles implements [HeaderReader].
func (f *FileField) UnmarshalFiles(files []*multipart.FileHeader) error {
	defer func() {
		f.postBinding(nil)
	}()

	if len(files) == 0 {
		return nil
	}

	// Contrary to the regular list field, we don't need any check here.
	// Passing a [*multipart.FileHeader] here ensures that the value
	// always has an [IsOk] flag.
	f.value = f.decoder.DecodeAny(
		f.preBinding(files[0]),
	)

	return nil
}

// FileListField is a field that holds a list of [FileOpener] values.
type FileListField struct {
	*ListField[FileOpener]
}

// NewFileListField returns a new [FileListField] instance.
func NewFileListField(name string, options ...any) *FileListField {
	return &FileListField{
		NewListField(name, DecodeFileHeader, options...),
	}
}

// UnmarshalFiles implements [HeaderReader].
func (f *FileListField) UnmarshalFiles(files []*multipart.FileHeader) error {
	defer func() {
		if len(f.value.V) == 0 {
			f.value.F |= IsEmpty
		}
		f.postBinding(nil)
	}()

	if len(files) == 0 {
		f.Set(nil)
		return nil
	}

	// Contrary to the regular list field, we don't need any check here.
	// Passing a [*multipart.FileHeader] here ensures that the value
	// always has an [IsOk] flag.
	f.value = Value[[]FileOpener]{}
	for _, file := range files {
		f.value.V = append(f.value.V, f.decoder.DecodeAny(
			f.preBinding(file),
		).V)
	}

	return nil
}

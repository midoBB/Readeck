// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package forms_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"codeberg.org/readeck/readeck/pkg/forms/v2"
)

func TestFileHeaderDecoder(t *testing.T) {
	fh := &multipart.FileHeader{
		Filename: "test.txt",
		Header:   textproto.MIMEHeader{"Content-Type": []string{"text/plain"}},
		Size:     int64(14),
	}
	fhEmpty := &multipart.FileHeader{
		Filename: "test.txt",
		Header:   textproto.MIMEHeader{"Content-Type": []string{"text/plain"}},
		Size:     int64(0),
	}

	t.Run("any", runAnyDecoder(forms.DecodeFileHeader, []anyValueTest[forms.FileOpener]{
		{nil, forms.Value[forms.FileOpener]{
			V: forms.FileOpener(nil),
			F: forms.IsNil | forms.IsEmpty,
		}},
		{"test", forms.Value[forms.FileOpener]{
			V: forms.StringOpener("test"),
			F: forms.IsOk,
		}},
		{"", forms.Value[forms.FileOpener]{
			V: forms.StringOpener(""),
			F: forms.IsOk | forms.IsEmpty,
		}},
		{fh, forms.Value[forms.FileOpener]{
			V: &forms.MultipartFileOpener{fh},
			F: forms.IsOk,
		}},
		{fhEmpty, forms.Value[forms.FileOpener]{
			V: &forms.MultipartFileOpener{fhEmpty},
			F: forms.IsOk | forms.IsEmpty,
		}},
		{12, forms.Value[forms.FileOpener]{
			V: forms.FileOpener(nil),
			F: forms.IsEmpty,
		}},
	}))

	t.Run("text", runTextDecoder(forms.DecodeFileHeader, []textValueTest[forms.FileOpener]{
		{"", forms.Value[forms.FileOpener]{
			V: forms.StringOpener(""),
			F: forms.IsOk | forms.IsEmpty,
		}},
		{"test", forms.Value[forms.FileOpener]{
			V: forms.StringOpener("test"),
			F: forms.IsOk,
		}},
	}))
}

func TestPMultipart(t *testing.T) {
	t.Run("single file", func(t *testing.T) {
		f := forms.Must(
			context.Background(),
			forms.NewTextField("name"),
			forms.NewFileField("file"),
		)

		assert := require.New(t)

		body := new(bytes.Buffer)
		mp := multipart.NewWriter(body)

		// Normal field
		assert.NoError(mp.WriteField("name", "alice"))

		// File field
		w, err := mp.CreateFormFile("file", "file.txt")
		assert.NoError(err)
		_, err = w.Write([]byte("test\ncontent\n"))
		assert.NoError(err)

		assert.NoError(mp.Close())

		r, _ := http.NewRequest(http.MethodPost, "/", body)
		r.Header.Set("content-type", mp.FormDataContentType())
		forms.Bind(f, r)

		assert.True(f.IsBound())
		assert.True(f.IsValid())

		assert.False(f.Get("name").IsNil())
		assert.Equal("alice", f.Get("name").Value())

		assert.Equal("-file-", fmt.Sprint(f.Get("file")))

		var content io.ReadCloser
		buf := new(strings.Builder)
		content, err = f.Get("file").(*forms.FileField).V().Open()
		assert.NoError(err)
		io.Copy(buf, content)
		assert.NoError(content.Close())
		assert.Equal("test\ncontent\n", buf.String())
	})

	t.Run("single file json", func(t *testing.T) {
		f := forms.Must(
			context.Background(),
			forms.NewTextField("name"),
			forms.NewFileField("file"),
		)

		assert := require.New(t)

		body := new(bytes.Buffer)
		enc := json.NewEncoder(body)
		assert.NoError(enc.Encode(map[string]string{
			"name": "alice",
			"file": "test\ncontent\n",
		}))

		r, _ := http.NewRequest(http.MethodPost, "/", body)
		r.Header.Set("content-type", "application/json")
		forms.Bind(f, r)

		assert.True(f.IsBound())
		assert.True(f.IsValid())

		assert.False(f.Get("name").IsNil())
		assert.Equal("alice", f.Get("name").Value())

		assert.Equal("-file-", fmt.Sprint(f.Get("file")))

		var content io.ReadCloser
		buf := new(strings.Builder)
		content, err := f.Get("file").(*forms.FileField).V().Open()
		assert.NoError(err)
		io.Copy(buf, content)
		assert.NoError(content.Close())
		assert.Equal("test\ncontent\n", buf.String())
	})

	t.Run("file list", func(t *testing.T) {
		f := forms.Must(
			context.Background(),
			forms.NewTextField("name"),
			forms.NewFileListField("file"),
		)

		assert := require.New(t)

		body := new(bytes.Buffer)
		mp := multipart.NewWriter(body)

		// Normal field
		assert.NoError(mp.WriteField("name", "alice"))

		// File field
		w, err := mp.CreateFormFile("file", "file.txt")
		assert.NoError(err)
		_, err = w.Write([]byte("test 1\n"))
		assert.NoError(err)

		w, err = mp.CreateFormFile("file", "file.txt")
		assert.NoError(err)
		_, err = w.Write([]byte("test 2\n"))
		assert.NoError(err)

		assert.NoError(mp.Close())

		r, _ := http.NewRequest(http.MethodPost, "/", body)
		r.Header.Set("content-type", mp.FormDataContentType())
		forms.Bind(f, r)

		assert.True(f.IsBound())
		assert.True(f.IsValid())

		assert.False(f.Get("name").IsNil())
		assert.Equal("alice", f.Get("name").Value())

		var content io.ReadCloser
		buf := new(strings.Builder)
		content, err = f.Get("file").(*forms.FileListField).V()[0].Open()
		assert.NoError(err)
		io.Copy(buf, content)
		assert.NoError(content.Close())
		assert.Equal("test 1\n", buf.String())

		buf.Reset()
		content, err = f.Get("file").(*forms.FileListField).V()[1].Open()
		assert.NoError(err)
		io.Copy(buf, content)
		assert.NoError(content.Close())
		assert.Equal("test 2\n", buf.String())
	})
}

func TestMultipartForm(t *testing.T) {
	newForm := func() *forms.Form {
		return forms.Must(
			context.Background(),
			forms.NewTextField("name", forms.Trim, forms.Required),
			forms.NewFileField("file", forms.Required),
		)
	}

	newFormList := func() *forms.Form {
		return forms.Must(
			context.Background(),
			forms.NewTextField("name", forms.Trim, forms.Required),
			forms.NewFileListField("files", forms.Required),
		)
	}

	t.Run("single file", runMultipartForm(func() forms.Binder {
		return newForm()
	}, []multipartTest{
		{
			func(_ *multipart.Writer) error {
				return nil
			},
			`{
				"is_valid": false,
				"errors": null,
				"fields": {
					"file": {
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
						"value": "",
						"errors": [
							"field is required"
					]
					}
				}
			}`,
		},
		{
			func(w *multipart.Writer) error {
				_ = w.WriteField("name", " alice  ")
				fw, err := w.CreateFormFile("file", "file.txt")
				if err != nil {
					return err
				}
				_, err = fw.Write([]byte("test\ncontent\n"))
				if err != nil {
					return err
				}
				return nil
			},
			`{
				"is_valid": true,
				"errors": null,
				"fields": {
					"file": {
						"is_null": false,
						"is_bound": true,
						"value": {
							"content-type": "application/octet-stream",
							"name": "file.txt",
							"size": 13
						},
						"errors": null
					},
					"name": {
						"is_null": false,
						"is_bound": true,
						"value": "alice",
						"errors": null
					}
				}
			}`,
		},
	}))

	t.Run("file list", runMultipartForm(func() forms.Binder {
		return newFormList()
	}, []multipartTest{
		{
			func(_ *multipart.Writer) error {
				return nil
			},
			`{
				"is_valid": false,
				"errors": null,
				"fields": {
					"files": {
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
						"value": "",
						"errors": [
							"field is required"
					]
					}
				}
			}`,
		},
		{
			func(w *multipart.Writer) error {
				_ = w.WriteField("name", " alice  ")
				for _, content := range []string{"test\n", "test\nsecond file\n"} {
					fw, err := w.CreateFormFile("files", "file.txt")
					if err != nil {
						return err
					}
					_, err = fw.Write([]byte(content))
					if err != nil {
						return err
					}
				}

				return nil
			},
			`{
				"is_valid": true,
				"errors": null,
				"fields": {
					"files": {
						"is_null": false,
						"is_bound": true,
						"value": [
							{
								"content-type": "application/octet-stream",
								"name": "file.txt",
								"size": 5
							},
							{
								"content-type": "application/octet-stream",
								"name": "file.txt",
								"size": 17
							}
						],
						"errors": null
					},
					"name": {
						"is_null": false,
						"is_bound": true,
						"value": "alice",
						"errors": null
					}
				}
			}`,
		},
	}))

	t.Run("single file json", runRequestForm(mimeJSON, func() forms.Binder {
		return newForm()
	}, []formTest{
		{
			`{}`,
			`{
				"is_valid": false,
				"errors": null,
				"fields": {
					"file": {
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
						"value": "",
						"errors": [
							"field is required"
					]
					}
				}
			}`,
		},
		{
			`{
				"name": " alice ",
				"file": "test\ncontent\n"
			}`,
			`{
				"is_valid": true,
				"errors": null,
				"fields": {
					"file": {
						"is_null": false,
						"is_bound": true,
						"value": {
							"name": "",
							"size": 13,
							"content-type": "text/plain"
						},
						"errors": null
					},
					"name": {
						"is_null": false,
						"is_bound": true,
						"value": "alice",
						"errors": null
					}
				}
			}`,
		},
		{
			`{
				"name": " alice ",
				"file": 12
			}`,
			`{
				"is_valid": false,
				"errors": null,
				"fields": {
					"file": {
						"is_null": false,
						"is_bound": false,
						"value": null,
						"errors": [
							"invalid value"
						]
					},
					"name": {
						"is_null": false,
						"is_bound": true,
						"value": "alice",
						"errors": null
					}
				}
			}`,
		},
		{
			`{
				"name": " alice ",
				"file": null
			}`,
			`{
				"is_valid": false,
				"errors": null,
				"fields": {
					"file": {
						"is_null": true,
						"is_bound": true,
						"value": null,
						"errors": [
							"field is required"
						]
					},
					"name": {
						"is_null": false,
						"is_bound": true,
						"value": "alice",
						"errors": null
					}
				}
			}`,
		},
	}))

	t.Run("single file values", runRequestForm(mimeValues, func() forms.Binder {
		return newForm()
	}, []formTest{
		{
			"",
			`{
				"is_valid": false,
				"errors": null,
				"fields": {
					"file": {
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
						"value": "",
						"errors": [
							"field is required"
					]
					}
				}
			}`,
		},
		{
			"name=alice&file=",
			`{
				"is_valid": false,
				"errors": null,
				"fields": {
					"file": {
						"is_null": false,
						"is_bound": true,
						"value": {
							"name": "",
							"size": 0,
							"content-type": "text/plain"
						},
						"errors": [
							"field is required"
						]
					},
					"name": {
						"is_null": false,
						"is_bound": true,
						"value": "alice",
						"errors": null
					}
				}
			}`,
		},
		{
			"name=alice&file=test",
			`{
				"is_valid": true,
				"errors": null,
				"fields": {
					"file": {
						"is_null": false,
						"is_bound": true,
						"value": {
							"name": "",
							"size": 4,
							"content-type": "text/plain"
						},
						"errors": null
					},
					"name": {
						"is_null": false,
						"is_bound": true,
						"value": "alice",
						"errors": null
					}
				}
			}`,
		},
	}))

	t.Run("file list json", runRequestForm(mimeJSON, func() forms.Binder {
		return newFormList()
	}, []formTest{
		{
			`{}`,
			`{
				"is_valid": false,
				"errors": null,
				"fields": {
					"files": {
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
						"value": "",
						"errors": [
							"field is required"
						]
					}
				}
			}`,
		},
		{
			`{"files": []}`,
			`{
				"is_valid": false,
				"errors": null,
				"fields": {
					"files": {
						"is_null": false,
						"is_bound": true,
						"value": null,
						"errors": [
							"field is required"
						]
					},
					"name": {
						"is_null": true,
						"is_bound": false,
						"value": "",
						"errors": [
							"field is required"
						]
					}
				}
			}`,
		},
		{
			`{
				"name": "alice",
				"files": ["abc"]
			}`,
			`{
				"is_valid": true,
				"errors": null,
				"fields": {
					"files": {
						"is_null": false,
						"is_bound": true,
						"value": [
							{
								"name": "",
								"content-type": "text/plain",
								"size": 3
							}
						],
						"errors": null
					},
					"name": {
						"is_null": false,
						"is_bound": true,
						"value": "alice",
						"errors": null
					}
				}
			}`,
		},
		{
			`{
				"name": "alice",
				"files": ["abc", "abcdefgh"]
			}`,
			`{
				"is_valid": true,
				"errors": null,
				"fields": {
					"files": {
						"is_null": false,
						"is_bound": true,
						"value": [
							{
								"name": "",
								"content-type": "text/plain",
								"size": 3
							},
							{
								"name": "",
								"content-type": "text/plain",
								"size": 8
							}
						],
						"errors": null
					},
					"name": {
						"is_null": false,
						"is_bound": true,
						"value": "alice",
						"errors": null
					}
				}
			}`,
		},
		{
			`{
				"name": "alice",
				"files": ["abc", null]
			}`,
			`{
				"is_valid": true,
				"errors": null,
				"fields": {
					"files": {
						"is_null": false,
						"is_bound": true,
						"value": [
							{
								"name": "",
								"content-type": "text/plain",
								"size": 3
							}
						],
						"errors": null
					},
					"name": {
						"is_null": false,
						"is_bound": true,
						"value": "alice",
						"errors": null
					}
				}
			}`,
		},
		{
			`{
				"name": "alice",
				"files": ["abc", 12]
			}`,
			`{
				"is_valid": false,
				"errors": null,
				"fields": {
					"files": {
						"is_null": false,
						"is_bound": false,
						"value": null,
						"errors": [
							"invalid value"
						]
					},
					"name": {
						"is_null": false,
						"is_bound": true,
						"value": "alice",
						"errors": null
					}
				}
			}`,
		},
	}))
}

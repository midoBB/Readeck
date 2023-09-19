// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package zipfs_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"codeberg.org/readeck/readeck/pkg/zipfs"
	"github.com/stretchr/testify/assert"
)

func TestHttp(t *testing.T) {
	tests := []struct {
		path            string
		headers         map[string]string
		status          int
		content         []byte
		responseHeaders map[string]string
	}{
		{
			"sub",
			nil,
			http.StatusNotFound,
			nil,
			nil,
		},
		{
			"test-base.txt",
			nil,
			http.StatusOK,
			[]byte("some content that's not compressed\n"),
			map[string]string{
				"content-length": "35",
				"content-type":   "text/plain; charset=utf-8",
				"last-modified":  "Mon, 18 Sep 2023 18:43:25 GMT",
			},
		},
		{
			"test-base.txt",
			map[string]string{
				"if-modified-since": "Mon, 18 Sep 2023 18:43:25 GMT",
			},
			http.StatusNotModified,
			nil,
			nil,
		},
		{
			"test-deflate.txt",
			nil,
			http.StatusOK,
			[]byte("some content that is compressed\nsome content that is compressed\nsome content that is compressed\n"),
			map[string]string{
				"content-length": "96",
			},
		},
		{
			"test-deflate.txt",
			map[string]string{"Accept-Encoding": "deflate"},
			http.StatusOK,
			[]byte{0x2b, 0xce, 0xcf, 0x4d, 0x55, 0x48, 0xce, 0xcf, 0x2b, 0x49, 0xcd, 0x2b, 0x51, 0x28, 0xc9, 0x48, 0x2c, 0x51, 0xc8, 0x2c, 0x6, 0xf2, 0x73, 0xb, 0x8a, 0x52, 0x8b, 0x8b, 0x53, 0x53, 0xb8, 0x8a, 0x29, 0x94, 0x7, 0x0},
			map[string]string{
				"content-length":   "36",
				"content-encoding": "deflate",
			},
		},
	}

	srv := zipfs.HTTPZipFile("fixtures/http.zip")

	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			w := httptest.NewRecorder()
			r, _ := http.NewRequest("GET", test.path, nil)
			if test.headers != nil {
				for k, v := range test.headers {
					r.Header.Set(k, v)
				}
			}

			srv.ServeHTTP(w, r)
			assert.Equal(t, test.status, w.Result().StatusCode)
			if w.Result().StatusCode != http.StatusOK {
				return
			}

			assert.Equal(t, test.content, w.Body.Bytes())

			for k, v := range test.responseHeaders {
				assert.Equal(t, v, w.Result().Header.Get(k))
			}
		})
	}

	t.Run("method", func(t *testing.T) {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("POST", "test-base.txt", nil)
		srv.ServeHTTP(w, r)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Result().StatusCode)
	})

	t.Run("head", func(t *testing.T) {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("HEAD", "test-base.txt", nil)
		srv.ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
		assert.Equal(t, "", w.Body.String())
	})

	t.Run("zip not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("GET", "test-base.txt", nil)
		zipfs.HTTPZipFile("fixtures/noop.zip").ServeHTTP(w, r)
		assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
	})

	t.Run("corrupt zip", func(t *testing.T) {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("GET", "test-base.txt", nil)
		zipfs.HTTPZipFile("fixtures/corrupt.zip").ServeHTTP(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})
}

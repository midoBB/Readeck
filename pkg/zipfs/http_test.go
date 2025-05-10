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
	"github.com/stretchr/testify/require"
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
		//nolint:bodyclose
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			assert := require.New(t)
			w := httptest.NewRecorder()
			r, err := http.NewRequest("GET", test.path, nil)
			assert.NoError(err)
			if test.headers != nil {
				for k, v := range test.headers {
					r.Header.Set(k, v)
				}
			}

			srv.ServeHTTP(w, r)
			assert.Equal(test.status, w.Result().StatusCode)
			if w.Result().StatusCode != http.StatusOK {
				return
			}

			assert.Equal(test.content, w.Body.Bytes())

			for k, v := range test.responseHeaders {
				assert.Equal(v, w.Result().Header.Get(k))
			}
		})
	}

	//nolint:bodyclose
	t.Run("method", func(t *testing.T) {
		assert := require.New(t)
		w := httptest.NewRecorder()
		r, err := http.NewRequest("POST", "test-base.txt", nil)
		assert.NoError(err)
		srv.ServeHTTP(w, r)
		assert.Equal(http.StatusMethodNotAllowed, w.Result().StatusCode)
	})

	//nolint:bodyclose
	t.Run("head", func(t *testing.T) {
		assert := require.New(t)
		w := httptest.NewRecorder()
		r, err := http.NewRequest("HEAD", "test-base.txt", nil)
		assert.NoError(err)
		srv.ServeHTTP(w, r)
		assert.Equal(http.StatusOK, w.Result().StatusCode)
		assert.Empty(w.Body.String())
	})

	//nolint:bodyclose
	t.Run("zip not found", func(t *testing.T) {
		assert := require.New(t)
		w := httptest.NewRecorder()
		r, err := http.NewRequest("GET", "test-base.txt", nil)
		assert.NoError(err)
		zipfs.HTTPZipFile("fixtures/noop.zip").ServeHTTP(w, r)
		assert.Equal(http.StatusNotFound, w.Result().StatusCode)
	})

	//nolint:bodyclose
	t.Run("corrupt zip", func(t *testing.T) {
		assert := require.New(t)
		w := httptest.NewRecorder()
		r, err := http.NewRequest("GET", "test-base.txt", nil)
		assert.NoError(err)
		zipfs.HTTPZipFile("fixtures/corrupt.zip").ServeHTTP(w, r)
		assert.Equal(http.StatusInternalServerError, w.Result().StatusCode)
	})
}

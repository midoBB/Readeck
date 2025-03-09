// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package csrf

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type cookieStore func() []byte

func mustHexDecode(s string) []byte {
	res, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return res
}

func (f cookieStore) Load(_ *http.Request, token any) error {
	if t, ok := token.(*[]byte); ok {
		*t = f()
	}
	return nil
}

func (f cookieStore) Save(w http.ResponseWriter, _ *http.Request, token any) error {
	w.Header().Set("x-token", hex.EncodeToString(token.([]byte)))
	return nil
}

var (
	okToken = mustHexDecode("cbcade8b75409fab4a9c762dd3c2e84ab22760521bb17475a160837d84c89aa7")
	okStore = cookieStore(func() []byte {
		return okToken
	})
)

func testRequest(store cookieStore,
	prepareRequest func(assert *require.Assertions, r *http.Request),
	checkResponse func(assert *require.Assertions, w *httptest.ResponseRecorder, err error),
) func(t *testing.T) {
	return func(t *testing.T) {
		assert := require.New(t)
		var innerRequest *http.Request
		mux := http.NewServeMux()
		mux.Handle("/", http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			innerRequest = r
		}))

		r := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		var err error

		handler := Protect(store, WithErrorHandler(func(w http.ResponseWriter, r *http.Request) {
			err = GetError(r)
			w.WriteHeader(412)
		}))(mux)

		handler.ServeHTTP(w, r)
		assert.Equal(200, w.Result().StatusCode)

		prepareRequest(assert, innerRequest)
		r = innerRequest.WithContext(context.Background())
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		checkResponse(assert, w, err)
	}
}

func TestMaskUnmaskTokens(t *testing.T) {
	assert := require.New(t)
	realToken, err := generateRandomBytes(tokenLength)
	assert.NoError(err)

	issued := mask(realToken)
	unmasked := unmask(issued)
	assert.True(compareTokens(unmasked, realToken))
}

func TestCSRF(t *testing.T) {
	t.Run("generate token", testRequest(
		cookieStore(func() []byte {
			return nil
		}),
		func(_ *require.Assertions, _ *http.Request) {},
		func(assert *require.Assertions, w *httptest.ResponseRecorder, _ error) {
			assert.Equal(200, w.Result().StatusCode)
			assert.NotEmpty(w.Header().Get("x-token"))
		},
	))

	t.Run("with header token", testRequest(
		okStore,
		func(assert *require.Assertions, r *http.Request) {
			masked, err := b64.DecodeString(Token(r))
			assert.NoError(err)
			assert.Equal(okToken, unmask(masked))
			r.Method = "POST"
			r.Header.Set("X-CSRF-Token", Token(r))
		},
		func(assert *require.Assertions, w *httptest.ResponseRecorder, _ error) {
			assert.Equal(200, w.Result().StatusCode)
		},
	))

	t.Run("https with header token", testRequest(
		okStore,
		func(assert *require.Assertions, r *http.Request) {
			masked, err := b64.DecodeString(Token(r))
			assert.NoError(err)
			assert.Equal(okToken, unmask(masked))
			r.Method = "POST"
			r.URL.Scheme = "https"
			r.URL.Host = "example.net"
			r.Header.Set("Referer", "https://example.net/test")
			r.Header.Set("X-CSRF-Token", Token(r))
		},
		func(assert *require.Assertions, w *httptest.ResponseRecorder, _ error) {
			assert.Equal(200, w.Result().StatusCode)
		},
	))

	t.Run("ok with form token", testRequest(
		okStore,
		func(_ *require.Assertions, r *http.Request) {
			r.Method = "POST"
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.Body = io.NopCloser(strings.NewReader(url.Values{
				FieldName(r): []string{Token(r)},
			}.Encode()))
		},
		func(assert *require.Assertions, w *httptest.ResponseRecorder, _ error) {
			assert.Equal(200, w.Result().StatusCode)
		},
	))

	t.Run("ok with multipart token", testRequest(
		okStore,
		func(assert *require.Assertions, r *http.Request) {
			body := new(bytes.Buffer)
			mp := multipart.NewWriter(body)
			f, err := mp.CreateFormField(FieldName(r))
			assert.NoError(err)
			_, _ = io.WriteString(f, Token(r))
			_ = mp.Close()

			r.Method = "PATCH"
			r.Header.Set("Content-Type", mp.FormDataContentType())
			r.Body = io.NopCloser(body)
		},
		func(assert *require.Assertions, w *httptest.ResponseRecorder, _ error) {
			assert.Equal(200, w.Result().StatusCode)
		},
	))

	t.Run("tampered token", testRequest(
		okStore,
		func(_ *require.Assertions, r *http.Request) {
			r.Method = "POST"
			masked := Token(r)
			m, _ := b64.DecodeString(masked)
			m[18] ^= m[18]

			r.Header.Set("X-CSRF-Token", b64.EncodeToString(m))
		},
		func(assert *require.Assertions, w *httptest.ResponseRecorder, err error) {
			assert.Equal(412, w.Result().StatusCode)
			assert.ErrorContains(err, "token does not match")
		},
	))

	t.Run("invalid token", testRequest(
		okStore,
		func(_ *require.Assertions, r *http.Request) {
			r.Method = "POST"
			r.Header.Set("X-CSRF-Token", "🙂test")
		},
		func(assert *require.Assertions, w *httptest.ResponseRecorder, err error) {
			assert.Equal(412, w.Result().StatusCode)
			assert.ErrorContains(err, "invalid token")
			assert.ErrorContains(err, "base64")
		},
	))

	t.Run("ko with form token", testRequest(
		okStore,
		func(_ *require.Assertions, r *http.Request) {
			r.Method = "POST"
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.Body = io.NopCloser(strings.NewReader(url.Values{
				"foo": []string{"abcd"},
			}.Encode()))
		},
		func(assert *require.Assertions, w *httptest.ResponseRecorder, err error) {
			assert.Equal(412, w.Result().StatusCode)
			assert.ErrorContains(err, "invalid token")
		},
	))

	t.Run("ko with multipart token", testRequest(
		okStore,
		func(assert *require.Assertions, r *http.Request) {
			body := new(bytes.Buffer)
			mp := multipart.NewWriter(body)
			f, err := mp.CreateFormField("foo")
			assert.NoError(err)
			_, _ = io.WriteString(f, "abcd")
			_ = mp.Close()

			r.Method = "PATCH"
			r.Header.Set("Content-Type", mp.FormDataContentType())
			r.Body = io.NopCloser(body)
		},
		func(assert *require.Assertions, w *httptest.ResponseRecorder, err error) {
			assert.Equal(412, w.Result().StatusCode)
			assert.ErrorContains(err, "invalid token")
		},
	))

	t.Run("https no referer", testRequest(
		okStore,
		func(_ *require.Assertions, r *http.Request) {
			r.Method = "POST"
			r.URL.Scheme = "https"
			r.URL.Host = "example.net"
		},
		func(assert *require.Assertions, w *httptest.ResponseRecorder, err error) {
			assert.Equal(412, w.Result().StatusCode)
			assert.ErrorContains(err, "invalid referrer")
		},
	))

	t.Run("https invalid referer", testRequest(
		okStore,
		func(_ *require.Assertions, r *http.Request) {
			r.Method = "POST"
			r.URL.Scheme = "https"
			r.URL.Host = "example.net"
			r.Header.Set("Referer", "http://example.org/")
		},
		func(assert *require.Assertions, w *httptest.ResponseRecorder, err error) {
			assert.Equal(412, w.Result().StatusCode)
			assert.ErrorContains(err, "referrer does not match")
		},
	))
}

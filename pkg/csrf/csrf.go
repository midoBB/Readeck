// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// Package csrf provides functions to setup CSRF protection.
// It's mostly a modern port of Gorilla CSRF.
// Copyright (c) 2023 The Gorilla Authors. All rights reserved.
// https://github.com/gorilla/csrf
package csrf

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
)

const tokenLength = 32

type contextKey struct {
	name string
}

var (
	b64             = base64.URLEncoding
	ctxTokenKey     = &contextKey{"token"}
	ctxFieldNameKey = &contextKey{"fieldname"}
	ctxErrorKey     = &contextKey{"error"}
	safeMethods     = []string{"GET", "HEAD", "OPTIONS", "TRACE"}
)

type storagHandler interface {
	Load(r *http.Request, token any) error
	Save(w http.ResponseWriter, r *http.Request, token any) error
}

type csrfHandler struct {
	h            http.Handler
	store        storagHandler
	fieldName    string
	headerName   string
	errorHandler func(w http.ResponseWriter, r *http.Request)
}

// Option describes a functional option for configuring the CSRF handler.
type Option func(*csrfHandler)

// WithFieldName sets the form field's name.
func WithFieldName(name string) Option {
	return func(ch *csrfHandler) {
		ch.fieldName = name
	}
}

// WithErrorHandler sets a custom error handler for rejected requests.
func WithErrorHandler(h func(w http.ResponseWriter, r *http.Request)) Option {
	return func(ch *csrfHandler) {
		ch.errorHandler = h
	}
}

// Protect is HTTP middleware that provides Cross-Site Request Forgery
// protection.
//
// It securely generates a masked (unique-per-request) token that
// can be embedded in the HTTP response (e.g. form field or HTTP header).
// The original token must be stored in a way that makes it innacessible to
// the page's content. The default storage is an HTTP only, encrypted, cookie.
// Requests that do not provide a matching token are served with an
// HTTP 303 Forbidden response.
func Protect(cookie storagHandler, options ...Option) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		h := &csrfHandler{
			h:            next,
			store:        cookie,
			fieldName:    "__csrf__",
			headerName:   "X-CSRF-Token",
			errorHandler: defaultErrorHandler,
		}

		for _, fn := range options {
			fn(h)
		}

		return h
	}
}

// ServeHTTP implements [http.Handler] for the CSRF handler.
func (ch *csrfHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var token []byte

	err := ch.store.Load(r, &token)
	if err != nil || len(token) != tokenLength {
		// If there was an error or no token at all, generate
		// a new token and save it in the storage (normally in a cookie)
		token, err = generateRandomBytes(tokenLength)
		if err != nil {
			ch.sendError(err, w, r)
			return
		}

		// Save the new token
		if err := ch.store.Save(w, r, token); err != nil {
			ch.sendError(err, w, r)
			return
		}
	}

	masked := mask(token)

	ctx := r.Context()
	ctx = context.WithValue(ctx, ctxTokenKey, masked)
	ctx = context.WithValue(ctx, ctxFieldNameKey, ch.fieldName)

	r = r.WithContext(ctx)

	if !slices.Contains(safeMethods, r.Method) {
		if r.URL.Scheme == "https" {
			referer, err := url.Parse(r.Referer())
			if err != nil || referer.String() == "" {
				ch.sendError(fmt.Errorf("%w %s", err, "invalid referrer"), w, r)
				return
			}

			valid := referer.Scheme == r.URL.Scheme && referer.Host == r.URL.Host

			if !valid {
				ch.sendError(errors.New("referrer does not match"), w, r)
				return
			}
		}

		rMasked, err := ch.requestToken(r)
		if err != nil {
			ch.sendError(fmt.Errorf("%w invalid token", err), w, r)
			return
		}

		if len(rMasked) == 0 {
			ch.sendError(errors.New("invalid token"), w, r)
			return
		}

		rToken := unmask(rMasked)

		if !compareTokens(rToken, token) {
			ch.sendError(errors.New("token does not match"), w, r)
			return
		}
	}

	// Set the Vary: Cookie header to protect clients from caching the response.
	w.Header().Add("Vary", "Cookie")

	// Handle request
	ch.h.ServeHTTP(w, r)
}

func (ch *csrfHandler) requestToken(r *http.Request) ([]byte, error) {
	// 1. check the header first
	issued := r.Header.Get(ch.headerName)

	// 2. fall back to the form value
	// this takes care of multipart or regular forms.
	if issued == "" {
		issued = r.PostFormValue(ch.fieldName)
	}

	// return empty when no token was found
	if issued == "" {
		return nil, nil
	}

	// decode the token
	return b64.DecodeString(issued)
}

func (ch *csrfHandler) sendError(err error, w http.ResponseWriter, r *http.Request) {
	ctx := context.WithValue(r.Context(), ctxErrorKey, err)
	ch.errorHandler(w, r.WithContext(ctx))
}

// Token returns a masked CSRF token ready for passing into HTML template or
// a JSON response body. An empty token will be returned if the middleware
// has not been applied (which will fail subsequent validation).
func Token(r *http.Request) string {
	if t, ok := r.Context().Value(ctxTokenKey).([]byte); ok {
		return b64.EncodeToString(t)
	}
	return ""
}

// FieldName returns the CSRF form field name.
func FieldName(r *http.Request) string {
	if n, ok := r.Context().Value(ctxFieldNameKey).(string); ok {
		return n
	}
	return ""
}

// GetError returns the CSRF error reason from the request's context.
func GetError(r *http.Request) error {
	err, _ := r.Context().Value(ctxErrorKey).(error)
	return err
}

func defaultErrorHandler(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
}

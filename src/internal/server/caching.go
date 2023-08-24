// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package server

import (
	"fmt"
	"hash/crc64"
	"net/http"
	"sort"
	"time"

	"github.com/readeck/readeck/configs"
	"github.com/readeck/readeck/internal/auth"
)

// Etager must provides a function that returns a list of
// strings used to build an etag header.
type Etager interface {
	GetSumStrings() []string
}

// LastModer must provides a function that returns a list
// of times used to build a Last-Modified header.
type LastModer interface {
	GetLastModified() []time.Time
}

type checkResult int

const (
	checkNone checkResult = iota
	checkTrue
	checkFalse
)

// WriteEtag adds an Etag header to the response, based on
// the values sent by GetSumStrings. The build date is always
// included.
func (s *Server) WriteEtag(w http.ResponseWriter, r *http.Request, taggers ...Etager) {
	h := crc64.New(crc64.MakeTable(crc64.ISO))
	for _, tager := range taggers {
		for _, x := range tager.GetSumStrings() {
			h.Write([]byte(x))
		}
	}
	h.Write([]byte(configs.BuildTime().String()))

	if user := auth.GetRequestUser(r); user.ID != 0 {
		fmt.Fprint(h, user.ID)
	}

	w.Header().Set("Etag", fmt.Sprintf("%x", h.Sum64()))
}

// WriteLastModified adds a Last-Modified headers using the most
// recent date of GetLastModified and the build date.
func (s *Server) WriteLastModified(w http.ResponseWriter, r *http.Request, moders ...LastModer) {
	mtimes := []time.Time{configs.BuildTime()}
	for _, m := range moders {
		mtimes = append(mtimes, m.GetLastModified()...)
	}

	if user := auth.GetRequestUser(r); user.ID != 0 {
		mtimes = append(mtimes, user.GetLastModified()...)
	}

	sort.Slice(mtimes, func(i, j int) bool {
		return mtimes[i].After(mtimes[j])
	})

	w.Header().Set("Last-Modified", mtimes[0].Format(http.TimeFormat))
}

// WithCacheControl sends the global caching headers
func (s *Server) WithCacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private")
		next.ServeHTTP(w, r)
	})
}

// WithCaching is a middleware that checks if an Etag and/or a
// Last-Modified headers are sent with the response. If the
// request has the correspondign cache header and theys match
// the request stops with a 304.
func (s *Server) WithCaching(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}

		if checkIfMatch(w, r)|checkIfModifiedSince(w, r) == checkTrue {
			writeNotModified(w)
			return
		}

		// Cancel the caching headers when there are messages.
		// It prevents the message to stay on the page forever.
		if len(s.Flashes(r)) > 0 {
			w.Header().Del("Last-Modified")
			w.Header().Del("Etag")
		}

		next.ServeHTTP(w, r)
	})
}

func writeNotModified(w http.ResponseWriter) {
	w.Header().Del("Content-Type")
	w.Header().Del("Content-Length")
	w.Header().Del("Content-Security-Policy")
	w.Header().Del("Last-Modified")
	w.Header().Del("Etag")

	w.WriteHeader(http.StatusNotModified)
}

func checkIfModifiedSince(w http.ResponseWriter, r *http.Request) checkResult {
	rh := r.Header.Get("If-Modified-Since")
	if rh == "" {
		return checkNone
	}
	wh := w.Header().Get("Last-Modified")
	if wh == "" {
		return checkNone
	}

	var err error
	var ims time.Time
	var modtime time.Time

	if ims, err = http.ParseTime(rh); err != nil {
		return checkFalse
	}

	if modtime, err = http.ParseTime(wh); err != nil {
		return checkFalse
	}

	ims = ims.Truncate(time.Second)
	modtime = modtime.Truncate(time.Second)

	if modtime.Before(ims) || modtime.Equal(ims) {
		return checkTrue
	}
	return checkFalse
}

func checkIfMatch(w http.ResponseWriter, r *http.Request) checkResult {
	rh := r.Header.Get("If-None-Match")
	if rh == "" {
		return checkNone
	}
	wh := w.Header().Get("Etag")
	if wh == "" {
		return checkNone
	}

	if rh == wh {
		return checkTrue
	}
	return checkFalse
}

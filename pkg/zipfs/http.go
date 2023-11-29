// SPDX-FileCopyrightText: © 2020 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package zipfs

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gabriel-vasile/mimetype"
)

// HTTPZipFile serves a zip file as an HTTP directory (without listing)
// It properly handles If-Modified-Since header and can serve the compressed
// content when deflate is in Accept-Encoding and the content is compressed
// with deflate.
type HTTPZipFile string

type serveOption func(w http.ResponseWriter, status int)

func (f HTTPZipFile) ServeHTTP(w http.ResponseWriter, r *http.Request, options ...serveOption) {
	if r.Method != "GET" && r.Method != "HEAD" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	filename := r.URL.Path
	zr, err := zip.OpenReader(string(f))
	if err != nil {
		f.error(w, err)
		return
	}
	defer zr.Close()

	zf, err := f.getEntry(filename, zr)
	if err != nil {
		f.error(w, err)
		return
	}

	f.serveEntry(w, r, zf, options...)
}

// getEntry returns a zip file entry. It must be.
func (f HTTPZipFile) getEntry(name string, zr *zip.ReadCloser) (*zip.File, error) {
	for _, x := range zr.File {
		if x.Name == name && !x.FileInfo().IsDir() {
			return x, nil
		}
	}
	return nil, os.ErrNotExist
}

func (f HTTPZipFile) serveEntry(w http.ResponseWriter, r *http.Request, zf *zip.File, options ...serveOption) {
	if f.checkIfModifiedSince(r, zf.Modified.UTC()) {
		applyOptions(options, w, http.StatusNotModified)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Last-Modified", zf.Modified.UTC().Format(http.TimeFormat))

	fp, err := zf.Open()
	if err != nil {
		f.error(w, err, options...)
		return
	}
	defer fp.Close() //nolint:errcheck

	// Sniff the content
	buf := new(bytes.Buffer)
	mtype, err := mimetype.DetectReader(io.TeeReader(fp, buf))
	if err != nil {
		f.error(w, err, options...)
		return
	}

	w.Header().Set("Content-Type", mtype.String())

	ae := r.Header.Get("Accept-Encoding")

	if r.Method == "HEAD" {
		return
	}

	// Directly send the compressed data when possible
	if strings.Contains(ae, "deflate") && zf.Method == zip.Deflate {
		w.Header().Set("Content-Encoding", "deflate")
		w.Header().Set("Content-Length", strconv.FormatUint(zf.CompressedSize64, 10))

		cfp, err := zf.OpenRaw()
		if err != nil {
			f.error(w, err, options...)
			return
		}

		applyOptions(options, w, http.StatusOK)
		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, cfp); err != nil {
			panic(err)
		}
		return
	}

	w.Header().Set("Content-Length", strconv.FormatUint(zf.UncompressedSize64, 10))
	applyOptions(options, w, http.StatusOK)
	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, io.MultiReader(buf, fp))
	if err != nil && !errors.Is(err, syscall.EPIPE) {
		panic(err)
	}
}

func (f HTTPZipFile) error(w http.ResponseWriter, err error, options ...serveOption) {
	status := http.StatusInternalServerError
	if os.IsNotExist(err) {
		status = http.StatusNotFound
	}

	applyOptions(options, w, status)
	http.Error(w, http.StatusText(status), status)
}

func (f HTTPZipFile) checkIfModifiedSince(r *http.Request, modtime time.Time) bool {
	ius := r.Header.Get("If-Modified-Since")
	if ius == "" {
		return false
	}
	t, err := http.ParseTime(ius)
	if err != nil {
		return false
	}

	modtime = modtime.Truncate(time.Second)
	if modtime.Before(t) || modtime.Equal(t) {
		return true
	}
	return false
}

func applyOptions(options []serveOption, w http.ResponseWriter, status int) {
	for _, f := range options {
		f(w, status)
	}
}

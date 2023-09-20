// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package zipfs_test

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	"codeberg.org/readeck/readeck/pkg/zipfs"
	"github.com/stretchr/testify/assert"
)

func getZipContent(s []byte) []string {
	r := bytes.NewReader(s)
	zr, err := zip.NewReader(r, r.Size())
	if err != nil {
		panic(err)
	}
	files := []string{}
	for _, x := range zr.File {
		files = append(files, x.Name)
	}
	return files
}

func TestZipRW(t *testing.T) {
	Dest := new(bytes.Buffer)

	z := zipfs.NewZipRW(Dest, nil, 0)
	err := z.Add(
		&zip.FileHeader{Name: "test/foo/bar"},
		strings.NewReader("test file"),
	)
	assert.NoError(t, err)

	err = z.Add(
		&zip.FileHeader{Name: "test/foo/new"},
		strings.NewReader("test file"),
	)
	assert.NoError(t, err)

	err = z.Add(
		&zip.FileHeader{Name: "abc.txt", Method: zip.Deflate},
		strings.NewReader("test file"),
	)
	assert.NoError(t, err)

	z.Close()

	t.Run("initial content", func(t *testing.T) {
		assert.EqualValues(t,
			[]string{
				"test/",
				"test/foo/",
				"test/foo/bar",
				"test/foo/new",
				"abc.txt",
			},
			getZipContent(Dest.Bytes()),
		)

		r := bytes.NewReader(Dest.Bytes())
		zr, err := zip.NewReader(r, r.Size())
		assert.NoError(t, err)
		rf, _ := zr.Open("test/foo/bar")
		contents, _ := io.ReadAll(rf)
		assert.Equal(t, []byte("test file"), contents)
	})

	t.Run("file exists", func(t *testing.T) {
		d := new(bytes.Buffer)

		z := zipfs.NewZipRW(d, nil, 0)
		err := z.Add(
			&zip.FileHeader{Name: "test/foo/bar"},
			strings.NewReader("test file"),
		)
		assert.NoError(t, err)

		err = z.Add(
			&zip.FileHeader{Name: "test/foo/bar/test"},
			strings.NewReader("test file"),
		)
		assert.ErrorContains(t, err, `ile "test/foo/bar" already exists`)

		err = z.Add(
			&zip.FileHeader{Name: "test/foo/bar"},
			strings.NewReader("test file"),
		)
		assert.ErrorContains(t, err, `file "test/foo/bar" already exists`)
	})

	t.Run("copy files", func(t *testing.T) {
		r := bytes.NewReader(Dest.Bytes())
		d := new(bytes.Buffer)

		z := zipfs.NewZipRW(d, r, int64(r.Len()))
		err := z.Copy("test/foo/bar")
		assert.NoError(t, err)

		err = z.Copy("test/foo/new")
		assert.NoError(t, err)

		err = z.Copy("test/foo/new")
		assert.ErrorContains(t, err, `file "test/foo/new" already exists`)

		z.Close()

		assert.EqualValues(t,
			[]string{
				"test/",
				"test/foo/",
				"test/foo/bar",
				"test/foo/new",
			},
			getZipContent(d.Bytes()),
		)
	})
}

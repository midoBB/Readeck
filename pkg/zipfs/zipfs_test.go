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
	"github.com/stretchr/testify/require"
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
	assert := require.New(t)
	Dest := new(bytes.Buffer)

	z := zipfs.NewZipRW(Dest, nil, 0)
	err := z.Add(
		&zip.FileHeader{Name: "test/foo/bar"},
		strings.NewReader("test file"),
	)
	assert.NoError(err)

	err = z.Add(
		&zip.FileHeader{Name: "test/foo/new"},
		strings.NewReader("test file"),
	)
	assert.NoError(err)

	err = z.Add(
		&zip.FileHeader{Name: "abc.txt", Method: zip.Deflate},
		strings.NewReader("test file"),
	)
	assert.NoError(err)
	assert.NoError(z.Close())

	t.Run("initial content", func(t *testing.T) {
		assert := require.New(t)
		assert.Exactly(
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
		assert.NoError(err)
		rf, err := zr.Open("test/foo/bar")
		assert.NoError(err)
		contents, err := io.ReadAll(rf)
		assert.NoError(err)
		assert.Equal([]byte("test file"), contents)
	})

	t.Run("file exists", func(t *testing.T) {
		assert := require.New(t)
		d := new(bytes.Buffer)

		z := zipfs.NewZipRW(d, nil, 0)
		err := z.Add(
			&zip.FileHeader{Name: "test/foo/bar"},
			strings.NewReader("test file"),
		)
		assert.NoError(err)

		err = z.Add(
			&zip.FileHeader{Name: "test/foo/bar/test"},
			strings.NewReader("test file"),
		)
		assert.ErrorContains(err, `file "test/foo/bar" already exists`)

		err = z.Add(
			&zip.FileHeader{Name: "test/foo/bar"},
			strings.NewReader("test file"),
		)
		assert.ErrorContains(err, `file "test/foo/bar" already exists`)
	})

	t.Run("copy files", func(t *testing.T) {
		assert := require.New(t)
		r := bytes.NewReader(Dest.Bytes())
		d := new(bytes.Buffer)

		z := zipfs.NewZipRW(d, r, int64(r.Len()))

		assert.NoError(z.Copy("test/foo/bar"))
		assert.NoError(z.Copy("test/foo/new"))
		assert.ErrorContains(z.Copy("test/foo/new"), `file "test/foo/new" already exists`)

		assert.NoError(z.Close())

		assert.Exactly(
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

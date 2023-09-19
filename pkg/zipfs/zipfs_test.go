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

func TestZipRW(t *testing.T) {
	buf := new(bytes.Buffer)

	z := zipfs.NewZipRW(buf, nil, 0)
	err := z.Add(
		&zip.FileHeader{Name: "test/foo/bar"},
		strings.NewReader("test file"),
	)
	assert.NoError(t, err)

	err = z.Add(
		&zip.FileHeader{Name: "abc.txt", Method: zip.Deflate},
		strings.NewReader("test file"),
	)
	assert.NoError(t, err)

	z.Close()

	r := bytes.NewReader(buf.Bytes())
	zr, err := zip.NewReader(r, r.Size())
	assert.NoError(t, err)

	expected := []string{
		"test/",
		"test/foo/",
		"test/foo/bar",
		"abc.txt",
	}
	files := []string{}
	for _, x := range zr.File {
		files = append(files, x.Name)
	}
	assert.EqualValues(t, expected, files)

	rf, _ := zr.Open("test/foo/bar")
	contents, _ := io.ReadAll(rf)
	assert.Equal(t, []byte("test file"), contents)

	t.Run("copy files", func(t *testing.T) {
		r := bytes.NewReader(buf.Bytes())
		buf := new(bytes.Buffer)

		z := zipfs.NewZipRW(buf, r, int64(r.Len()))
		err := z.Copy("test/foo/bar")
		assert.NoError(t, err)
		z.Close()
	})
}

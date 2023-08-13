// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package img_test

import (
	"bytes"
	"image"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/readeck/readeck/pkg/img"
)

func TestNativeImageProps(t *testing.T) {
	r := bytes.NewReader(newImage(200, 100))
	im, err := img.NewNativeImage(r)
	assert.Nil(t, err)

	assert.Equal(t, im.Format(), "png")
	assert.Equal(t, im.ContentType(), "image/png")
	assert.Equal(t, im.Width(), uint(200))
	assert.Equal(t, im.Height(), uint(100))

	m := im.Image()
	assert.Equal(t, m.Bounds().Dx(), 200)
	assert.Equal(t, m.Bounds().Dy(), 100)
}

func TestNativeImageLoadError(t *testing.T) {
	im, err := img.NewNativeImage(bytes.NewReader([]byte{}))
	assert.Nil(t, im)
	assert.Error(t, err)
	assert.EqualError(t, err, "image: unknown format")
}

func TestNativeImageTooBig(t *testing.T) {
	r := bytes.NewReader(newImage(6001, 5000))
	im, err := img.NewNativeImage(r)
	assert.Nil(t, im)
	assert.Error(t, err)
	assert.EqualError(t, err, "image is too big")
}

func TestNativeImageResize(t *testing.T) {
	r := bytes.NewReader(newImage(200, 100))
	im, err := img.NewNativeImage(r)
	assert.Nil(t, err)
	im.Resize(400, 200)

	buf := new(bytes.Buffer)
	err = im.Encode(buf)
	assert.Nil(t, err)

	p, _, err := image.Decode(buf)
	assert.Nil(t, err)
	assert.Equal(t, 400, p.Bounds().Dx())
	assert.Equal(t, 200, p.Bounds().Dy())
}

func TestNativeImageEncode(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"png", "png"},
		{"", "png"},
		{"jpeg", "jpeg"},
		{"gif", "gif"},
		{"bmp", "jpeg"},
	}

	data := newImage(200, 100)
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			im, err := img.NewNativeImage(bytes.NewReader(data))
			assert.Nil(t, err)

			im.SetFormat(test.format)

			buf := new(bytes.Buffer)
			err = im.Encode(buf)
			assert.Nil(t, err)

			im, err = img.NewNativeImage(buf)
			assert.Nil(t, err)
			assert.Equal(t, test.expected, im.Format())
		})
	}
}

func TestNativeImageOperations(t *testing.T) {
	tests := []func(*img.NativeImage) error{
		func(im *img.NativeImage) error { return im.Grayscale() },
		func(im *img.NativeImage) error { return im.Gray16() },
		func(im *img.NativeImage) error { return im.Clean() },
		func(im *img.NativeImage) error { return im.SetCompression(img.CompressionBest) },
		func(im *img.NativeImage) error { return im.SetQuality(1) },
	}

	data := newImage(200, 100)
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			im, err := img.NewNativeImage(bytes.NewReader(data))
			assert.Nil(t, err)
			err = test(im)
			assert.Nil(t, err)
		})
	}
}

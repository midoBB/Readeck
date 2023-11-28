// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package img_test

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"codeberg.org/readeck/readeck/pkg/img"
)

func newImage(w, h int) []byte {
	tl := image.Point{0, 0}
	br := image.Point{w, h}

	m := image.NewNRGBA(image.Rectangle{tl, br})

	buf := new(bytes.Buffer)
	e := &png.Encoder{CompressionLevel: png.BestSpeed}
	if err := e.Encode(buf, m); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

func TestImageFit(t *testing.T) {
	data := newImage(200, 100)

	tests := []struct {
		size     [2]int
		expected [2]int
	}{
		{[2]int{100, 100}, [2]int{100, 50}},
		{[2]int{300, 300}, [2]int{200, 100}},
		{[2]int{50, 100}, [2]int{50, 25}},
		{[2]int{100, 50}, [2]int{100, 50}},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			im, err := img.NewNativeImage(bytes.NewReader(data))
			assert.Nil(t, err)
			err = img.Fit(im, uint(test.size[0]), uint(test.size[1]))
			assert.Nil(t, err)
			assert.Equal(t, test.expected, [2]int{int(im.Width()), int(im.Height())})
		})
	}
}

func TestImagePipeline(t *testing.T) {
	data := newImage(200, 100)

	tests := []struct {
		pipeline []img.ImageFilter
		err      string
	}{
		{
			[]img.ImageFilter{
				func(i img.Image) error { return i.Clean() },
			},
			"",
		},
		{
			[]img.ImageFilter{
				func(i img.Image) error { return i.Clean() },
				func(i img.Image) error { return errors.New("some error") },
			},
			"some error",
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			im, err := img.NewNativeImage(bytes.NewReader(data))
			assert.Nil(t, err)
			err = img.Pipeline(im, test.pipeline...)
			if test.err != "" {
				assert.EqualError(t, err, test.err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

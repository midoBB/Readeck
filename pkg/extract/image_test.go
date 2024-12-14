// SPDX-FileCopyrightText: © 2020 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package extract

import (
	"bytes"
	"errors"
	"image"
	"net/url"
	"testing"

	"codeberg.org/readeck/readeck/pkg/img"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

func TestRemoteImage(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "/bogus", newFileResponder("images/bogus"))
	httpmock.RegisterResponder("GET", "/404", httpmock.NewJsonResponderOrPanic(404, ""))
	httpmock.RegisterResponder("GET", "/error", httpmock.NewErrorResponder(errors.New("HTTP")))

	formats := []string{"jpeg", "png", "gif", "ico", "bmp"}
	for _, name := range formats {
		name = "/img." + name
		httpmock.RegisterResponder("GET", name, newFileResponder("images/"+name))
	}

	t.Run("RemoteImage", func(t *testing.T) {
		t.Run("errors", func(t *testing.T) {
			tests := []struct {
				name string
				path string
				err  string
			}{
				{"url", "", "No image URL"},
				{"404", "/404", "Invalid response status (404)"},
				{"http", "/error", `Get "/error": HTTP`},
				{"bogus", "/bogus", "no img handler for application/octet-stream"},
			}

			for _, x := range tests {
				t.Run(x.name, func(t *testing.T) {
					ri, err := NewRemoteImage(x.path, nil)
					require.Nil(t, ri)
					if ri != nil {
						defer ri.Close() //nolint:errcheck
					}
					require.Equal(t, x.err, err.Error())
				})
			}
		})

		for _, format := range formats {
			t.Run(format, func(t *testing.T) {
				ri, err := NewRemoteImage("/img."+format, nil)
				require.NoError(t, err)
				defer ri.Close() //nolint:errcheck
				require.Equal(t, format, ri.Format())
			})
		}

		t.Run("fit", func(t *testing.T) {
			assert := require.New(t)
			ri, _ := NewRemoteImage("/img.png", nil)
			defer ri.Close() //nolint:errcheck

			w := ri.Width()
			h := ri.Height()
			assert.Equal([]uint{240, 181}, []uint{w, h})

			assert.NoError(img.Fit(ri, uint(24), uint(24)))
			assert.Equal(uint(24), ri.Width())
			assert.Equal(uint(18), ri.Height())

			assert.NoError(img.Fit(ri, 240, 240))
			assert.Equal(uint(24), ri.Width())
			assert.Equal(uint(18), ri.Height())
		})

		t.Run("encode", func(t *testing.T) {
			tests := []struct {
				name     string
				path     string
				format   string
				expected string
			}{
				{"auto-png", "/img.png", "", "png"},
				{"jpeg-jpeg", "/img.jpeg", "jpeg", "jpeg"},
				{"gif-gif", "/img.gif", "gif", "gif"},
				{"png-png", "/img.png", "png", "png"},
				{"png-gif", "/img.png", "gif", "gif"},
			}

			for _, x := range tests {
				t.Run(x.format, func(t *testing.T) {
					assert := require.New(t)
					ri, err := NewRemoteImage(x.path, nil)
					assert.NoError(err)
					defer func() {
						if err := ri.Close(); err != nil {
							panic(err)
						}
					}()
					assert.NoError(ri.SetFormat(x.format))

					var buf bytes.Buffer
					assert.NoError(ri.Encode(&buf))

					_, format, _ := image.DecodeConfig(bytes.NewReader(buf.Bytes()))
					assert.Equal(format, ri.Format())
				})
			}
		})
	})
}

func TestPicture(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "/img", newFileResponder("images/img.png"))

	base, _ := url.Parse("http://x/index.html")

	t.Run("URL error", func(t *testing.T) {
		p, err := NewPicture("/\b0x7f", base)
		require.Nil(t, p)
		require.Error(t, err)
	})

	t.Run("HTTP error", func(t *testing.T) {
		p, _ := NewPicture("/nowhere", base)
		err := p.Load(nil, 100, "")
		require.Error(t, err)
	})

	t.Run("Load", func(t *testing.T) {
		assert := require.New(t)
		p, _ := NewPicture("/img", base)

		assert.Equal("", p.Encoded())

		err := p.Load(nil, 100, "")
		assert.NoError(err)

		assert.Equal([2]int{100, 75}, p.Size)
		assert.Equal("image/png", p.Type)

		header := []byte{137, 80, 78, 71, 13, 10, 26, 10}
		assert.Equal(header, p.Bytes()[0:8])
		assert.Equal("iVBORw0K", p.Encoded()[0:8])
	})
}

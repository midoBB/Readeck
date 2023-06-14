package img_test

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
	"github.com/stretchr/testify/assert"

	"github.com/readeck/readeck/pkg/img"
)

func assertXMLEqual(t *testing.T, expected, actual string) bool {
	eNode, err := xmlquery.Parse(strings.NewReader(expected))
	if err != nil {
		t.Error(err)
		t.Fail()
		return false
	}
	aNode, err := xmlquery.Parse(strings.NewReader(actual))
	if err != nil {
		t.Error(err)
		t.Fail()
		return false
	}

	return assert.Equal(t, eNode.OutputXML(true), aNode.OutputXML(true))
}

func TestSvgImage(t *testing.T) {
	t.Run("load image", func(t *testing.T) {
		tests := []struct {
			svg string
			err string
		}{
			{
				`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
				<svg xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" version="1.1">
				</svg>
				`,
				"",
			},
			{
				"<?xml",
				"XML syntax error on line 1: unexpected EOF",
			},
			{
				`<?xml version="1.0" standalone="yes"?>
				<!DOCTYPE test [ <!ENTITY xxe SYSTEM "file:///etc/hostname" > ]>
				<svg width="128px" height="128px" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1">
				  <text font-size="16" x="0" y="16">&xxe;</text>
				</svg>`,
				"XML syntax error on line 4: invalid character entity &xxe;",
			},
		}

		for i, test := range tests {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				im, err := img.NewSvgImage(strings.NewReader(test.svg))
				if test.err != "" {
					assert.EqualError(t, err, test.err)
					return
				}

				assert.Nil(t, err)
				assert.Equal(t, im.Format(), "svg")
				assert.Equal(t, im.ContentType(), "image/svg+xml")
				assert.Nil(t, im.Close())
				assert.Nil(t, im.SetFormat("jpeg"))
				assert.Nil(t, im.SetCompression(img.CompressionBest))
			})
		}
	})

	t.Run("image size", func(t *testing.T) {
		tests := []struct {
			svg string
			w   int
			h   int
		}{
			{
				`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
				<svg xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" version="1.1">
				</svg>
				`,
				300, 150,
			},
			{
				`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
				<svg xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" version="1.1"
				 width="1280" height="720">
				</svg>
				`,
				1280, 720,
			},
			{
				`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
				<svg xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" version="1.1"
				 width="100%" height="50%">
				</svg>
				`,
				100, 50,
			},
			{
				`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
				<svg xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" version="1.1"
				 width="100%" height="50">
				</svg>
				`,
				300, 150,
			},
			{
				`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
				<svg xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" version="1.1"
				 viewBox="0 0 200 300">
				</svg>
				`,
				200, 300,
			},
			{
				`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
				<svg xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" version="1.1"
				 width="400" viewBox="0 0 200 300">
				</svg>
				`,
				400, 600,
			},
			{
				`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
				<svg xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" version="1.1"
				 height="1200" viewBox="0 0 200 300">
				</svg>
				`,
				800, 1200,
			},
		}

		for i, test := range tests {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				im, err := img.NewSvgImage(strings.NewReader(test.svg))
				assert.Nil(t, err)
				assert.Equal(t, test.w, int(im.Width()))
				assert.Equal(t, test.h, int(im.Height()))
			})
		}
	})

	t.Run("resize", func(t *testing.T) {
		tests := []struct {
			svg      string
			w        int
			h        int
			expected string
		}{
			{
				`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
				<svg xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" version="1.1">
				</svg>`,
				500, 400,
				`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
				<svg xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" version="1.1"
				 width="500" height="400">
				</svg>`,
			},
			{
				`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
				<svg xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" version="1.1"
				 width="1280" height="720">
				</svg>`,
				500, 400,
				`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
				<svg xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" version="1.1"
				 width="500" height="400">
				</svg>`,
			},
			{
				`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
				<svg xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" version="1.1"
				 viewBox="0 0 100 200">
				</svg>`,
				500, 400,
				`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
				 <svg xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" version="1.1"
				  viewBox="0 0 100 200" width="500" height="400">
				</svg>`,
			},
		}

		for i, test := range tests {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				im, err := img.NewSvgImage(strings.NewReader(test.svg))
				assert.Nil(t, err)
				im.Resize(uint(test.w), uint(test.h))
				b := &bytes.Buffer{}
				im.Encode(b)
				assertXMLEqual(t, test.expected, b.String())
			})
		}
	})
}

func TestClean(t *testing.T) {
	tests := []struct {
		svg      string
		expected string
	}{
		{
			`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
			<svg xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" version="1.1">
			  <script type="text/ecmascript">
			    <![CDATA[
			      alert('Hax!');
			    ]]>
			  </script>
			</svg>`,
			`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
			<svg xmlns="http://www.w3.org/2000/svg" version="1.1">
			</svg>`,
		},
		{
			`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
			<svg xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1">
			  <defs id="defs4">
			    <circle id="my_circle" cx="100" cy="50" r="40" fill="red"/>
			  </defs>
			  <g>
			    <use href="#my_circle" x="20" y="20"/>
				<use xlink:href="#my_circle" x="50" y="50"/>
				<use href="http://example.net/test.svg#abc" x="20" y="20"/>
				<use xlink:href="http://example.net/test.svg#abc" x="20" y="20"/>
			  </g>
			</svg>`,
			`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
			<svg xmlns="http://www.w3.org/2000/svg" version="1.1">
			  <defs id="defs4">
			    <circle id="my_circle" cx="100" cy="50" r="40" fill="red"/>
			  </defs>
			  <g>
			    <use href="#my_circle" x="20" y="20"/>
				<use xlink:href="#my_circle" x="50" y="50"/>
			  </g>
			</svg>`,
		},
		{
			`
			<?xml version="1.0" encoding="UTF-8" standalone="no"?>
			<!-- Created with Inkscape (http://www.inkscape.org/) -->
			<svg xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:cc="http://creativecommons.org/ns#" xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:svg="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns="http://www.w3.org/2000/svg" version="1.1"
			 width="512" height="512" id="svg2">
			<title>Mostly harmless</title>
			<metadata id="metadata7">Some metadata</metadata>

			<script type="text/ecmascript">
			<![CDATA[
			alert('Hax!');
			]]>
			</script>
			<style type="text/css">
			<![CDATA[ svg{display:none} ]]>
			</style>

			<defs id="defs4">
				<circle id="my_circle" cx="100" cy="50" r="40" fill="red"/>
			</defs>

			<g id="layer1">
			<a xlink:href="www.hax.ru">
				<use xlink:href="#my_circle" x="20" y="20"/>
				<use xlink:href="#my_circle" x="100" y="50"/>
				<use xlink:href="http://example.net/test.svg#abc"/>
				<use href="http://example.net/test.svg#abc"/>
			</a>
			</g>
			<text>
				<tspan>It was the best of times</tspan>
				<tspan dx="-140" dy="15">It was the worst of times.</tspan>
			</text>
			</svg>
			`,
			`
			<?xml version="1.0" encoding="UTF-8" standalone="no"?>
			<!-- Created with Inkscape (http://www.inkscape.org/) -->
			<svg xmlns:cc="http://creativecommons.org/ns#" xmlns:svg="http://www.w3.org/2000/svg" xmlns="http://www.w3.org/2000/svg" version="1.1"
			 width="512" height="512" id="svg2">
			<title>Mostly harmless</title>
			<metadata id="metadata7">Some metadata</metadata>

			<style type="text/css">
			<![CDATA[ svg{display:none} ]]>
			</style>

			<defs id="defs4">
				<circle id="my_circle" cx="100" cy="50" r="40" fill="red"/>
			</defs>

			<g id="layer1">
			<a xlink:href="www.hax.ru">
				<use xlink:href="#my_circle" x="20" y="20"/>
				<use xlink:href="#my_circle" x="100" y="50"/>
			</a>
			</g>
			<text>
				<tspan>It was the best of times</tspan>
				<tspan dx="-140" dy="15">It was the worst of times.</tspan>
			</text>
			</svg>
			`,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			im, err := img.NewSvgImage(strings.NewReader(test.svg))
			assert.Nil(t, err)
			im.Clean()

			b := &bytes.Buffer{}
			im.Encode(b)
			assertXMLEqual(t, test.expected, b.String())
		})
	}
}

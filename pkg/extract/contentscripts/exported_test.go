// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package contentscripts_test

import (
	"strconv"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"

	"codeberg.org/readeck/readeck/pkg/extract"
	"codeberg.org/readeck/readeck/pkg/extract/contentscripts"
)

func TestExported(t *testing.T) {
	t.Run("processMessageProxy", func(t *testing.T) {
		tests := []struct {
			src      string
			value    func(value goja.Value, drop *extract.Drop) any
			expected any
		}{
			{
				`unescapeURL("https://api.example.net/oembed?url=https%3A%2F%2Fwww.example.net%2Ftest%2Fobject&format=json")`,
				func(value goja.Value, _ *extract.Drop) any {
					return value.Export()
				},
				"https://api.example.net/oembed?url=https://www.example.net/test/object&format=json",
			},
			{
				"$.domain",
				func(value goja.Value, drop *extract.Drop) any {
					return value.Export()
				},
				"example.net",
			},
			{
				"$.host",
				func(value goja.Value, drop *extract.Drop) any {
					return value.Export()
				},
				"host.example.net",
			},
			{
				"$.url",
				func(value goja.Value, _ *extract.Drop) any {
					return value.Export()
				},
				"https://host.example.net/",
			},
			{
				"$.authors",
				func(value goja.Value, _ *extract.Drop) any {
					return value.Export()
				},
				[]string{"jack"},
			},
			{
				`$.authors = ["john"]`,
				func(value goja.Value, d *extract.Drop) any {
					return d.Authors
				},
				[]string{"john"},
			},
			{
				"$.description",
				func(value goja.Value, _ *extract.Drop) any {
					return value.Export()
				},
				"content description",
			},
			{
				`$.description = "new description"`,
				func(value goja.Value, d *extract.Drop) any {
					return d.Description
				},
				"new description",
			},
			{
				"$.title",
				func(value goja.Value, _ *extract.Drop) any {
					return value.Export()
				},
				"content title",
			},
			{
				`$.title = "new title"`,
				func(value goja.Value, d *extract.Drop) any {
					return d.Title
				},
				"new title",
			},
			{
				"$.type",
				func(value goja.Value, _ *extract.Drop) any {
					return value.Export()
				},
				"article",
			},
			{
				`$.type = "video"`,
				func(value goja.Value, d *extract.Drop) any {
					return d.DocumentType
				},
				"video",
			},
			{
				`$.meta["test"]`,
				func(value goja.Value, _ *extract.Drop) any {
					return value.Export()
				},
				[]string{"test value"},
			},
			{
				`"test" in $.meta`,
				func(value goja.Value, _ *extract.Drop) any {
					return value.Export()
				},
				true,
			},
			{
				`"abc" in $.meta`,
				func(value goja.Value, _ *extract.Drop) any {
					return value.Export()
				},
				false,
			},
			{
				`$.meta["abc"] = ["xyz", 123]`,
				func(value goja.Value, d *extract.Drop) any {
					return d.Meta["abc"]
				},
				[]string{"xyz", "123"},
			},
			{
				`$.meta["abc"] = "xyz"`,
				func(value goja.Value, d *extract.Drop) any {
					return d.Meta["abc"]
				},
				[]string{"xyz"},
			},
			{
				`Object.keys($.meta)`,
				func(value goja.Value, _ *extract.Drop) any {
					return value.Export()
				},
				[]any{"test"},
			},
			{
				`delete($.meta["test"])`,
				func(value goja.Value, d *extract.Drop) any {
					return d.Meta["test"]
				},
				[]string(nil),
			},
			{
				`
				const xml = '<test><item id="a">T1</item><item id="b">T2</item></test>'
				decodeXML(xml)
				`,
				func(value goja.Value, _ *extract.Drop) any {
					return value.Export()
				},
				map[string]any{
					"test": map[string]any{
						"item": []map[string]any{
							{"#text": "T1", "@id": "a"},
							{"#text": "T2", "@id": "b"},
						},
					},
				},
			},
		}

		for i, test := range tests {
			t.Run(strconv.Itoa(i+1), func(t *testing.T) {
				extractor, _ := extract.New("https://host.example.net/")
				extractor.Drop().Meta["test"] = []string{"test value"}
				extractor.Drop().Authors = []string{"jack"}
				extractor.Drop().Description = "content description"
				extractor.Drop().Title = "content title"
				extractor.Drop().DocumentType = "article"

				pm := &extract.ProcessMessage{
					Extractor: extractor,
				}

				vm := contentscripts.New()
				vm.SetProcessMessage(pm)

				v, err := vm.RunProgram(testProgram("test", test.src))
				if assert.NoError(t, err) {
					assert.Equal(t, test.expected, test.value(v, extractor.Drop()))
				}
			})
		}
	})

	t.Run("processMessageProxy error", func(t *testing.T) {
		vm := contentscripts.New()
		_, err := vm.RunProgram(testProgram("test", `$.authors`))
		assert.ErrorContains(t, err, "no extractor")
	})

	t.Run("siteConfig", func(t *testing.T) {
		cf := contentscripts.SiteConfig{HTTPHeaders: map[string]string{}}

		vm := contentscripts.New()
		vm.Set("config", &cf)

		_, err := vm.RunProgram(testProgram("test", `
			config.titleSelectors.push("//title", "//main//h1")
			config.httpHeaders["user-agent"] = "curl/7"
		`))
		assert.NoError(t, err)
		assert.Equal(t, cf.TitleSelectors, []string{"//title", "//main//h1"})
		assert.Equal(t, cf.HTTPHeaders, map[string]string{"user-agent": "curl/7"})
	})
}

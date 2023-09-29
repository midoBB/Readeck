// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package contentscripts_test

import (
	"net/url"
	"testing"
	"testing/fstest"

	"codeberg.org/readeck/readeck/pkg/extract/contentscripts"
	"github.com/stretchr/testify/assert"
)

func TestSiteConfig(t *testing.T) {
	t.Run("find config", func(t *testing.T) {
		root := fstest.MapFS{
			"global.json": &fstest.MapFile{
				Data: []byte("{}"),
			},
			"example.net.json": &fstest.MapFile{
				Data: []byte("{}"),
			},
			"test.example.net.json": &fstest.MapFile{
				Data: []byte("{}"),
			},
			".example.net.json": &fstest.MapFile{
				Data: []byte("{}"),
			},
			".net.json": &fstest.MapFile{
				Data: []byte("{}"),
			},
			"xn--protin-bva.com.json": &fstest.MapFile{
				Data: []byte("{}"),
			},
		}
		d := contentscripts.NewSiteconfigDiscovery(root)

		tests := []struct {
			host  string
			files []string
		}{
			{"example.net", []string{"example.net.json", ".example.net.json", ".net.json", "global.json"}},
			{"test.example.net", []string{"test.example.net.json", ".example.net.json", ".net.json", "global.json"}},
			{"site.net", []string{".net.json", "global.json"}},
			{"example.com", []string{"global.json"}},
			{"pérotin.com", []string{"xn--protin-bva.com.json", "global.json"}},
			{"xn--protin-bva.com", []string{"xn--protin-bva.com.json", "global.json"}},
		}

		for _, test := range tests {
			t.Run(test.host, func(t *testing.T) {
				u := &url.URL{Host: test.host}
				cf, err := contentscripts.NewConfigForURL(d, u)
				assert.NoError(t, err)
				assert.Equal(t, test.files, cf.Files())
			})
		}
	})

	t.Run("merge config", func(t *testing.T) {
		cf := &contentscripts.SiteConfig{
			BodySelectors: []string{"//div[@id='content']"},
			Prune:         true,
			HTTPHeaders: map[string]string{
				"x-test": "abc",
			},
		}
		cf.Merge(&contentscripts.SiteConfig{
			BodySelectors: []string{"//div[@id='page']"},
			Prune:         false,
			HTTPHeaders: map[string]string{
				"x-test": "123",
				"x-v":    "abc",
			},
		})

		assert.Equal(t, &contentscripts.SiteConfig{
			BodySelectors: []string{"//div[@id='content']", "//div[@id='page']"},
			Prune:         false,
			HTTPHeaders: map[string]string{
				"x-test": "123",
				"x-v":    "abc",
			},
		}, cf)
	})

	t.Run("autodetect_on_failure", func(t *testing.T) {
		root := fstest.MapFS{
			"global.json": &fstest.MapFile{
				Data: []byte("{}"),
			},
			".example.net.json": &fstest.MapFile{
				Data: []byte("{}"),
			},
			"test1.example.net.json": &fstest.MapFile{
				Data: []byte("{}"),
			},
			"test2.example.net.json": &fstest.MapFile{
				Data: []byte(`{"autodetect_on_failure": false}`),
			},
		}
		d := contentscripts.NewSiteconfigDiscovery(root)

		tests := []struct {
			host  string
			files []string
		}{
			{"test1.example.net", []string{"test1.example.net.json", ".example.net.json", "global.json"}},
			{"test2.example.net", []string{"test2.example.net.json"}},
		}

		for _, test := range tests {
			t.Run(test.host, func(t *testing.T) {
				u := &url.URL{Host: test.host}
				cf, err := contentscripts.NewConfigForURL(d, u)
				assert.NoError(t, err)
				assert.Equal(t, test.files, cf.Files())
			})
		}
	})

	t.Run("parse error", func(t *testing.T) {
		root := fstest.MapFS{
			"example.net.json": &fstest.MapFile{
				Data: []byte("{test"),
			},
		}
		d := contentscripts.NewSiteconfigDiscovery(root)
		u := &url.URL{Host: "example.net"}

		cf, err := contentscripts.NewConfigForURL(d, u)
		assert.Error(t, err)
		assert.Nil(t, cf)
	})
}

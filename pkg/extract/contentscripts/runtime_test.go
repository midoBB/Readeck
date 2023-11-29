// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package contentscripts_test

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"

	"codeberg.org/readeck/readeck/pkg/extract"
	"codeberg.org/readeck/readeck/pkg/extract/contentscripts"
)

func testProgram(name string, src string) *contentscripts.Program {
	p, err := goja.Compile(name, src, true)
	if err != nil {
		panic(err)
	}
	return &contentscripts.Program{
		Program: p,
		Name:    name,
	}
}

func TestRuntime(t *testing.T) {
	t.Run("setConfig", func(t *testing.T) {
		vm, _ := contentscripts.New()

		err := vm.AddScript("1", strings.NewReader(`
		exports.isActive = function() { return true }

		exports.setConfig = function(config) {
			config.titleSelectors = ["/html/head/title"]
			config.bodySelectors = ["//body"]
		}
		`))

		assert := require.New(t)
		assert.NoError(err)

		err = vm.AddScript("2", strings.NewReader(`
		exports.isActive = function() { return true }

		exports.setConfig = function(config) {
			config.bodySelectors.push("//body/div")
		}
		`))
		assert.NoError(err)

		err = vm.AddScript("3", strings.NewReader(""))
		assert.NoError(err)

		cf := &contentscripts.SiteConfig{}
		_ = vm.SetConfig(cf)

		assert.Equal([]string{"/html/head/title"}, cf.TitleSelectors)
		assert.Equal([]string{"//body", "//body/div"}, cf.BodySelectors)
	})

	t.Run("processMeta", func(t *testing.T) {
		extractor, _ := extract.New("https://example.net/")
		pm := &extract.ProcessMessage{
			Extractor: extractor,
		}

		vm, _ := contentscripts.New()
		vm.SetProcessMessage(pm)

		err := vm.AddScript("1", strings.NewReader(`
		exports.isActive = function() { return true }

		exports.processMeta = function() {
			$.meta["script.name"] = __name__
		}
		`))

		assert := require.New(t)
		assert.NoError(err)

		err = vm.ProcessMeta()
		assert.NoError(err)

		assert.Equal([]string{"1"}, pm.Extractor.Drop().Meta["script.name"])
	})

	t.Run("error list", func(t *testing.T) {
		extractor, _ := extract.New("https://example.net/")
		pm := &extract.ProcessMessage{
			Extractor: extractor,
		}

		vm, _ := contentscripts.New()
		vm.SetProcessMessage(pm)

		err := vm.AddScript("1", strings.NewReader(`
		exports.isActive = function() { return true }

		exports.processMeta = function() {
			throw new Error("script 1")
		}
		`))

		assert := require.New(t)
		assert.NoError(err)

		err = vm.AddScript("2", strings.NewReader(`
		exports.isActive = function() { return true }

		exports.processMeta = function() {
			throw new Error("script 2")
		}
		`))
		assert.NoError(err)

		err = vm.ProcessMeta()
		assert.Error(err)
		assert.ErrorContains(err, "script 1")
		assert.ErrorContains(err, "script 2")
	})
}

// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package contentscripts_test

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"

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
		vm := contentscripts.New()

		err := vm.AddScript("1", strings.NewReader(`
		exports.isActive = function() { return true }

		exports.setConfig = function(config) {
			config.titleSelectors = ["/html/head/title"]
			config.bodySelectors = ["//body"]
		}
		`))
		assert.NoError(t, err)

		err = vm.AddScript("2", strings.NewReader(`
		exports.isActive = function() { return true }

		exports.setConfig = function(config) {
			config.bodySelectors.push("//body/div")
		}
		`))
		assert.NoError(t, err)

		err = vm.AddScript("3", strings.NewReader(""))
		assert.NoError(t, err)

		cf := &contentscripts.SiteConfig{}
		vm.SetConfig(cf)

		assert.Equal(t, []string{"/html/head/title"}, cf.TitleSelectors)
		assert.Equal(t, []string{"//body", "//body/div"}, cf.BodySelectors)
	})

	t.Run("processMeta", func(t *testing.T) {
		extractor, _ := extract.New("https://example.net/")
		pm := &extract.ProcessMessage{
			Extractor: extractor,
		}

		vm := contentscripts.New()
		vm.SetProcessMessage(pm)

		err := vm.AddScript("1", strings.NewReader(`
		exports.isActive = function() { return true }

		exports.processMeta = function() {
			$.meta["script.name"] = __name__
		}
		`))
		assert.NoError(t, err)

		err = vm.ProcessMeta()
		assert.NoError(t, err)

		assert.Equal(t, []string{"1"}, pm.Extractor.Drop().Meta["script.name"])
	})

	t.Run("error list", func(t *testing.T) {
		extractor, _ := extract.New("https://example.net/")
		pm := &extract.ProcessMessage{
			Extractor: extractor,
		}

		vm := contentscripts.New()
		vm.SetProcessMessage(pm)

		err := vm.AddScript("1", strings.NewReader(`
		exports.isActive = function() { return true }

		exports.processMeta = function() {
			throw new Error("script 1")
		}
		`))
		assert.NoError(t, err)

		err = vm.AddScript("2", strings.NewReader(`
		exports.isActive = function() { return true }

		exports.processMeta = function() {
			throw new Error("script 2")
		}
		`))
		assert.NoError(t, err)

		err = vm.ProcessMeta()
		assert.Error(t, err)
		assert.ErrorContains(t, err, "script 1")
		assert.ErrorContains(t, err, "script 2")
	})
}

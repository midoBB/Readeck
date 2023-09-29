// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bookmarks

import (
	"io/fs"
	"os"
	"path"

	log "github.com/sirupsen/logrus"

	"codeberg.org/readeck/readeck/configs"
	"codeberg.org/readeck/readeck/pkg/extract/contentscripts"
)

var contentScriptRegistry = []*contentscripts.Program{}

func loadContentScripts(logger *log.Logger) []*contentscripts.Program {
	res := []*contentscripts.Program{}
	for _, root := range configs.Config.Extractor.ContentScripts {
		rootFS := os.DirFS(root)
		err := fs.WalkDir(rootFS, ".", func(name string, x fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if x.IsDir() || path.Ext(name) != ".js" {
				return nil
			}

			fd, err := rootFS.Open(name)
			if err != nil {
				logger.WithError(err).Error("content script")
				return nil
			}
			defer fd.Close()

			p, err := contentscripts.NewProgram(path.Join(root, name), fd)
			if err != nil {
				logger.WithError(err).Error("content script")
				return nil
			}
			res = append(res, p)
			return nil
		})
		if err != nil {
			logger.WithError(err).Error("content script")
		}
	}

	return res
}

// LoadContentScripts loads the content scripts when Readeck is not
// configured in dev mode.
// In dev mode, scripts are reloaded on each extraction.
func LoadContentScripts() {
	if !configs.Config.Main.DevMode {
		contentScriptRegistry = loadContentScripts(log.New())
	}
}

// GetContentScripts returns the compiled content scripts, either from
// the cache or by browsing the configured folders.
func GetContentScripts(logger *log.Logger) []*contentscripts.Program {
	if configs.Config.Main.DevMode {
		return loadContentScripts(logger)
	}
	return contentScriptRegistry
}

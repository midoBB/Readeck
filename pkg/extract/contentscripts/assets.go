// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package contentscripts

import (
	"embed"
	"io/fs"
	"path"
)

//go:embed assets assets/**/*
var assets embed.FS

var (
	// SiteConfigFiles is the default site-config files discovery.
	SiteConfigFiles *SiteConfigDiscovery

	siteConfigFS     fs.FS
	preloadedScripts []*Program
)

func init() {
	var err error
	if siteConfigFS, err = fs.Sub(assets, "assets/site-config"); err != nil {
		panic(err)
	}
	SiteConfigFiles = NewSiteconfigDiscovery(siteConfigFS)

	// Preload scripts
	scripts, err := fs.Glob(assets, "assets/scripts/*.js")
	if err != nil {
		panic(err)
	}

	for _, x := range scripts {
		r, err := assets.Open(x)
		if err != nil {
			panic(err)
		}
		p, err := NewProgram(path.Join("builtin", path.Base(x)), r)
		if err != nil {
			panic(err)
		}
		preloadedScripts = append(preloadedScripts, p)
	}
}

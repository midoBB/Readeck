// SPDX-FileCopyrightText: © 2020 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package fftr

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed site-config site-config/**/*
var assets embed.FS

// siteConfigFS returns the given site-config subfolder as an fs.FS instance.
func siteConfigFS(name string) fs.FS {
	sub, err := fs.Sub(assets, fmt.Sprintf("site-config/%s", name))
	if err != nil {
		panic(err)
	}
	return sub
}

// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package migrations

import (
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/go-shiori/dom"
	"golang.org/x/net/html"

	"codeberg.org/readeck/readeck/configs"
	"codeberg.org/readeck/readeck/pkg/zipfs"
)

// MigrateBookmarkIDs performs a migration inside bookmark zip files
// It changes the id and href attributes to conform to what the
// extractor does from now on.
func MigrateBookmarkIDs(_ *goqu.TxDatabase, _ fs.FS) error {
	// Find all zip files
	p := filepath.Join(configs.Config.Main.DataDirectory, "bookmarks")

	err := filepath.Walk(p, bookmarkFileMigrateIDs)
	if err != nil {
		// Remove all .zip.tmp files
		files, _ := filepath.Glob(filepath.Join(p, "**/*.zip.tmp"))
		for _, x := range files {
			os.Remove(x)
		}
	}

	return err
}

func bookmarkFileMigrateIDs(path string, info fs.FileInfo, e error) (err error) {
	if e != nil {
		return e
	}
	if info.IsDir() {
		return nil
	}

	if filepath.Ext(path) != ".zip" {
		return nil
	}

	z := zipfs.NewZipRW(nil, nil, 0)
	if err = z.AddSourceFile(path); err != nil {
		return err
	}
	defer z.Close()

	fd, err := z.Source().Open("index.html")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// no index.html, stop here
			return nil
		}
		return err
	}
	defer fd.Close()

	top, err := html.Parse(fd)
	if err != nil {
		return err
	}

	changed := setIDs(top)
	if changed == 0 {
		// no need to change the zip file, stop here
		return nil
	}

	// Create the new zip file
	dest := fmt.Sprintf("%s.tmp", path)
	if err = z.AddDestFile(dest); err != nil {
		return err
	}

	for _, x := range z.SrcFiles() {
		if x.Name == "index.html" {
			if err = z.Add(&x.FileHeader, strings.NewReader(dom.OuterHTML(top))); err != nil {
				return err
			}
			continue
		}

		if err = z.Copy(x.Name); err != nil {
			return err
		}
	}

	if err = z.Close(); err != nil {
		return
	}
	return os.Rename(dest, path)
}

func setIDs(top *html.Node) int {
	// Set a random prefix for the whole document
	chars := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
	rand.Shuffle(len(chars), func(i, j int) {
		chars[i], chars[j] = chars[j], chars[i]
	})
	prefix := fmt.Sprintf("%s.%s", chars[0:2], chars[3:7])

	total := 0

	// Update all nodes with an id attribute
	for _, node := range dom.QuerySelectorAll(top, "[id]") {
		if value := dom.GetAttribute(node, "id"); value != "" {
			dom.SetAttribute(node, "id", fmt.Sprintf("%s.%s", prefix, value))
			total++
		}
	}

	// Update all a[name], because we'll update the href="#..." later
	for _, node := range dom.QuerySelectorAll(top, "a[name]") {
		if value := dom.GetAttribute(node, "name"); value != "" {
			dom.SetAttribute(node, "name", fmt.Sprintf("%s.%s", prefix, value))
			total++
		}
	}

	// Update all nodes with an href attribute starting with "#"
	for _, node := range dom.QuerySelectorAll(top, "[href^='#']") {
		if value := strings.TrimPrefix(dom.GetAttribute(node, "href"), "#"); value != "" {
			dom.SetAttribute(node, "href", fmt.Sprintf("#%s.%s", prefix, value))
			total++
		}
	}

	return total
}

// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/cristalhq/acmd"
	"github.com/doug-martin/goqu/v9"

	"codeberg.org/readeck/readeck/internal/bookmarks"
)

func init() {
	commands = append(commands, acmd.Command{
		Name:        "cleanup",
		Description: "Remove loading bookmarks",
		ExecFunc:    runCleanup,
	})
}

func runCleanup(_ context.Context, args []string) error {
	var flags appFlags
	if err := flags.Flags().Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	// Init application
	if err := appPreRun(&flags); err != nil {
		return err
	}
	defer appPostRun()

	println("⚙️ removing loading bookmarks")
	if err := removeLoadingBookmarks(); err != nil {
		return err
	}

	println("⚙️ removing orphan files")
	return removeOrphanFiles()
}

func removeLoadingBookmarks() error {
	var items []*bookmarks.Bookmark
	ds := bookmarks.Bookmarks.Query().Where(
		goqu.C("state").Eq(bookmarks.StateLoading),
	)
	if err := ds.ScanStructs(&items); err != nil {
		return err
	}

	if len(items) == 0 {
		println("  ⭐ all good!")
		return nil
	}

	for _, b := range items {
		l := slog.With(
			slog.String("id", b.UID),
			slog.String("title", b.Title),
		)
		if err := b.Delete(); err != nil {
			l.Error("deleting bookmarks", slog.Any("err", err))
			continue
		}
		l.Info("bookmark deleted")
	}

	return nil
}

func removeOrphanFiles() error {
	dirs, err := filepath.Glob(filepath.Join(bookmarks.StoragePath(), "*/*.zip"))
	if err != nil {
		return err
	}
	i := 0
	for _, x := range dirs {
		bookmarkID := strings.TrimSuffix(filepath.Base(x), ".zip")
		_, err := bookmarks.Bookmarks.GetOne(goqu.C("uid").Eq(bookmarkID))
		if err == nil {
			continue
		}
		if !errors.Is(err, bookmarks.ErrBookmarkNotFound) {
			return err
		}

		l := slog.With(
			slog.String("file", x),
		)
		b := &bookmarks.Bookmark{
			UID: bookmarkID,
		}
		u, _ := b.GetBaseFileURL()
		b.FilePath = u
		b.RemoveFiles()
		l.Info("file removed")

		i++
	}

	if i > 0 {
		fmt.Printf("  ❌ %d file(s) removed\n", i)
	} else {
		println("  ⭐ no orphan files")
	}

	return nil
}

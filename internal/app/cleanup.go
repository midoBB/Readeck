// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"flag"
	"log/slog"

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

	var items []*bookmarks.Bookmark
	ds := bookmarks.Bookmarks.Query().Where(
		goqu.C("state").Eq(bookmarks.StateLoading),
	)
	if err := ds.ScanStructs(&items); err != nil {
		return err
	}

	if len(items) == 0 {
		println("⭐ all good!")
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

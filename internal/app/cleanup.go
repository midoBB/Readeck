// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"flag"

	"github.com/cristalhq/acmd"
	"github.com/doug-martin/goqu/v9"
	log "github.com/sirupsen/logrus"

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
		l := log.WithField("id", b.UID).WithField("title", b.Title)
		if err := b.Delete(); err != nil {
			l.WithError(err).Error("deleting bookmarks")
			continue
		}
		l.Info("bookmark deleted")
	}

	return nil
}

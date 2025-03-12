// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package migrations

import (
	"database/sql/driver"
	"encoding/json"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/doug-martin/goqu/v9"

	"codeberg.org/readeck/readeck/configs"
	"codeberg.org/readeck/readeck/internal/db/types"
	"codeberg.org/readeck/readeck/pkg/base58"
)

type m16BbookmarkAnnotations []*m16BookmarkAnnotation

// m16BookmarkAnnotation is an annotation that can be serialized in a database JSON column.
type m16BookmarkAnnotation struct {
	ID            string    `json:"id"`
	StartSelector string    `json:"start_selector"`
	StartOffset   int       `json:"start_offset"`
	EndSelector   string    `json:"end_selector"`
	EndOffset     int       `json:"end_offset"`
	Color         string    `json:"color"`
	Created       time.Time `json:"created"`
	Text          string    `json:"text"`
}

// Scan loads a BookmarkAnnotations instance from a column.
func (a *m16BbookmarkAnnotations) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	v, err := types.JSONBytes(value)
	if err != nil {
		return err
	}
	json.Unmarshal(v, a) //nolint:errcheck
	return nil
}

// Value encodes a BookmarkAnnotations instance for storage.
func (a m16BbookmarkAnnotations) Value() (driver.Value, error) {
	for _, x := range a {
		if x.Color == "" {
			x.Color = "yellow"
		}
	}

	v, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	return string(v), nil
}

// M16uuidFields performs a migration that updates all short UUIDs to
// base-58 encoded ones. It then renames all files in the data/bookmarks folder
// so they match the new IDs.
func M16uuidFields(db *goqu.TxDatabase, _ fs.FS) error {
	type postUpdate func(table string, ids []int) error
	var updateTable func(table string, column string, fn postUpdate) error

	rootPath := filepath.Join(configs.Config.Main.DataDirectory, "bookmarks")

	type bookmarkFile struct {
		ID          int                     `db:"id"`
		UID         string                  `db:"uid"`
		FilePath    string                  `db:"file_path"`
		Annotations m16BbookmarkAnnotations `db:"annotations"`
	}

	renameBookmarkFile := func(table string, ids []int) error {
		var records []bookmarkFile

		ds := db.Select().From(table).Where(goqu.C("id").In(ids))
		if err := ds.ScanStructs(&records); err != nil {
			return err
		}

		// Update annotations
		for _, x := range records {
			if len(x.Annotations) == 0 {
				continue
			}

			for i := range x.Annotations {
				x.Annotations[i].ID = base58.NewUUID()
			}
			if _, err := db.Update(table).Set(map[string]any{
				"annotations": x.Annotations,
			}).Where(goqu.C("id").Eq(x.ID)).Executor().Exec(); err != nil {
				return err
			}
		}

		// Update file_path
		pathCases := goqu.Case()
		nameMap := map[string]string{}

		for _, x := range records {
			newPath := filepath.Join(x.UID[0:2], x.UID)
			pathCases = pathCases.When(goqu.C("id").Eq(x.ID), newPath)
			nameMap[x.FilePath] = newPath
		}

		if _, err := db.Update(table).Set(goqu.Record{"file_path": pathCases}).Executor().Exec(); err != nil {
			return err
		}

		// Move files
		for oldPath, newPath := range nameMap {
			oldPath = filepath.Join(rootPath, oldPath+".zip")
			newPath = filepath.Join(rootPath, newPath+".zip")
			slog.Debug("moving files",
				slog.String("old", oldPath),
				slog.String("new", newPath),
			)

			// Move file
			if err := os.MkdirAll(filepath.Dir(newPath), 0o750); err != nil {
				return err
			}
			if err := os.Rename(oldPath, newPath); err != nil {
				return err
			}

			// Remove old parent directory (will err if not empty)
			_ = os.Remove(filepath.Dir(oldPath))
		}

		return nil
	}

	updateTable = func(table, column string, fn postUpdate) error {
		ds, err := db.Select(goqu.C("id"), goqu.C(column)).From(table).Executor().Query()
		if err != nil {
			return err
		}

		c := goqu.Case()
		ids := []int{}

		for ds.Next() {
			var id int
			var uid string
			if err := ds.Scan(&id, &uid); err != nil {
				return err
			}
			newUID := base58.NewUUID()
			c = c.When(goqu.C("id").Eq(id), newUID)
			ids = append(ids, id)
		}

		// Stop when nothing needs updating
		if len(c.GetWhens()) == 0 {
			return nil
		}

		if _, err := db.Update(table).Set(goqu.Record{column: c}).Executor().Exec(); err != nil {
			return err
		}

		if fn == nil {
			return nil
		}

		return fn(table, ids)
	}

	tables := []struct {
		name   string
		column string
		fn     postUpdate
	}{
		{"token", "uid", nil},
		{"credential", "uid", nil},
		{"bookmark_collection", "uid", nil},
		{"bookmark", "uid", renameBookmarkFile},
	}

	for _, t := range tables {
		if err := updateTable(t.name, t.column, t.fn); err != nil {
			return err
		}
	}

	return nil
}

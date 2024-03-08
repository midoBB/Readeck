// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package migrations

import (
	"io/fs"

	"codeberg.org/readeck/readeck/internal/db/types"
	"github.com/doug-martin/goqu/v9"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// M11sortLabels sorts all the labels in bookmarks, according
// to the new collation rules.
func M11sortLabels(db *goqu.TxDatabase, _ fs.FS) error {
	ds := db.Select(goqu.C("id"), goqu.C("labels")).From("bookmark")

	bookmarkList := []struct {
		ID     int           `db:"id"`
		Labels types.Strings `db:"labels"`
	}{}

	err := ds.ScanStructs(&bookmarkList)
	if err != nil {
		return err
	}

	unicodeCollate := collate.New(language.Und, collate.Loose, collate.Numeric)
	ids := []int{}
	cases := goqu.Case()
	casePlaceholder := "?"
	if db.Dialect() == "postgres" {
		casePlaceholder = "?::jsonb"
	}

	for _, x := range bookmarkList {
		if len(x.Labels) == 0 {
			continue
		}
		labels := make(types.Strings, len(x.Labels))
		copy(labels, x.Labels)
		unicodeCollate.SortStrings(labels)

		cases = cases.When(goqu.C("id").Eq(x.ID), goqu.L(casePlaceholder, labels))
		ids = append(ids, x.ID)
	}

	_, err = db.Update("bookmark").Prepared(true).
		Set(goqu.Record{"labels": cases}).
		Where(goqu.C("id").In(ids)).
		Executor().Exec()
	return err
}

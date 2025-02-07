// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package migrations

import (
	"io/fs"
	"time"

	"github.com/araddon/dateparse"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

// M15SqliteDates performs a migration to update all datetime
// formats with SQLite.
// We can't only use a SQL script since we want to fix incorrectly set
// dates. These wront dates most likely come from use of the non CGO sqlite
// driver (modernc).
func M15SqliteDates(db *goqu.TxDatabase, _ fs.FS) error {
	if db.Dialect() != "sqlite3" {
		return nil
	}

	updateDates := func(table string, columns ...string) error {
		c := make([]any, len(columns)+1)
		c[0] = goqu.C("id")
		for i := range columns {
			c[i+1] = goqu.C(columns[i])
		}
		ds, err := db.Select(c...).From(table).Executor().Query()
		if err != nil {
			return err
		}

		cases := map[string]exp.CaseExpression{}
		for _, name := range columns {
			cases[name] = goqu.Case()
		}

		for ds.Next() {
			var id int
			scanRes := make([]any, len(columns)+1)
			scanRes[0] = &id
			values := make([]*string, len(columns))
			for i := range values {
				scanRes[i+1] = &values[i]
			}
			if err := ds.Scan(scanRes...); err != nil {
				return err
			}
			for i, name := range columns {
				if values[i] == nil {
					continue
				}
				dt, err := dateparse.ParseAny(*values[i])
				if err != nil {
					return err
				}

				// Skip properly formated dates
				v := dt.Format(time.RFC3339Nano)
				if v == *values[i] {
					continue
				}

				cases[name] = cases[name].When(
					goqu.C("id").Eq(id),
					v,
				)
			}
		}

		// Build a record for non empty cases
		records := goqu.Record{}
		for k, c := range cases {
			if len(c.GetWhens()) == 0 {
				continue
			}
			records[k] = c.Else(goqu.C(k))
		}

		if len(records) == 0 {
			// nothing to do here
			return nil
		}

		_, err = db.Update(table).Set(records).Executor().Exec()
		if err != nil {
			return err
		}

		return nil
	}

	type table struct {
		name string
		cols []string
	}
	tables := []table{
		{"migration", []string{"applied"}},
		{"user", []string{"created", "updated"}},
		{"token", []string{"created", "expires"}},
		{"credential", []string{"created"}},
		{"bookmark", []string{"created", "updated", "published"}},
		{"bookmark_collection", []string{"created", "updated"}},
	}

	for _, t := range tables {
		err := updateDates(t.name, t.cols...)
		if err != nil {
			return err
		}

	}

	return nil
}

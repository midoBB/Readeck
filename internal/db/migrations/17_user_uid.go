// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package migrations

import (
	"io/fs"

	"github.com/doug-martin/goqu/v9"

	"codeberg.org/readeck/readeck/pkg/base58"
)

// M17useruid adds a "uid" column to the "user" table.
// It creates the new UID values for existing users and then
// adds the UNIQUE constraint to the column.
func M17useruid(db *goqu.TxDatabase, _ fs.FS) error {
	var sql string

	// 1. create the new column
	switch db.Dialect() {
	case "sqlite3":
		sql = `ALTER TABLE "user" ADD COLUMN uid TEXT NOT NULL DEFAULT ''`
	case "postgres":
		sql = `ALTER TABLE "user" ADD COLUMN uid varchar(32) NOT NULL DEFAULT ''`
	}

	if _, err := db.Exec(sql); err != nil {
		return err
	}

	// 2. create the new uids
	ds, err := db.Select(goqu.C("id")).From("user").Executor().Query()
	if err != nil {
		return err
	}
	cases := goqu.Case()
	for ds.Next() {
		var id int
		if err := ds.Scan(&id); err != nil {
			return err
		}
		cases = cases.When(goqu.C("id").Eq(id), base58.NewUUID())
	}
	if len(cases.GetWhens()) > 0 {
		if _, err := db.Update("user").Set(goqu.Record{
			"uid": cases,
		}).Executor().Exec(); err != nil {
			return err
		}
	}

	// 3. make the column unique
	switch db.Dialect() {
	case "sqlite3":
		sql = `CREATE UNIQUE INDEX idx_user_uid ON 'user' (uid)`
	case "postgres":
		sql = `CREATE UNIQUE INDEX user_uid_key ON "user" (uid); ALTER TABLE "user" ALTER COLUMN uid DROP DEFAULT`
	}
	if _, err := db.Exec(sql); err != nil {
		return err
	}

	return nil
}

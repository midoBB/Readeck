// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package db

import (
	"github.com/doug-martin/goqu/v9"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// InsertWithID executes an insert statement and returns the value
// of the field given by "r".
// Depending on the database, it uses different ways to do just that.
func InsertWithID(stmt *goqu.InsertDataset, r string) (id int, err error) {
	if Driver().Dialect() == "postgres" {
		_, err = stmt.Returning(goqu.C(r)).Executor().ScanVal(&id)
		return
	}
	res, err := stmt.Executor().Exec()
	if err != nil {
		return id, err
	}

	i, _ := res.LastInsertId()
	id = int(i)

	return
}

var unicodeCollate = collate.New(language.Und, collate.Loose, collate.Numeric)

// UnaccentCompare performs a string comparison after removing accents.
var UnaccentCompare = unicodeCollate.CompareString

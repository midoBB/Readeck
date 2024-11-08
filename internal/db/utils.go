// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package db

import (
	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

var unicodeCollate = collate.New(language.Und, collate.Loose, collate.Numeric)

// UnaccentCompare performs a string comparison after removing accents.
var UnaccentCompare = unicodeCollate.CompareString

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

// BooleanExpresion returns the provided [exp.Expression] or its negation when
// "value" is false.
func BooleanExpresion(expr exp.Expression, value bool) exp.Expression {
	if value {
		return expr
	}
	return goqu.Func("NOT", expr)
}

// JSONArrayLength returns a json(b)_array_length statement of the given identifier.
func JSONArrayLength(identifier exp.IdentifierExpression) exp.SQLFunctionExpression {
	switch Driver().Dialect() {
	case "postgres":
		return goqu.Func(
			"jsonb_array_length",
			goqu.Case().
				When(goqu.Func("jsonb_typeof", identifier).Eq("array"), identifier).
				Else(goqu.V("[]")),
		)
	case "sqlite3":
		return goqu.Func(
			"json_array_length",
			goqu.Case().
				When(goqu.Func("json_valid", identifier), identifier).
				Else(goqu.V("[]")),
		)
	}

	return nil
}

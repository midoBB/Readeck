// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// Package filters provides query filters for specific operations.
package filters

import (
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

// JSONListFilter appends filters on list value to an existing dataset.
// It adds statements in order to find rows with JSON arrays containing the
// given expressions.
// Supported comparaisons are "Eq", "Neq", "Like" and "NotLike"
//
//	JSONListFilter(ds, goqu.T("books").C("tags").Eq("fiction"), goqu.T("books").C("tags").Neq("space"))
func JSONListFilter(ds *goqu.SelectDataset, expressions ...exp.BooleanExpression) *goqu.SelectDataset {
	if len(expressions) == 0 {
		return ds
	}

	res := goqu.And()

	switch dialect := ds.Dialect().Dialect(); dialect {
	case "postgres":
		for _, e := range expressions {
			col := e.LHS()
			cmp := "eq"
			op := "EXISTS"

			from := goqu.Dialect(dialect).Select(goqu.C("value")).From(goqu.Func(
				"jsonb_array_elements_text",
				goqu.Case().Value(goqu.Func("jsonb_typeof", col)).
					When(goqu.L("'array'"), col).
					Else(goqu.L("'[]'")),
			))

			switch e.Op() {
			case exp.LikeOp:
				cmp = "ilike"
			case exp.NotLikeOp:
				cmp = "ilike"
				op = "NOT EXISTS"
			case exp.NeqOp:
				op = "NOT EXISTS"
			}

			res = res.Append(goqu.L("? ?", goqu.L(op),
				from.Where(goqu.Ex{"value": goqu.Op{cmp: e.RHS()}}),
			))
		}
	case "sqlite3":
		for _, e := range expressions {
			col := e.LHS()
			cmp := "eq"
			op := "EXISTS"

			from := goqu.Dialect(dialect).From(goqu.Func(
				"json_each",
				goqu.Case().Value(goqu.Func("json_type",
					goqu.Case().Value(goqu.Func("json_valid", col)).
						When(goqu.L("true"), col).
						Else(goqu.L("'[]'")),
				)).
					When(goqu.L("'array'"), col).
					Else(goqu.L("'[]'")),
			))

			switch e.Op() {
			case exp.LikeOp:
				cmp = "like"
			case exp.NotLikeOp:
				cmp = "like"
				op = "NOT EXISTS"
			case exp.NeqOp:
				op = "NOT EXISTS"
			}

			res = res.Append(
				goqu.L("? ?", goqu.L(op),
					from.Where(goqu.Ex{"json_each.value": goqu.Op{cmp: e.RHS()}})),
			)
		}
	}

	return ds.Where(res)
}

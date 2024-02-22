// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package filters_test

import (
	"fmt"
	"testing"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/stretchr/testify/require"

	"codeberg.org/readeck/readeck/internal/db/filters"
)

type queryExpect struct {
	sql  string
	args []interface{}
}

func TestStringsFilter(t *testing.T) {
	tests := []struct {
		expressions []exp.BooleanExpression
		expected    map[string]queryExpect
	}{
		{
			[]exp.BooleanExpression{},
			map[string]queryExpect{
				"sqlite3": {
					"SELECT * FROM `T`",
					[]interface{}{},
				},
				"postgres": {
					`SELECT * FROM "T"`,
					[]interface{}{},
				},
			},
		},
		{
			[]exp.BooleanExpression{goqu.I("T.tags").Eq("test")},
			map[string]queryExpect{
				"sqlite3": {
					"SELECT * FROM `T` WHERE EXISTS (SELECT * FROM json_each(CASE json_type(CASE json_valid(`T`.`tags`) WHEN true THEN `T`.`tags` ELSE '[]' END) WHEN 'array' THEN `T`.`tags` ELSE '[]' END) WHERE (`json_each`.`value` = ?))",
					[]interface{}{"test"},
				},
				"postgres": {
					`SELECT * FROM "T" WHERE EXISTS (SELECT "value" FROM jsonb_array_elements_text(CASE jsonb_typeof("T"."tags") WHEN 'array' THEN "T"."tags" ELSE '[]' END) WHERE ("value" = $1))`,
					[]interface{}{"test"},
				},
			},
		},
		{
			[]exp.BooleanExpression{goqu.I("T.tags").Neq("test")},
			map[string]queryExpect{
				"sqlite3": {
					"SELECT * FROM `T` WHERE NOT EXISTS (SELECT * FROM json_each(CASE json_type(CASE json_valid(`T`.`tags`) WHEN true THEN `T`.`tags` ELSE '[]' END) WHEN 'array' THEN `T`.`tags` ELSE '[]' END) WHERE (`json_each`.`value` = ?))",
					[]interface{}{"test"},
				},
				"postgres": {
					`SELECT * FROM "T" WHERE NOT EXISTS (SELECT "value" FROM jsonb_array_elements_text(CASE jsonb_typeof("T"."tags") WHEN 'array' THEN "T"."tags" ELSE '[]' END) WHERE ("value" = $1))`,
					[]interface{}{"test"},
				},
			},
		},
		{
			[]exp.BooleanExpression{goqu.I("T.tags").Like("test")},
			map[string]queryExpect{
				"sqlite3": {
					"SELECT * FROM `T` WHERE EXISTS (SELECT * FROM json_each(CASE json_type(CASE json_valid(`T`.`tags`) WHEN true THEN `T`.`tags` ELSE '[]' END) WHEN 'array' THEN `T`.`tags` ELSE '[]' END) WHERE (`json_each`.`value` LIKE ?))",
					[]interface{}{"test"},
				},
				"postgres": {
					`SELECT * FROM "T" WHERE EXISTS (SELECT "value" FROM jsonb_array_elements_text(CASE jsonb_typeof("T"."tags") WHEN 'array' THEN "T"."tags" ELSE '[]' END) WHERE ("value" ILIKE $1))`,
					[]interface{}{"test"},
				},
			},
		},
		{
			[]exp.BooleanExpression{goqu.I("T.tags").NotLike("test")},
			map[string]queryExpect{
				"sqlite3": {
					"SELECT * FROM `T` WHERE NOT EXISTS (SELECT * FROM json_each(CASE json_type(CASE json_valid(`T`.`tags`) WHEN true THEN `T`.`tags` ELSE '[]' END) WHEN 'array' THEN `T`.`tags` ELSE '[]' END) WHERE (`json_each`.`value` LIKE ?))",
					[]interface{}{"test"},
				},
				"postgres": {
					`SELECT * FROM "T" WHERE NOT EXISTS (SELECT "value" FROM jsonb_array_elements_text(CASE jsonb_typeof("T"."tags") WHEN 'array' THEN "T"."tags" ELSE '[]' END) WHERE ("value" ILIKE $1))`,
					[]interface{}{"test"},
				},
			},
		},
		{
			[]exp.BooleanExpression{goqu.I("T.tags").Eq("test"), goqu.I("T.labels").Neq("test2")},
			map[string]queryExpect{
				"sqlite3": {
					"SELECT * FROM `T` WHERE (EXISTS (SELECT * FROM json_each(CASE json_type(CASE json_valid(`T`.`tags`) WHEN true THEN `T`.`tags` ELSE '[]' END) WHEN 'array' THEN `T`.`tags` ELSE '[]' END) WHERE (`json_each`.`value` = ?)) AND NOT EXISTS (SELECT * FROM json_each(CASE json_type(CASE json_valid(`T`.`labels`) WHEN true THEN `T`.`labels` ELSE '[]' END) WHEN 'array' THEN `T`.`labels` ELSE '[]' END) WHERE (`json_each`.`value` = ?)))",
					[]interface{}{"test", "test2"},
				},
				"postgres": {
					`SELECT * FROM "T" WHERE (EXISTS (SELECT "value" FROM jsonb_array_elements_text(CASE jsonb_typeof("T"."tags") WHEN 'array' THEN "T"."tags" ELSE '[]' END) WHERE ("value" = $1)) AND NOT EXISTS (SELECT "value" FROM jsonb_array_elements_text(CASE jsonb_typeof("T"."labels") WHEN 'array' THEN "T"."labels" ELSE '[]' END) WHERE ("value" = $2)))`,
					[]interface{}{"test", "test2"},
				},
			},
		},
	}

	for i, test := range tests {
		for _, dialect := range []string{"sqlite3", "postgres"} {
			if _, ok := test.expected[dialect]; !ok {
				continue
			}
			t.Run(fmt.Sprintf("%d-%s", i+1, dialect), func(t *testing.T) {
				ds := goqu.Dialect(dialect).Select().From("T")
				ds = filters.JSONListFilter(ds, test.expressions...)

				sql, args, err := ds.Prepared(true).ToSQL()
				require.NoError(t, err)

				require.Equal(t,
					test.expected[dialect].sql,
					sql,
				)
				require.Equal(t, test.expected[dialect].args, args)
			})
		}
	}
}

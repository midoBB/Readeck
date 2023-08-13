// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bookmarks

import (
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"

	"github.com/readeck/readeck/internal/searchstring"
)

var allowedSearchFields = map[string]bool{
	"author": true,
	"label":  true,
	"site":   true,
	"title":  true,
}

// searchString is a list of search terms
type searchString []searchstring.SearchTerm

// newSearchString parse an in and returns a new searchString.
func newSearchString(input string) (res searchString) {
	var err error
	res, err = searchstring.Parse(input)
	if err != nil {
		return
	}
	return
}

// addField adds untagged terms from the input to the searchString
// with a defined field name.
func (st *searchString) addField(name, input string) {
	var n searchString
	var err error
	n, err = searchstring.Parse(input)
	if err != nil {
		return
	}
	fields, _ := n.popField("")
	for _, x := range fields {
		x.Field = name
		*st = append(*st, x)
	}
}

// dedup returns a new searchString instance without duplicates
func (st searchString) dedup() (res searchString) {
	res = searchString{}
	seen := make(map[string]map[string]struct{})
	for _, t := range st {
		if _, ok := seen[t.Field][t.Value]; !ok {
			res = append(res, t)
			if _, ok := seen[t.Field]; !ok {
				seen[t.Field] = make(map[string]struct{})
			}
			seen[t.Field][t.Value] = struct{}{}
		}
	}

	return res
}

// popField removes the given field types and returns a new
// the removed fields and the new instance.
func (st searchString) popField(fieldName string) (fields searchString, newString searchString) {
	fields = searchString{}
	newString = searchString{}

	for _, t := range st {
		if t.Field == fieldName {
			fields = append(fields, t)
		} else {
			newString = append(newString, t)
		}
	}

	return
}

// fieldString returns a string for all the terms of a given field.
func (st searchString) fieldString(name string) string {
	res := []string{}
	for _, x := range st {
		if x.Field == name {
			res = append(res, x.Quoted())
		}
	}
	return strings.Join(res, " ")
}

// toSelectDataSet returns an augmented select dataset including the search query.
// Its implementation differs on database dialect.
func (st searchString) toSelectDataSet(ds *goqu.SelectDataset) *goqu.SelectDataset {
	if len(st) == 0 {
		return ds
	}

	switch ds.Dialect().Dialect() {
	case "postgres":
		return st.toPG(ds)
	case "sqlite3":
		return st.toSQLite(ds)
	}

	panic("dialect not implemented")
}

func (st searchString) toPG(ds *goqu.SelectDataset) *goqu.SelectDataset {
	where := goqu.And()
	order := []exp.OrderedExpression{}

	// In order to use the GIN indexes, we build a fairly big but very efficient query.
	// For general search, we add a group of OR clauses to the main clauses list.
	for _, x := range st {
		var fields = []string{"bs.title", "bs.description", "bs.text", "bs.site", "bs.author", "bs.label"}

		value := x.Value
		if x.Quotes {
			value = fmt.Sprintf(`"%s"`, value)
		}

		if x.Field != "" && allowedSearchFields[x.Field] {
			fields = []string{fmt.Sprintf("bs.%s", x.Field)}
		}

		w := goqu.Or()
		for _, f := range fields {
			w = w.Append(goqu.L(`? @@ websearch_to_tsquery('ts', ?)`, goqu.L(f), value))
			order = append(order, goqu.L(`ts_rank_cd(?, websearch_to_tsquery('ts', ?))`, goqu.L(f), value).Desc())
		}
		where = where.Append(w)
	}

	return ds.Prepared(false).Join(
		goqu.T("bookmark_search").As("bs"),
		goqu.On(goqu.Ex{"bs.bookmark_id": goqu.I("b.id")}),
	).
		Where(where).
		Order(order...)
}

func (st searchString) toSQLite(ds *goqu.SelectDataset) *goqu.SelectDataset {
	// This is a huge mess. We must pass the search query as a full literal,
	// otherwise it fails on many edge cases.
	// /!\ HERE ARE DRAGONS!
	// We must absolutely properly escape the search value to avoid injections.
	matchQ := []string{}
	rpl := strings.NewReplacer(`"`, `""`, `'`, `''`)

	for _, x := range st {
		q := fmt.Sprintf(`"%s"`, rpl.Replace(x.Value))

		if x.Field != "" && allowedSearchFields[x.Field] {
			q = fmt.Sprintf("%s:%s", x.Field, q)
		}

		matchQ = append(matchQ, q)
	}

	return ds.Join(
		goqu.T("bookmark_idx").As("bi"),
		goqu.On(goqu.Ex{"bi.rowid": goqu.I("b.id")}),
	).
		Where(goqu.L(`bookmark_idx match '?'`, goqu.L(strings.Join(matchQ, " ")))).
		Order(goqu.L("bm25(bookmark_idx, 12.0, 6.0, 5.0, 2.0, 4.0)").Asc())
}

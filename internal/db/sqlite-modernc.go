// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// To compile with CGO_ENABLED=0
//go:build !cgo

package db

import (
	"database/sql"
	"net/url"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3" // dialect
	"modernc.org/sqlite"                               // driver
)

func init() {
	drivers["sqlite3"] = &sqliteConnector{}

	sqlite.MustRegisterCollationUtf8("UNICODE", UnaccentCompare)
}

type sqliteConnector struct{}

func (c *sqliteConnector) Name() string {
	return "modernc/sqlite"
}

func (c *sqliteConnector) Dialect() string {
	return "sqlite3"
}

func (c *sqliteConnector) Open(dsn *url.URL) (*sql.DB, error) {
	uri := *dsn

	// Remove scheme
	uri.Scheme = ""

	// Set default options
	uri.RawQuery = "?_txlock=immediate"
	db, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		PRAGMA foreign_keys = 1;
		PRAGMA journal_mode = WAL;
		PRAGMA mmap_size = 30000000000;
		PRAGMA cache_size = 1000000000;
		PRAGMA secure_delete = 1;
		PRAGMA synchronous = NORMAL;
		PRAGMA busy_timeout = 5000;
		PRAGMA temp_store = memory;
	`)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(2)
	return db, nil
}

func (c *sqliteConnector) HasTable(name string) (bool, error) {
	ds := Q().Select(goqu.C("name")).
		From(goqu.T("sqlite_master")).
		Where(
			goqu.C("type").Eq("table"),
			goqu.C("name").Eq(name),
		)
	var res string

	if _, err := ds.ScanVal(&res); err != nil {
		return false, err
	}

	return res == name, nil
}

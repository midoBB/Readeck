// SPDX-FileCopyrightText: © 2020 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// To compile with CGO_ENABLED=1
//go:build cgo

package db

import (
	"database/sql"
	"net/url"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3" // dialect
	"github.com/mattn/go-sqlite3"                      // driver

	"codeberg.org/readeck/readeck/internal/db/exp"
)

func init() {
	drivers["sqlite3"] = &sqliteConnector{}

	sql.Register("sqlite3_extended", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			return conn.RegisterCollation("UNICODE", exp.UnaccentCompare)
		},
	})
}

type sqliteConnector struct{}

func (c *sqliteConnector) Name() string {
	return "mattn/go-sqlite3"
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
	db, err := sql.Open("sqlite3_extended", uri.String())
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

func (c *sqliteConnector) Version() string {
	var res string
	if _, err := Q().Select(goqu.Func("sqlite_version")).ScanVal(&res); err != nil {
		panic(err)
	}
	return res
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

// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3" // dialect
	"github.com/ncruces/go-sqlite3"
	sqliteDriver "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed" // wasm
	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/ncruces/go-sqlite3/vfs/memdb"

	"codeberg.org/readeck/readeck/internal/db/exp"
)

func init() {
	drivers["sqlite3"] = &sqliteConnector{}
}

func openSqlite(name string) (*sql.DB, error) {
	return sqliteDriver.Open(name, func(c *sqlite3.Conn) error {
		return c.CreateCollation("UNICODE", exp.UnicodeCollate.Compare)
	})
}

func getSqliteDsn(dsn *url.URL) (*url.URL, error) {
	if dsn.Opaque == ":memory:" {
		return &url.URL{
			Scheme: "memory",
			Opaque: "/memory.db",
		}, nil
	}

	var err error
	uri := &url.URL{Scheme: "file"}

	// Support initial dsn in several forms
	switch {
	case dsn.Opaque != "":
		// could be sqlite3:some/path
		uri.Path, err = filepath.Abs(dsn.Opaque)
	case dsn.Path != "":
		// or sqlite3:///some/path
		uri.Path, err = filepath.Abs(dsn.Path)
	default:
		err = fmt.Errorf("%s is not a valid database URI", dsn)
	}
	if err != nil {
		return nil, err
	}

	// Convert it to file:<path> (without // path prefix)
	uri.Opaque = uri.Path
	uri.Path = ""

	return uri, nil
}

type sqliteConnector struct{}

func (c *sqliteConnector) Name() string {
	return "ncruces/go-sqlite3"
}

func (c *sqliteConnector) Dialect() string {
	return "sqlite3"
}

func (c *sqliteConnector) Open(dsn *url.URL) (*sql.DB, error) {
	var err error

	// Prepare URI
	uri, err := getSqliteDsn(dsn)
	if err != nil {
		return nil, err
	}
	query := uri.Query()

	if uri.Scheme == "memory" {
		// In-memory database (for tests)
		var memoryDB []byte
		memdb.Create("memory.db", memoryDB)
		uri.Scheme = "file"
		query.Set("vfs", "memdb")
	}

	query.Set("_txtlock", "immediate")
	query.Set("_timefmt", "auto")
	uri.RawQuery = query.Encode()

	slog.Debug("connect to database", slog.Any("dsn", uri.String()))
	db, err := openSqlite(uri.String())
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		PRAGMA busy_timeout = 5000;
		PRAGMA foreign_keys = 1;
		PRAGMA journal_mode = WAL;
		PRAGMA mmap_size = 30000000000;
		PRAGMA cache_size = 1000000000;
		PRAGMA secure_delete = 1;
		PRAGMA synchronous = NORMAL;
		PRAGMA temp_store = memory;
	`)
	if err != nil {
		return nil, err
	}

	if vfs.SupportsSharedMemory {
		db.SetMaxOpenConns(2)
	} else {
		db.SetMaxOpenConns(1)
	}
	return db, nil
}

func (c *sqliteConnector) Version() string {
	var res string
	if _, err := Q().Select(goqu.Func("sqlite_version")).ScanVal(&res); err != nil {
		panic(err)
	}
	return res
}

func (c *sqliteConnector) DiskUsage() (int64, error) {
	var sizeBytes int64
	if _, err := Q().Select(
		goqu.L("page_count * page_size")).
		From(goqu.L("pragma_page_count()")).
		CrossJoin(goqu.L("pragma_page_size()")).
		ScanVal(&sizeBytes); err != nil {
		panic(err)
	}
	return sizeBytes, nil
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

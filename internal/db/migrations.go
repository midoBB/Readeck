package db

import (
	"io/fs"
	"io/ioutil"
	"math/rand"
	"path"

	"github.com/doug-martin/goqu/v9"

	"github.com/readeck/readeck/internal/db/migrations"
)

type migrationFunc func(*goqu.TxDatabase, fs.FS) error

type migrationEntry struct {
	id       int
	name     string
	funcList []migrationFunc
}

// newMigrationEntry creates a new migration which contains an id, a name and a list
// of functions performing the migration.
func newMigrationEntry(id int, name string, funcList ...migrationFunc) migrationEntry {
	res := migrationEntry{
		id:       id,
		name:     name,
		funcList: []migrationFunc{},
	}
	res.funcList = funcList
	return res
}

func applyMigrationFile(name string) func(td *goqu.TxDatabase, _ fs.FS) (err error) {
	return func(td *goqu.TxDatabase, _ fs.FS) (err error) {
		var fd fs.File
		if fd, err = migrations.Files.Open(path.Join(td.Dialect(), name)); err != nil {
			return
		}

		var sql []byte
		if sql, err = ioutil.ReadAll(fd); err != nil {
			return
		}

		_, err = td.Exec(string(sql))
		return
	}
}

// migrationList is our full migration list
var migrationList = []migrationEntry{
	newMigrationEntry(1, "user_seed", func(td *goqu.TxDatabase, _ fs.FS) (err error) {
		// Add a seed column to the user table
		sql := `ALTER TABLE "user" ADD COLUMN seed INTEGER NOT NULL DEFAULT 0;`

		if _, err = td.Exec(sql); err != nil {
			return
		}

		// Set a new seed on every user
		var ids []int64
		if err = td.From("user").Select("id").ScanVals(&ids); err != nil {
			return
		}
		for _, id := range ids {
			seed := rand.Intn(32767)
			_, err = td.Update("user").
				Set(goqu.Record{"seed": seed}).
				Where(goqu.C("id").Eq(id)).
				Executor().Exec()
			if err != nil {
				return
			}
		}

		return
	}),

	newMigrationEntry(2, "bookmark_collection", applyMigrationFile("02_bookmark_collection.sql")),
}

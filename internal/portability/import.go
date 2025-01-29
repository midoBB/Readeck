// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package portability

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/lithammer/shortuuid/v4"

	"codeberg.org/readeck/readeck/internal/auth/credentials"
	"codeberg.org/readeck/readeck/internal/auth/tokens"
	"codeberg.org/readeck/readeck/internal/auth/users"
	"codeberg.org/readeck/readeck/internal/bookmarks"
	"codeberg.org/readeck/readeck/internal/db"
	"codeberg.org/readeck/readeck/pkg/zipfs"
)

// Importer is a content importer.
type Importer struct {
	usernames []string
	users     map[int]int
	zr        *zip.Reader
	output    io.Writer
}

// NewImporter creates a new [Importer].
func NewImporter(zr *zip.Reader, usernames []string) (*Importer, error) {
	return &Importer{
		zr:        zr,
		usernames: usernames,
		users:     map[int]int{},
		output:    io.Discard,
	}, nil
}

// Output returns the message output writer.
func (imp *Importer) Output() io.Writer {
	return imp.output
}

// SetOutput sets the message output writer.
func (imp *Importer) SetOutput(w io.Writer) {
	imp.output = w
}

// Load loads all the data into Readeck's database and content folder.
func (imp *Importer) Load() error {
	fd, err := imp.zr.Open("data.json")
	if err != nil {
		return err
	}

	fnList := []func(*goqu.TxDatabase, *portableData) error{
		imp.loadUsers,
		imp.loadTokens,
		imp.loadCredentials,
		imp.loadCollections,
		imp.loadBookmarks,
	}

	var data portableData
	dec := json.NewDecoder(fd)
	if err = dec.Decode(&data); err != nil {
		return err
	}
	tx, err := db.Q().Begin()
	if err != nil {
		return err
	}
	return tx.Wrap(func() error {
		for _, fn := range fnList {
			if err = fn(tx, &data); err != nil {
				return err
			}
		}
		return nil
	})
}

func (imp *Importer) loadUsers(tx *goqu.TxDatabase, data *portableData) (err error) {
	allUsers := len(imp.usernames) == 0

	for _, item := range data.Users {
		if allUsers {
			imp.usernames = append(imp.usernames, item.Username)
		}

		if !slices.Contains(imp.usernames, item.Username) {
			continue
		}

		var user *users.User
		user, err = users.Users.GetOne(
			goqu.Or(
				goqu.C("username").Eq(item.Username),
				goqu.C("email").Eq(item.Email),
			),
		)
		if err != nil && !errors.Is(err, users.ErrNotFound) {
			return err
		}
		if user != nil {
			fmt.Fprintf( // nolint:errcheck
				imp.output,
				"\tERR: user \"%s\" or \"%s\" already exists\n", item.Username, item.Email,
			)
			continue
		}

		originalID := item.ID
		if item.ID, err = insertInto(tx, users.TableName, item, func(x *users.User) {
			x.ID = 0
		}); err != nil {
			return
		}
		imp.users[originalID] = item.ID
	}

	fmt.Fprintf(imp.output, "\t- %d user(s) imported\n", len(imp.users)) // nolint:errcheck
	return
}

func (imp *Importer) loadTokens(tx *goqu.TxDatabase, data *portableData) (err error) {
	ids := slices.Collect(maps.Keys(imp.users))

	i := 0
	for _, item := range data.Tokens {
		if item.UserID == nil {
			continue
		}
		if !slices.Contains(ids, *item.UserID) {
			continue
		}

		if item.ID, err = insertInto(tx, tokens.TableName, item, func(x *tokens.Token) {
			x.ID = 0
			x.UserID = ptrTo(imp.users[*x.UserID])
		}); err != nil {
			return
		}
		i++
	}

	fmt.Fprintf(imp.output, "\t- %d token(s) imported\n", i) // nolint:errcheck
	return
}

func (imp *Importer) loadCredentials(tx *goqu.TxDatabase, data *portableData) (err error) {
	ids := slices.Collect(maps.Keys(imp.users))

	i := 0
	for _, item := range data.Credentials {
		if item.UserID == nil {
			continue
		}
		if !slices.Contains(ids, *item.UserID) {
			continue
		}

		if item.ID, err = insertInto(tx, credentials.TableName, item, func(x *credentials.Credential) {
			x.ID = 0
			x.UserID = ptrTo(imp.users[*x.UserID])
		}); err != nil {
			return
		}
		i++
	}

	fmt.Fprintf(imp.output, "\t- %d credential(s) imported\n", i) // nolint:errcheck
	return
}

func (imp *Importer) loadCollections(tx *goqu.TxDatabase, data *portableData) (err error) {
	ids := slices.Collect(maps.Keys(imp.users))

	i := 0
	for _, item := range data.BookmarkCollections {
		if item.UserID == nil {
			continue
		}
		if !slices.Contains(ids, *item.UserID) {
			continue
		}

		if item.ID, err = insertInto(tx, bookmarks.CollectionTable, item, func(x *bookmarks.Collection) {
			x.ID = 0
			x.UID = shortuuid.New()
			x.UserID = ptrTo(imp.users[*x.UserID])
		}); err != nil {
			return
		}
		i++
	}

	fmt.Fprintf(imp.output, "\t- %d collection(s) imported\n", i) // nolint:errcheck
	return
}

func (imp *Importer) loadBookmarks(tx *goqu.TxDatabase, data *portableData) (err error) {
	ids := slices.Collect(maps.Keys(imp.users))

	i := 0
	for _, item := range data.Bookmarks {
		if !slices.Contains(ids, item.UserID) {
			continue
		}

		if err = imp.loadBookmark(tx, &item); err != nil {
			return
		}
		i++
	}

	fmt.Fprintf(imp.output, "\t- %d bookmark(s) imported\n", i) // nolint:errcheck
	return
}

func (imp *Importer) loadBookmark(tx *goqu.TxDatabase, item *bookmarkItem) (err error) {
	p := path.Join("bookmarks", item.UID, "info.json")
	fd, err := imp.zr.Open(p)
	if err != nil {
		return err
	}

	var b bookmarks.Bookmark
	dec := json.NewDecoder(fd)
	if err = dec.Decode(&b); err != nil {
		return
	}

	if b.ID, err = insertInto(tx, bookmarks.TableName, &b, func(x *bookmarks.Bookmark) {
		x.ID = 0
		x.UID = shortuuid.New()
		x.FilePath, _ = x.GetBaseFileURL()
		x.UserID = ptrTo(imp.users[*x.UserID])
	}); err != nil {
		return
	}

	// Copy files to zipfile
	dest := filepath.Join(bookmarks.StoragePath(), b.FilePath+".zip")
	if err = os.MkdirAll(path.Dir(dest), 0o750); err != nil {
		return err
	}
	w, err := os.Create(dest)
	if err != nil {
		return err
	}

	zw := zipfs.NewZipRW(w, nil, 0)
	defer zw.Close() // nolint:errcheck

	prefix := "bookmarks/" + item.UID + "/container/"
	for _, f := range imp.zr.File {
		if f.FileInfo().IsDir() {
			continue
		}

		if !strings.HasPrefix(f.Name, prefix) {
			continue
		}

		r, err := f.Open()
		if err != nil {
			return err
		}
		f.Name = strings.TrimPrefix(f.Name, prefix)
		if err = zw.Add(&f.FileHeader, r); err != nil {
			r.Close() // nolint:errcheck
			return err
		}
		if err = r.Close(); err != nil {
			return err
		}
	}

	return
}

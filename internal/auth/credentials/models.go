// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// Package credentials contains the models and functions to manage
// user credentials.
package credentials

import (
	"errors"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	argon2 "github.com/hlandau/passlib/hash/argon2/raw"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"codeberg.org/readeck/readeck/configs"
	"codeberg.org/readeck/readeck/internal/auth/users"
	"codeberg.org/readeck/readeck/internal/db"
	"codeberg.org/readeck/readeck/internal/db/types"
	"codeberg.org/readeck/readeck/pkg/base58"
)

const (
	// TableName is the credential table name in database.
	TableName = "credential"
)

var (
	// Credentials is the app password manager.
	Credentials = Manager{}

	// ErrNotFound is returned when a credential record was not found.
	ErrNotFound = errors.New("not found")

	// Fixed values for password hashing.
	argonTime    = uint32(4)
	argonMemory  = uint32(32 * 1024)
	argonThreads = uint8(4)
)

// Credential is an credential record.
type Credential struct {
	ID        int           `db:"id" goqu:"skipinsert,skipupdate"`
	UID       string        `db:"uid"`
	UserID    *int          `db:"user_id"`
	Created   time.Time     `db:"created" goqu:"skipupdate"`
	LastUsed  *time.Time    `db:"last_used"`
	IsEnabled bool          `db:"is_enabled"`
	Name      string        `db:"name"`
	Password  string        `db:"password"`
	Roles     types.Strings `db:"roles"`
}

// UserCredential is the combination of an credential and its user.
type UserCredential struct {
	Credential *Credential `db:"c"`
	User       *users.User `db:"u"`
}

// Manager is a query helper for credential entries.
type Manager struct{}

// Query returns a prepared goqu SelectDataset that can be extended later.
func (m *Manager) Query() *goqu.SelectDataset {
	return db.Q().From(goqu.T(TableName).As("c")).Prepared(true)
}

// GetOne executes the a select query and returns the first result or an error
// when there's no result.
func (m *Manager) GetOne(expressions ...goqu.Expression) (*Credential, error) {
	var c Credential
	found, err := m.Query().Where(expressions...).ScanStruct(&c)

	switch {
	case err != nil:
		return nil, err
	case !found:
		return nil, ErrNotFound
	}

	return &c, nil
}

// GetUser attempts to find a user with a matching credential.
// It returns nil with ErrNotFound if no user and/or password match the query.
func (m *Manager) GetUser(username, password string) (*UserCredential, error) {
	// First get the user from its username
	u, err := users.Users.GetOne(goqu.C("username").Eq(username))
	if err != nil {
		if errors.Is(err, users.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Prepare the credential and hash the given password for
	// the query against the credential table
	c := &Credential{UserID: &u.ID}
	hash, err := HashPassword(u, password)
	if err != nil {
		return nil, err
	}

	// Look for the credential with the given hash
	q := m.Query().
		Select().
		Where(goqu.I("c.is_enabled").Eq(true)).
		Where(goqu.I("c.user_id").Eq(u.ID)).
		Where(goqu.I("c.password").Eq(hash))

	if found, err := q.ScanStruct(c); err != nil {
		return nil, err
	} else if found {
		return &UserCredential{Credential: c, User: u}, nil
	}

	return nil, ErrNotFound
}

// Create insert a new credential in the database.
func (m *Manager) Create(c *Credential) error {
	if c.UserID == nil {
		return errors.New("no token user")
	}
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("no application")
	}

	c.Created = time.Now()
	c.UID = base58.NewUUID()

	ds := db.Q().Insert(TableName).
		Rows(c).
		Prepared(true)

	id, err := db.InsertWithID(ds, "id")
	if err != nil {
		return err
	}

	c.ID = id
	return nil
}

// GenerateCredential creates a new credential with a random name and passphrase.
// It returns the [Credential] instance, the unencrypted passphrase and an error if any.
func (m *Manager) GenerateCredential(user *users.User) (c *Credential, cleartext string, err error) {
	var name string
	if name, err = MakePassphrase(2); err != nil {
		return
	}
	c = &Credential{
		UserID:    &user.ID,
		IsEnabled: true,
		Name:      cases.Title(language.AmericanEnglish).String(name),
	}
	if cleartext, err = c.NewPassphrase(user); err != nil {
		c = nil
		return
	}

	err = m.Create(c)
	return
}

// Update updates some user values.
func (c *Credential) Update(v any) error {
	if c.ID == 0 {
		return errors.New("no ID")
	}

	_, err := db.Q().Update(TableName).Prepared(true).
		Set(v).
		Where(goqu.C("id").Eq(c.ID)).
		Executor().Exec()

	return err
}

// Save updates all the password values.
func (c *Credential) Save() error {
	return c.Update(c)
}

// Delete removes a token from the database.
func (c *Credential) Delete() error {
	_, err := db.Q().Delete(TableName).Prepared(true).
		Where(goqu.C("id").Eq(c.ID)).
		Executor().Exec()

	return err
}

// NewPassphrase creates a new passphrase and adds its hash to the
// [Credential.Password] field.
// It returns the cleartext passphrase.
func (c *Credential) NewPassphrase(user *users.User) (cleartext string, err error) {
	if cleartext, err = MakePassphrase(6); err != nil {
		return "", err
	}
	if c.Password, err = HashPassword(user, cleartext); err != nil {
		return "", err
	}
	return cleartext, nil
}

// HashPassword returns a new hashed password.
func HashPassword(user *users.User, password string) (string, error) {
	// We create a salt based on the user UID, expanding the main PRK
	// so we have a per user salt.
	salt, err := configs.Keys.Expand("credential_"+user.UID, 16)
	if err != nil {
		return "", err
	}

	// Direct call to argon2 hashing, using the salt and strong defaults
	return argon2.Argon2(password, salt, argonTime, argonMemory, argonThreads), nil
}

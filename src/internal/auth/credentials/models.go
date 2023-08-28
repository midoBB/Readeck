// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package credentials

import (
	"errors"
	"strings"
	"time"

	"github.com/hlandau/passlib"
	"github.com/hlandau/passlib/hash/argon2"
	"gopkg.in/hlandau/passlib.v1/abstract"

	"github.com/doug-martin/goqu/v9"
	"github.com/lithammer/shortuuid/v4"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"codeberg.org/readeck/readeck/internal/auth/users"
	"codeberg.org/readeck/readeck/internal/db"
)

const (
	// TableName is the credential table name in database.
	TableName = "credential"
)

var (
	// Credentials is the app password manager
	Credentials = Manager{}

	// ErrNotFound is returned when a user record was not found.
	ErrNotFound = errors.New("not found")

	// We insert here a less safe password hashing scheme because
	// the verification happens on all the enabled user passwords.
	// It's a balance between security and the API response time.
	passlibContext = passlib.Context{
		Schemes: append([]abstract.Scheme{
			argon2.New(4, 1024, 4),
		}, passlib.DefaultSchemes...),
	}
)

// Credential is an credential record
type Credential struct {
	ID        int        `db:"id" goqu:"skipinsert,skipupdate"`
	UID       string     `db:"uid"`
	UserID    *int       `db:"user_id"`
	Created   time.Time  `db:"created" goqu:"skipupdate"`
	IsEnabled bool       `db:"is_enabled"`
	Name      string     `db:"name"`
	Password  string     `db:"password"`
	Roles     db.Strings `db:"roles"`
}

// UserCredential is the combination of an credential and its user
type UserCredential struct {
	Credential *Credential `db:"c"`
	User       *users.User `db:"u"`
}

// Manager is a query helper for credential entries
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
	q := m.Query().
		Select().
		Join(
			goqu.T(users.TableName).As("u"),
			goqu.On(goqu.I("u.id").Eq(goqu.I("c.user_id"))),
		).
		Where(goqu.I("c.is_enabled").Eq(true)).
		Where(goqu.I("u.username").Eq(username)).
		Order(goqu.I("c.created").Desc())

	items := []*UserCredential{}

	if err := q.ScanStructs(&items); err != nil {
		return nil, err
	}

	for _, item := range items {
		if item.Credential.CheckPassword(password) {
			return item, nil
		}
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
	c.UID = shortuuid.New()

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
// It returns the Credential instance, the unencrypted passphrase and an error if any.
func (m *Manager) GenerateCredential(userID int) (c *Credential, passphrase string, err error) {
	var name string
	if passphrase, err = MakePassphrase(6); err != nil {
		return
	}
	if name, err = MakePassphrase(2); err != nil {
		return
	}
	c = &Credential{
		UserID:    &userID,
		IsEnabled: true,
		Name:      cases.Title(language.AmericanEnglish).String(name),
	}
	if c.Password, err = c.HashPassword(passphrase); err != nil {
		c = nil
		return
	}

	err = m.Create(c)
	return
}

// Update updates some user values.
func (c *Credential) Update(v interface{}) error {
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

// Delete removes a token from the database
func (c *Credential) Delete() error {
	_, err := db.Q().Delete(TableName).Prepared(true).
		Where(goqu.C("id").Eq(c.ID)).
		Executor().Exec()

	return err
}

// CheckPassword checks if the given password matches the
// current app password.
func (c *Credential) CheckPassword(password string) bool {
	newhash, err := passlibContext.Verify(password, c.Password)
	if err != nil {
		return false
	}

	if newhash != "" {
		_ = c.Update(goqu.Record{"password": newhash})
	}

	return true
}

// HashPassword returns a new hashed password
func (c *Credential) HashPassword(password string) (string, error) {
	return passlibContext.Hash(password)
}

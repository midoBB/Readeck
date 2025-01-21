// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bookmarks

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/lithammer/shortuuid/v4"

	"codeberg.org/readeck/readeck/internal/db"
	"codeberg.org/readeck/readeck/internal/db/types"
	"codeberg.org/readeck/readeck/pkg/forms"
)

const (
	// CollectionTable is the collection table name in database.
	CollectionTable = "bookmark_collection"
)

var (
	// Collections is the collection query manager.
	Collections = CollectionManager{}

	// ErrCollectionNotFound is returned when a collection record was not found.
	ErrCollectionNotFound = errors.New("not found")
)

// Collection is a collection record in the database.
type Collection struct {
	ID       int               `db:"id" goqu:"skipinsert,skipupdate"`
	UID      string            `db:"uid"`
	UserID   *int              `db:"user_id"`
	Created  time.Time         `db:"created" goqu:"skipupdate"`
	Updated  time.Time         `db:"updated"`
	Name     string            `db:"name"`
	IsPinned bool              `db:"is_pinned"`
	Filters  CollectionFilters `db:"filters"`
}

// CollectionManager is a query helper for bookmark entries.
type CollectionManager struct{}

// Query returns a prepared goqu SelectDataset that can be extended later.
func (m *CollectionManager) Query() *goqu.SelectDataset {
	return db.Q().From(goqu.T(CollectionTable).As("c")).Prepared(true)
}

// GetOne executes the a select query and returns the first result or an error
// when there's no result.
func (m *CollectionManager) GetOne(expressions ...goqu.Expression) (*Collection, error) {
	var c Collection
	found, err := m.Query().Where(expressions...).ScanStruct(&c)

	switch {
	case err != nil:
		return nil, err
	case !found:
		return nil, ErrBookmarkNotFound
	}

	return &c, nil
}

// Create inserts a new collection in the database.
func (m *CollectionManager) Create(collection *Collection) error {
	if collection.UserID == nil {
		return errors.New("no collection user")
	}

	collection.Created = time.Now()
	collection.Updated = collection.Created
	collection.UID = shortuuid.New()

	ds := db.Q().Insert(CollectionTable).
		Rows(collection).
		Prepared(true)

	id, err := db.InsertWithID(ds, "id")
	if err != nil {
		return err
	}

	collection.ID = id

	return nil
}

// Update updates some collection values.
func (c *Collection) Update(v interface{}) error {
	if c.ID == 0 {
		return errors.New("no ID")
	}

	switch v := v.(type) {
	case map[string]interface{}:
		v["updated"] = time.Now()
	default:
		//
	}

	_, err := db.Q().Update(CollectionTable).Prepared(true).
		Set(v).
		Where(goqu.C("id").Eq(c.ID)).
		Executor().Exec()

	return err
}

// Save updates all the collection values.
func (c *Collection) Save() error {
	c.Updated = time.Now()
	return c.Update(c)
}

// Delete removes a collection from the database.
func (c *Collection) Delete() error {
	_, err := db.Q().Delete(CollectionTable).Prepared(true).
		Where(goqu.C("id").Eq(c.ID)).
		Executor().Exec()

	return err
}

// GetSumStrings returns the string used to generate the etag
// of the collection(s).
func (c *Collection) GetSumStrings() []string {
	return []string{c.UID, c.Updated.String()}
}

// CollectionFilters contains the filters applied by a collection.
type CollectionFilters struct {
	Search     string        `json:"search"`
	Title      string        `json:"title"`
	Author     string        `json:"author"`
	Site       string        `json:"site"`
	Type       types.Strings `json:"type"`
	Labels     string        `json:"labels"`
	ReadStatus types.Strings `json:"read_status"`
	IsMarked   *bool         `json:"is_marked"`
	IsArchived *bool         `json:"is_archived"`
	IsLoaded   *bool         `json:"is_loaded"`
	HasErrors  *bool         `json:"has_errors"`
	HasLabels  *bool         `json:"has_labels"`
	RangeStart string        `json:"range_start"`
	RangeEnd   string        `json:"range_end"`
}

// Scan loads a CollectionFilters instance from a column.
func (s *CollectionFilters) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	v, err := types.JSONBytes(value)
	if err != nil {
		return err
	}
	json.Unmarshal(v, s) //nolint:errcheck
	return nil
}

// Value encodes a CollectionFilters value for storage.
func (s CollectionFilters) Value() (driver.Value, error) {
	v, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(v), nil
}

// LoadForm loads the form values as properties.
func (s *CollectionFilters) LoadForm(f forms.Binder) error {
	values := map[string]any{}

	for name, field := range f.Fields() {
		values[name] = field.Value()
	}

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(values)
	if err != nil {
		return err
	}

	return json.Unmarshal(buf.Bytes(), s)
}

// ToValues returns the result of a JSON encoding to a map.
func (s *CollectionFilters) ToValues() (map[string]any, error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(s); err != nil {
		return nil, err
	}
	res := map[string]any{}
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		return nil, err
	}
	return res, nil
}

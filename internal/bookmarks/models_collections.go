package bookmarks

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/lithammer/shortuuid"

	"github.com/readeck/readeck/internal/db"
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

func (c *Collection) get(name string) interface{} {
	switch name {
	case "name":
		return c.Name
	case "is_pinned":
		return c.IsPinned
	case "search":
		return c.Filters.Search
	case "title":
		return c.Filters.Title
	case "author":
		return c.Filters.Author
	case "site":
		return c.Filters.Site
	case "type":
		return c.Filters.Type
	case "labels":
		return c.Filters.Labels
	case "is_marked":
		if c.Filters.IsMarked == nil {
			return nil
		}
		return *c.Filters.IsMarked
	case "is_archived":
		if c.Filters.IsArchived == nil {
			return nil
		}
		return *c.Filters.IsArchived
	}

	panic(fmt.Errorf(`unknown field "%s"`, name))
}

func (c *Collection) set(name string, value interface{}) {
	switch name {
	case "name":
		c.Name = value.(string)
		return
	case "is_pinned":
		c.IsPinned = value.(bool)
		return
	case "search":
		c.Filters.Search = value.(string)
		return
	case "title":
		c.Filters.Title = value.(string)
		return
	case "author":
		c.Filters.Author = value.(string)
		return
	case "site":
		c.Filters.Site = value.(string)
		return
	case "type":
		c.Filters.Type = value.(string)
		return
	case "labels":
		c.Filters.Labels = value.(string)
		return
	case "is_marked":
		c.Filters.IsMarked = value.(*bool)
		return
	case "is_archived":
		c.Filters.IsArchived = value.(*bool)
		return
	}

	panic(fmt.Errorf(`unknown field "%s"`, name))
}

func (c *Collection) flatten() map[string]interface{} {
	res := map[string]interface{}{
		"name":        c.Name,
		"is_pinned":   c.IsPinned,
		"search":      c.Filters.Search,
		"title":       c.Filters.Title,
		"author":      c.Filters.Author,
		"site":        c.Filters.Site,
		"type":        c.Filters.Type,
		"labels":      c.Filters.Labels,
		"is_archived": nil,
		"is_marked":   nil,
	}
	if c.Filters.IsArchived != nil {
		res["is_archived"] = *c.Filters.IsArchived
	}
	if c.Filters.IsMarked != nil {
		res["is_marked"] = *c.Filters.IsMarked
	}

	return res
}

// Save updates all the collection values.
func (c *Collection) Save() error {
	c.Updated = time.Now()
	return c.Update(c)
}

// Delete removes a collection from the database
func (c *Collection) Delete() error {
	_, err := db.Q().Delete(CollectionTable).Prepared(true).
		Where(goqu.C("id").Eq(c.ID)).
		Executor().Exec()

	return err
}

// GetSumStrings returns the string used to generate the etag
// of the collection(s)
func (c *Collection) GetSumStrings() []string {
	return []string{c.UID, c.Updated.String()}
}

// CollectionFilters contains the filters applied by a collection.
type CollectionFilters struct {
	Search     string `json:"search"`
	Title      string `json:"title"`
	Author     string `json:"author"`
	Site       string `json:"site"`
	Type       string `json:"type"`
	Labels     string `json:"labels"`
	IsMarked   *bool  `json:"is_marked"`
	IsArchived *bool  `json:"is_archived"`
}

// Scan loads a CollectionFilters instance from a column.
func (s *CollectionFilters) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	v, err := db.JSONBytes(value)
	if err != nil {
		return err
	}
	json.Unmarshal(v, s)
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

// type collectionFilterMap map[string]interface{}

type filterMap map[string]interface{}

func (m filterMap) Value() (driver.Value, error) {
	v, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(v), nil
}

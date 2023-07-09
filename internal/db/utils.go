package db

import (
	sql_driver "database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/doug-martin/goqu/v9"
)

// InsertWithID executes an insert statement and returns the value
// of the field given by "r".
// Depending on the database, it uses different ways to do just that.
func InsertWithID(stmt *goqu.InsertDataset, r string) (id int, err error) {
	if Driver().Dialect() == "postgres" {
		_, err = stmt.Returning(goqu.C(r)).Executor().ScanVal(&id)
		return
	}
	res, err := stmt.Executor().Exec()
	if err != nil {
		return id, err
	}

	i, _ := res.LastInsertId()
	id = int(i)

	return
}

// JSONBytes converts a string or a []uint8 to a []byte value.
// We need this with sqlite and postgresql not returning the same
// data type for their json fields.
func JSONBytes(value interface{}) ([]byte, error) {
	switch x := value.(type) {
	case string:
		return []byte(x), nil
	case []uint8:
		return x, nil
	}

	return []byte{}, fmt.Errorf("unknown data type for %+v", value)
}

// Strings is a list of strings stored in a column.
type Strings []string

// Scan loads a Strings instance from a column.
func (s *Strings) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	v, err := JSONBytes(value)
	if err != nil {
		return err
	}
	json.Unmarshal(v, s)
	return nil
}

// Value encodes a Strings instance for storage.
func (s Strings) Value() (sql_driver.Value, error) {
	v, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(v), nil
}

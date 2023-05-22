package bookmarks

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"golang.org/x/net/html"

	"github.com/readeck/readeck/internal/db"
	"github.com/readeck/readeck/pkg/annotate"
)

// BookmarkAnnotations is a mapping of annotations.
type BookmarkAnnotations map[string]*BookmarkAnnotation

// BookmarkAnnotation is an annotation that can be serialized in a database JSON column.
type BookmarkAnnotation struct {
	StartSelector string    `json:"start_selector"`
	StartOffset   int       `json:"start_offset"`
	EndSelector   string    `json:"end_selector"`
	EndOffset     int       `json:"end_offset"`
	Created       time.Time `json:"created"`
	Text          string    `json:"text"`
}

// Scan loads a BookmarkAnnotations instance from a column.
func (a *BookmarkAnnotations) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	v, err := db.JSONBytes(value)
	if err != nil {
		return err
	}
	json.Unmarshal(v, a)
	return nil
}

// Value encodes a BookmarkAnnotations instance for storage.
func (a BookmarkAnnotations) Value() (driver.Value, error) {
	v, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	return string(v), nil
}

// addToNode adds one annotation to a DOM node (the designated root)
func (a *BookmarkAnnotation) addToNode(root *html.Node, tagName string, options ...annotate.WrapCallback) error {
	return annotate.AddAnnotation(
		root, tagName,
		a.StartSelector, a.StartOffset,
		a.EndSelector, a.EndOffset,
		options...,
	)
}

// addToNode adds all annotations to a DOM node (the designated root)
func (a BookmarkAnnotations) addToNode(root *html.Node, tagName string, options ...func(string, *html.Node, int)) error {
	for id, annotation := range a {
		err := annotation.addToNode(root, tagName, func(n *html.Node, index int) {
			for _, f := range options {
				f(id, n, index)
			}
		})
		if err != nil {
			return err
		}
	}
	return nil
}

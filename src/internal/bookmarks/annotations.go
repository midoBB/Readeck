// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bookmarks

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"

	"github.com/readeck/readeck/internal/db"
	"github.com/readeck/readeck/pkg/annotate"
)

// BookmarkAnnotations is a mapping of annotations.
type BookmarkAnnotations []*BookmarkAnnotation

// BookmarkAnnotation is an annotation that can be serialized in a database JSON column.
type BookmarkAnnotation struct {
	ID            string    `json:"id"`
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
	for _, annotation := range a {
		err := annotation.addToNode(root, tagName, func(n *html.Node, index int) {
			for _, f := range options {
				f(annotation.ID, n, index)
			}
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// get retrieves an annotation or returns nil if it does not exist
func (a *BookmarkAnnotations) get(id string) *BookmarkAnnotation {
	for _, x := range *a {
		if x.ID == id {
			return x
		}
	}
	return nil
}

// add adds a new annotation to the list
func (a *BookmarkAnnotations) add(item *BookmarkAnnotation) {
	set := *a
	set = append(set, item)
	*a = set
}

// sort sorts the annotations based on their position in the root document
func (a *BookmarkAnnotations) sort(root *html.Node, tagName string) {
	set := BookmarkAnnotations{}
	for _, n := range dom.QuerySelectorAll(root, tagName) {
		if dom.GetAttribute(n, "id") == "" {
			continue
		}
		id := dom.GetAttribute(n, "data-annotation-id-value")
		if item := a.get(id); item != nil {
			set = append(set, item)
		}
	}
	*a = set
}

// delete removes an annotation from the list
func (a *BookmarkAnnotations) delete(id string) {
	item := a.get(id)
	if item == nil {
		return
	}
	set := BookmarkAnnotations{}
	for _, x := range *a {
		if x.ID == id {
			continue
		}
		set = append(set, x)
	}
	*a = set
}

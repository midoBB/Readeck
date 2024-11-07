// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package routes

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/url"
	"time"

	"codeberg.org/readeck/readeck/internal/bookmarks"
	"codeberg.org/readeck/readeck/internal/bookmarks/tasks"
	"codeberg.org/readeck/readeck/pkg/forms"
)

type (
	ctxCollectionFormKey struct{}
)

type collectionDeleteForm struct {
	*forms.Form
}

type filterMap map[string]interface{}

func (m filterMap) Value() (driver.Value, error) {
	v, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(v), nil
}

func newCollectionDeleteForm(tr forms.Translator) (f *collectionDeleteForm) {
	f = &collectionDeleteForm{forms.Must(
		forms.NewBooleanField("cancel"),
		forms.NewTextField("_to"),
	)}
	f.SetLocale(tr)
	return
}

func (f *collectionDeleteForm) trigger(c *bookmarks.Collection) error {
	if !f.Get("cancel").IsNil() && f.Get("cancel").Value().(bool) {
		return tasks.DeleteCollectionTask.Cancel(c.ID)
	}

	return tasks.DeleteCollectionTask.Run(c.ID, c.ID)
}

type collectionForm struct {
	*forms.Form
	Filters *filterForm

	filterFields map[string]struct{}
}

func newCollectionForm(tr forms.Translator) (f *collectionForm) {
	f = &collectionForm{Form: forms.Must(
		forms.NewTextField("name", forms.Trim, forms.Required),
		forms.NewBooleanField("is_pinned"),
	)}
	f.SetLocale(tr)

	f.Filters = newFilterForm(tr)
	f.filterFields = map[string]struct{}{
		"search":      {},
		"title":       {},
		"author":      {},
		"site":        {},
		"type":        {},
		"labels":      {},
		"is_marked":   {},
		"is_archived": {},
		"is_loaded":   {},
		"has_errors":  {},
		"has_labels":  {},
		"range_start": {},
		"range_end":   {},
	}

	return
}

func (f *collectionForm) Fields() []*forms.FormField {
	fields := f.Form.Fields()
	res := make([]*forms.FormField, len(fields)+len(f.filterFields))
	i := 0
	for _, field := range f.Form.Fields() {
		res[i] = field
		i++
	}
	for _, field := range f.Filters.Fields() {
		if _, ok := f.filterFields[field.Name()]; ok {
			res[i] = field
			i++
		}
	}

	return res
}

func (f *collectionForm) FieldMap() map[string]*forms.FormField {
	res := f.Form.FieldMap()
	for k, v := range f.Filters.FieldMap() {
		res[k] = v
	}
	return res
}

func (f *collectionForm) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		IsValid bool                        `json:"is_valid"`
		Errors  forms.Errors                `json:"errors"`
		Fields  map[string]*forms.FormField `json:"fields"`
	}{
		IsValid: f.IsValid(),
		Errors:  f.Errors(),
		Fields:  f.FieldMap(),
	})
}

func (f *collectionForm) Get(name string) *forms.FormField {
	if _, ok := f.filterFields[name]; ok {
		return f.Filters.Get(name)
	}
	return f.Form.Get(name)
}

func (f *collectionForm) Bind() {
	f.Form.Bind()

	c, _ := f.Context().Value(ctxCollectionFormKey{}).(*bookmarks.Collection)
	if c != nil {
		f.Get("name").SetValidators(forms.Trim, forms.RequiredOrNil)
	}

	// Some default values
	f.Get("is_pinned").Set(false)
}

// BindQueryString bind this form from a request's query string
// without performing validation.
func (f *collectionForm) BindQueryString(values url.Values) {
	for _, field := range f.Fields() {
		if v, ok := values[field.Name()]; ok {
			for _, x := range v {
				field.UnmarshalText([]byte(x)) //nolint:errcheck
			}
		}
	}
}

func (f *collectionForm) setCollection(c *bookmarks.Collection) {
	ctx := context.WithValue(f.Context(), ctxCollectionFormKey{}, c)
	f.SetContext(ctx)

	for _, field := range f.Fields() {
		switch n := field.Name(); n {
		case "name":
			field.Set(c.Name)
		case "is_pinned":
			field.Set(c.IsPinned)
		case "search":
			field.Set(c.Filters.Search)
		case "title":
			field.Set(c.Filters.Title)
		case "author":
			field.Set(c.Filters.Author)
		case "site":
			field.Set(c.Filters.Site)
		case "type":
			field.Set(c.Filters.Type)
		case "labels":
			field.Set(c.Filters.Labels)
		case "is_marked":
			if c.Filters.IsMarked == nil {
				field.Set(nil)
				continue
			}
			field.Set(*c.Filters.IsMarked)
		case "is_archived":
			if c.Filters.IsArchived == nil {
				field.Set(nil)
				continue
			}
			field.Set(*c.Filters.IsArchived)
		case "is_loaded":
			if c.Filters.IsLoaded == nil {
				field.Set(nil)
				continue
			}
			field.Set(*c.Filters.IsLoaded)
		case "has_errors":
			if c.Filters.HasErrors == nil {
				field.Set(nil)
				continue
			}
			field.Set(*c.Filters.HasErrors)
		case "has_labels":
			if c.Filters.HasLabels == nil {
				field.Set(nil)
				continue
			}
			field.Set(*c.Filters.HasLabels)
		case "range_start":
			field.Set(c.Filters.RangeStart)
		case "range_end":
			field.Set(c.Filters.RangeEnd)
		}
	}
}

func (f *collectionForm) createCollection(userID int) (*bookmarks.Collection, error) {
	if !f.IsBound() {
		return nil, errors.New("form is not bound")
	}

	c := &bookmarks.Collection{
		UserID: &userID,
		Name:   f.Get("name").String(),
		Filters: bookmarks.CollectionFilters{
			Search:     f.Get("search").String(),
			Title:      f.Get("title").String(),
			Author:     f.Get("author").String(),
			Site:       f.Get("site").String(),
			Labels:     f.Get("labels").String(),
			Type:       f.Get("type").String(),
			IsMarked:   nil,
			IsArchived: nil,
			IsLoaded:   nil,
			HasErrors:  nil,
			HasLabels:  nil,
			RangeStart: f.Get("range_start").String(),
			RangeEnd:   f.Get("range_end").String(),
		},
	}

	if !f.Get("is_marked").IsNil() {
		v := f.Get("is_marked").Value().(bool)
		c.Filters.IsMarked = &v
	}

	if !f.Get("is_archived").IsNil() {
		v := f.Get("is_archived").Value().(bool)
		c.Filters.IsArchived = &v
	}

	if !f.Get("is_loaded").IsNil() {
		v := f.Get("is_loaded").Value().(bool)
		c.Filters.IsLoaded = &v
	}
	if !f.Get("has_errors").IsNil() {
		v := f.Get("has_errors").Value().(bool)
		c.Filters.HasErrors = &v
	}
	if !f.Get("has_labels").IsNil() {
		v := f.Get("has_labels").Value().(bool)
		c.Filters.HasLabels = &v
	}

	err := bookmarks.Collections.Create(c)
	if err != nil {
		f.AddErrors("", forms.ErrUnexpected)
	}
	return c, err
}

func (f *collectionForm) updateCollection(c *bookmarks.Collection) (res map[string]interface{}, err error) {
	if !f.IsBound() {
		err = errors.New("form is not bound")
		return
	}

	res = map[string]interface{}{}
	current := c.Flatten()
	updated := c.Flatten()

	forms.Validate(f.Filters)
	for _, field := range f.Fields() {
		n := field.Name()
		switch n {
		case "name", "is_pinned":
			if field.IsBound() {
				updated[n] = field.Value()
			}
		default:
			updated[n] = field.Value()
		}
	}

	updateMap := map[string]interface{}{}
	updateMap["filters"] = filterMap{}
	needsFilters := false
	for k, v := range updated {
		if v != current[k] {
			res[k] = v
		}

		_, inFilters := f.filterFields[k]
		needsFilters = needsFilters || (inFilters && v != current[k])

		if inFilters {
			updateMap["filters"].(filterMap)[k] = v
			continue
		}

		if v == current[k] {
			continue
		}
		updateMap[k] = v
	}

	if !needsFilters {
		delete(updateMap, "filters")
	}

	if len(res) > 0 {
		res["updated"] = time.Now()
		updateMap["updated"] = res["updated"]
		if err = c.Update(updateMap); err != nil {
			f.AddErrors("", forms.ErrUnexpected)
			return
		}
	}
	res["id"] = c.UID
	return
}

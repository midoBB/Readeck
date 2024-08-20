// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package importer

import (
	"encoding/json"
	"errors"
	"io"
	"time"

	"codeberg.org/readeck/readeck/internal/db/types"
	"codeberg.org/readeck/readeck/pkg/forms"
)

type goodlinksAdapter struct {
	idx   int
	Items []goodlinksItem `json:"items"`
}

type goodlinksItem struct {
	Link    string        `json:"url"`
	Title   string        `json:"title"`
	AddedAt int64         `json:"addedAt"`
	Tags    types.Strings `json:"tags"`
	Starred bool          `json:"starred"`
}

func (bi *goodlinksItem) URL() string {
	return bi.Link
}

func (bi *goodlinksItem) Meta() (*BookmarkMeta, error) {
	return &BookmarkMeta{
		Title:    bi.Title,
		Created:  time.Unix(bi.AddedAt, 0).UTC(),
		Labels:   bi.Tags,
		IsMarked: bi.Starred,
	}, nil
}

func (adapter *goodlinksAdapter) Name(tr forms.Translator) string {
	return tr.Gettext("GoodLinks Export File")
}

func (adapter *goodlinksAdapter) Form() forms.Binder {
	return newMultipartForm()
}

func (adapter *goodlinksAdapter) Params(form forms.Binder) ([]byte, error) {
	f := form.(*multipartForm).dataReader()
	if f == nil {
		return nil, errors.New("unable to load content")
	}
	defer f.Close() //nolint:errcheck

	dec := json.NewDecoder(f)
	err := dec.Decode(&adapter.Items)
	if err != nil {
		form.AddErrors("data", forms.Gettext("Empty or invalid import file"))
		return nil, nil
	}

	return json.Marshal(adapter)
}

func (adapter *goodlinksAdapter) LoadData(data []byte) error {
	return json.Unmarshal(data, adapter)
}

func (adapter *goodlinksAdapter) Next() (BookmarkImporter, error) {
	if adapter.idx+1 > len(adapter.Items) {
		return nil, io.EOF
	}

	adapter.idx++
	return &adapter.Items[adapter.idx-1], nil
}

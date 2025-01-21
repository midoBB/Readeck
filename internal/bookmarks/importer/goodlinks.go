// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package importer

import (
	"context"
	"encoding/json"
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
	AddedAt float64       `json:"addedAt"`
	Tags    types.Strings `json:"tags"`
	Starred bool          `json:"starred"`
}

func (bi *goodlinksItem) URL() string {
	return bi.Link
}

func (bi *goodlinksItem) Meta() (*BookmarkMeta, error) {
	return &BookmarkMeta{
		Title:    bi.Title,
		Created:  time.Unix(int64(bi.AddedAt), 0).UTC(),
		Labels:   bi.Tags,
		IsMarked: bi.Starred,
	}, nil
}

func (adapter *goodlinksAdapter) Name(tr forms.Translator) string {
	return tr.Gettext("GoodLinks Export File")
}

func (adapter *goodlinksAdapter) Form() forms.Binder {
	return forms.Must(
		context.Background(),
		forms.NewFileField("data", forms.Required),
	)
}

func (adapter *goodlinksAdapter) Params(form forms.Binder) ([]byte, error) {
	if !form.IsValid() {
		return nil, nil
	}

	reader, err := form.Get("data").(*forms.FileField).V().Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close() //nolint:errcheck

	dec := json.NewDecoder(reader)
	err = dec.Decode(&adapter.Items)
	if err != nil {
		form.AddErrors("data", errInvalidFile)
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

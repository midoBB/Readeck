// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package importer

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"

	"codeberg.org/readeck/readeck/pkg/forms"
)

type textAdapter struct {
	idx  int
	URLs []string `json:"url_list"`
}

func (adapter *textAdapter) Name(tr forms.Translator) string {
	return tr.Gettext("Text File")
}

func (adapter *textAdapter) Form() forms.Binder {
	return newMultipartForm()
}

func (adapter *textAdapter) Params(form forms.Binder) ([]byte, error) {
	f := form.(*multipartForm).dataReader()
	if f == nil {
		return nil, errors.New("unable to load content")
	}
	defer f.Close() //nolint:errcheck

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		b, err := newURLBookmark(strings.TrimSpace(scanner.Text()))
		if err == nil && b.URL() != "" && !slices.Contains(adapter.URLs, b.URL()) {
			adapter.URLs = append(adapter.URLs, b.URL())
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(adapter.URLs) == 0 {
		form.AddErrors("data", forms.Gettext("Empty or invalid import file"))
		return nil, nil
	}

	slices.Reverse(adapter.URLs)
	return json.Marshal(adapter)
}

func (adapter *textAdapter) LoadData(data []byte) error {
	return json.Unmarshal(data, adapter)
}

func (adapter *textAdapter) Next() (BookmarkImporter, error) {
	if adapter.idx+1 > len(adapter.URLs) {
		return nil, io.EOF
	}

	adapter.idx++
	return newURLBookmark(adapter.URLs[adapter.idx-1])
}

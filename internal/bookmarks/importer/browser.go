// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package importer

import (
	"encoding/json"
	"errors"
	"io"
	"slices"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"

	"codeberg.org/readeck/readeck/pkg/forms"
)

type browserAdapter struct {
	idx  int
	URLs []string `json:"url_list"`
}

func (adapter *browserAdapter) Form() forms.Binder {
	return newMultipartForm()
}

func (adapter *browserAdapter) Params(form forms.Binder) ([]byte, error) {
	f := form.(*multipartForm).dataReader()
	if f == nil {
		return nil, errors.New("unable to load content")
	}
	defer f.Close() //nolint:errcheck

	root, err := html.Parse(f)
	if err != nil {
		form.AddErrors("data", errors.New("unabled to read HTML content"), err)
		return nil, nil
	}

	for _, n := range dom.QuerySelectorAll(root, "dt > a[href]") {
		b, err := newURLBookmark(dom.GetAttribute(n, "href"))
		if err != nil {
			continue
		}
		adapter.URLs = append(adapter.URLs, b.URL())
	}

	slices.Reverse(adapter.URLs)
	return json.Marshal(adapter)
}

func (adapter *browserAdapter) LoadData(data []byte) error {
	return json.Unmarshal(data, adapter)
}

func (adapter *browserAdapter) Next() (bookmarkImporter, error) {
	if adapter.idx+1 > len(adapter.URLs) {
		return nil, io.EOF
	}

	adapter.idx++
	return newURLBookmark(adapter.URLs[adapter.idx-1])
}

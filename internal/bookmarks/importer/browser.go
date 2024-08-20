// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package importer

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"

	"codeberg.org/readeck/readeck/internal/db/types"
	"codeberg.org/readeck/readeck/pkg/forms"
)

type browserAdapter struct {
	idx   int
	Items []browserBookmarkItem `json:"items"`
}

type browserBookmarkItem struct {
	Link    string        `json:"url"`
	Title   string        `json:"title"`
	Created time.Time     `json:"created"`
	Labels  types.Strings `json:"labels"`
}

func (bi *browserBookmarkItem) URL() string {
	return bi.Link
}

func (bi *browserBookmarkItem) Meta() (*BookmarkMeta, error) {
	return &BookmarkMeta{
		Title:   bi.Title,
		Created: bi.Created,
		Labels:  bi.Labels,
	}, nil
}

func (adapter *browserAdapter) Name(tr forms.Translator) string {
	return tr.Gettext("Browser Bookmarks")
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
		form.AddErrors("data", forms.Gettext("Unabled to read HTML content"), err)
		return nil, nil
	}

	for _, n := range dom.QuerySelectorAll(root, "dt > a[href]") {
		uri, err := url.Parse(dom.GetAttribute(n, "href"))
		if err != nil {
			continue
		}
		uri.Fragment = ""
		if !slices.Contains(allowedSchemes, uri.Scheme) {
			continue
		}

		if slices.ContainsFunc(adapter.Items, func(bi browserBookmarkItem) bool {
			return bi.Link == uri.String()
		}) {
			continue
		}

		item := browserBookmarkItem{
			Created: time.Now(),
			Link:    uri.String(),
			Title:   strings.TrimSpace(dom.TextContent(n)),
			Labels:  types.Strings{},
		}

		if dom.HasAttribute(n, "add_date") {
			if ts, err := strconv.Atoi(dom.GetAttribute(n, "add_date")); err == nil {
				item.Created = time.Unix(int64(ts), 0)
			}
		}

		// Fetch labels from the TAGS attribute when present (pinboard, maybe others)
		for _, label := range strings.Split(dom.GetAttribute(n, "tags"), ",") {
			if label = strings.TrimSpace(label); label != "" {
				item.Labels = append(item.Labels, label)
			}
		}

		adapter.Items = append(adapter.Items, item)
	}

	if len(adapter.Items) == 0 {
		form.AddErrors("data", forms.Gettext("Empty or invalid import file"))
		return nil, nil
	}

	slices.Reverse(adapter.Items)
	return json.Marshal(adapter)
}

func (adapter *browserAdapter) LoadData(data []byte) error {
	return json.Unmarshal(data, adapter)
}

func (adapter *browserAdapter) Next() (BookmarkImporter, error) {
	if adapter.idx+1 > len(adapter.Items) {
		return nil, io.EOF
	}

	adapter.idx++
	return &adapter.Items[adapter.idx-1], nil
}

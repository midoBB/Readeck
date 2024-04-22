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

type pocketFileAdapter struct {
	idx   int
	Items []pocketBookmarkItem
}

type pocketBookmarkItem struct {
	Link       string        `json:"url"`
	Title      string        `json:"title"`
	Labels     types.Strings `json:"labels"`
	IsArchived bool          `json:"is_archived"`
	Created    time.Time     `json:"created"`
}

func (bi *pocketBookmarkItem) URL() string {
	return bi.Link
}

func (bi *pocketBookmarkItem) Meta() (*BookmarkMeta, error) {
	return &BookmarkMeta{
		Title:      bi.Title,
		Labels:     bi.Labels,
		IsArchived: bi.IsArchived,
		Created:    bi.Created,
	}, nil
}

func (adapter *pocketFileAdapter) Form() forms.Binder {
	return newMultipartForm()
}

func (adapter *pocketFileAdapter) Params(form forms.Binder) ([]byte, error) {
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

	section := ""
	for _, node := range dom.QuerySelectorAll(root, "body>h1, body>ul") {
		if dom.TagName(node) == "h1" {
			section = strings.ToLower(strings.TrimSpace(dom.TextContent(node)))
			continue
		}

		for _, n := range dom.QuerySelectorAll(node, "li>a[href]") {
			uri, err := url.Parse(dom.GetAttribute(n, "href"))
			if err != nil {
				continue
			}
			uri.Fragment = ""
			if !slices.Contains(allowedSchemes, uri.Scheme) {
				continue
			}

			if slices.ContainsFunc(adapter.Items, func(bi pocketBookmarkItem) bool {
				return bi.Link == uri.String()
			}) {
				continue
			}

			item := pocketBookmarkItem{
				Created:    time.Now(),
				Link:       uri.String(),
				IsArchived: section == "read archive",
				Labels:     types.Strings{},
			}

			title := strings.TrimSpace(dom.TextContent(n))
			if title != item.Link {
				item.Title = title
			}

			for _, label := range strings.Split(dom.GetAttribute(n, "tags"), ",") {
				if label = strings.TrimSpace(label); label != "" {
					item.Labels = append(item.Labels, label)
				}
			}

			ts, err := strconv.Atoi(dom.GetAttribute(n, "time_added"))
			if err == nil {
				item.Created = time.Unix(int64(ts), 0)
			}

			adapter.Items = append(adapter.Items, item)
		}
	}

	if len(adapter.Items) == 0 {
		form.AddErrors("data", forms.Gettext("Empty or invalid import file"))
		return nil, nil
	}

	slices.SortFunc(adapter.Items, func(a, b pocketBookmarkItem) int {
		return int(a.Created.Unix()) - int(b.Created.Unix())
	})

	return json.Marshal(adapter)
}

func (adapter *pocketFileAdapter) LoadData(data []byte) error {
	return json.Unmarshal(data, adapter)
}

func (adapter *pocketFileAdapter) Next() (BookmarkImporter, error) {
	if adapter.idx+1 > len(adapter.Items) {
		return nil, io.EOF
	}

	adapter.idx++
	return &adapter.Items[adapter.idx-1], nil
}

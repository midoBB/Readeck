// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bookmarks

import (
	"context"
	"net/url"
	"slices"

	"github.com/go-shiori/dom"

	"codeberg.org/readeck/readeck/pkg/extract"
)

var ctxExtractLinksKey struct{}

// CleanDomProcessor is a last pass of cleaning on the resulting DOM node.
// It removes unwanted attributes, empty tags and set some defaults.
func CleanDomProcessor(m *extract.ProcessMessage, next extract.Processor) extract.Processor {
	if m.Step() != extract.StepDom {
		return next
	}

	if m.Dom == nil {
		return next
	}

	m.Log.Debug("cleaning resulting DOM")

	bleach.clean(m.Dom)
	bleach.removeEmptyNodes(m.Dom)
	bleach.setLinkRel(m.Dom)

	return next
}

// extractLinksProcessor extracts all the web links (http and https) in the page
// and store the list in the extractor context.
func extractLinksProcessor(m *extract.ProcessMessage, next extract.Processor) extract.Processor {
	if m.Step() != extract.StepDom {
		return next
	}

	if m.Dom == nil {
		return next
	}

	m.Log.Debug("extract links from content")
	links := BookmarkLinks{}
	seen := map[string]*extract.Drop{}

	for _, node := range dom.QuerySelectorAll(m.Dom, "a[href]") {
		href := dom.GetAttribute(node, "href")
		URL, err := url.Parse(href)
		if err != nil {
			continue
		}
		URL.Fragment = ""

		if _, ok := seen[URL.String()]; ok {
			continue
		}

		d := extract.NewDrop(URL)
		if d.URL.String() == m.Extractor.Drop().URL.String() {
			continue
		}

		if URL.Scheme == "http" || URL.Scheme == "https" {
			seen[URL.String()] = extract.NewDrop(URL)
			links = append(links, BookmarkLink{URL: URL.String(), Domain: d.Domain})
		}
	}

	links = slices.CompactFunc(links, func(a, b BookmarkLink) bool {
		return a.URL == b.URL
	})

	m.Extractor.Context = context.WithValue(m.Extractor.Context, ctxExtractLinksKey, links)
	return next
}

func GetExtractedLinks(ctx context.Context) BookmarkLinks {
	if links, ok := ctx.Value(ctxExtractLinksKey).(BookmarkLinks); ok {
		return links
	}
	return BookmarkLinks{}
}

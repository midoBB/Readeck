// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// Package linkheader is a parser for "Link" HTTP header values.
package linkheader

import (
	"net/http"
	"strings"
)

// Link is a parsed link header value.
type Link struct {
	URL   string
	Rel   string
	Type  string
	Title string
}

// ParseLink parses the "Link" HTTP header and returns a list of [Link].
func ParseLink(header http.Header) (links []Link) {
	for _, s := range header[http.CanonicalHeaderKey("link")] {
		for link := range strings.SplitSeq(s, ",") {
			link = strings.TrimSpace(link)
			parts := strings.Split(link, ";")
			if parts[0][0] != '<' || parts[0][len(parts[0])-1] != '>' {
				continue
			}
			l := Link{
				URL: strings.Trim(parts[0], "<>"),
			}

			for _, param := range parts[1:] {
				p := strings.Split(strings.TrimSpace(param), "=")
				if len(p) < 2 {
					continue
				}
				name, value := p[0], strings.Trim(p[1], `"`)
				switch name {
				case "rel":
					l.Rel = value
				case "type":
					l.Type = value
				case "title":
					l.Title = value
				}
			}

			links = append(links, l)
		}
	}

	return
}

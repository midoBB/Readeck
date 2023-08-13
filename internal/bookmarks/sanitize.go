// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bookmarks

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

// bleachPolicy holds the cleaning rules and provides methods to
// perform the DOM cleaning.
type bleachPolicy struct {
	blockAttrs []*regexp.Regexp
}

var selfClosingTags = map[string]struct{}{
	"area":     {},
	"base":     {},
	"br":       {},
	"col":      {},
	"command":  {},
	"embed":    {},
	"hr":       {},
	"img":      {},
	"input":    {},
	"keygen":   {},
	"link":     {},
	"menuitem": {},
	"meta":     {},
	"param":    {},
	"source":   {},
	"track":    {},
	"wbr":      {},
}

var bleach = bleachPolicy{
	blockAttrs: []*regexp.Regexp{
		regexp.MustCompile(`^class$`),
		regexp.MustCompile(`^data-`),
		regexp.MustCompile(`^on[a-z]+`),
		regexp.MustCompile(`^(rel|srcset|sizes)$`),
	},
}

// clean discards unwanted attributes from all nodes.
func (p bleachPolicy) clean(node *html.Node) {
	for i := len(node.Attr) - 1; i >= 0; i-- {
		k := node.Attr[i].Key
		for _, r := range p.blockAttrs {
			if r.MatchString(k) {
				dom.RemoveAttribute(node, k)
				break
			}
		}
	}

	for child := dom.FirstElementChild(node); child != nil; child = dom.NextElementSibling(child) {
		p.clean(child)
	}
}

// removeEmptyNodes removes the nodes that are empty.
// empty means: no child nodes, no attributes and no text content.
func (p bleachPolicy) removeEmptyNodes(top *html.Node) {
	nodes := dom.QuerySelectorAll(top, "*")
	dom.RemoveNodes(nodes, func(node *html.Node) bool {
		// Keep self closing tags
		if _, ok := selfClosingTags[dom.TagName(node)]; ok {
			return false
		}

		// Keep <a name> tags
		if dom.TagName(node) == "a" && dom.GetAttribute(node, "name") != "" {
			return false
		}

		// Keep nodes with children
		if len(dom.Children(node)) > 0 {
			return false
		}

		// Keep nodes with any text
		if strings.TrimFunc(dom.TextContent(node), isHTMLSpace) != "" {
			return false
		}

		// Remove node
		return true
	})
}

// setLinkRel adds a default "rel" attribute on all "a" tags.
func (p bleachPolicy) setLinkRel(top *html.Node) {
	dom.ForEachNode(dom.QuerySelectorAll(top, "a[href]"), func(node *html.Node, _ int) {
		dom.SetAttribute(node, "rel", "nofollow noopener noreferrer")
	})
}

// isHTMLSpace returns true if a rune is a space as defined by the HTML spec
func isHTMLSpace(r rune) bool {
	if uint32(r) <= unicode.MaxLatin1 {
		switch r {
		case '\t', '\n', '\r', ' ':
			return true
		}
		return false
	}
	return false
}

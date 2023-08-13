// SPDX-FileCopyrightText: © 2020 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package contents

import (
	"bytes"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/net/html"

	"github.com/antchfx/htmlquery"
	"github.com/go-shiori/dom"
	"github.com/go-shiori/go-readability"

	"github.com/readeck/readeck/pkg/extract"
	"github.com/readeck/readeck/pkg/extract/srcset"
)

var (
	rxSpace   = regexp.MustCompile(`[ ]+`)
	rxNewLine = regexp.MustCompile(`\r?\n\s*(\r?\n)+`)
)

// Readability is a processor that executes readability on the drop content.
func Readability(options ...func(*readability.Parser)) extract.Processor {
	return func(m *extract.ProcessMessage, next extract.Processor) extract.Processor {
		if m.Step() != extract.StepDom {
			return next
		}

		if m.Extractor.Drop().IsMedia() {
			m.ResetContent()
			return next
		}

		fixNoscriptImages(m.Dom)
		convertPictureNodes(m.Dom, m)

		parser := readability.NewParser()
		for _, f := range options {
			f(&parser)
		}

		article, err := parser.ParseDocument(m.Dom, m.Extractor.Drop().URL)
		if err != nil {
			m.Log.WithError(err).Error("readability error")
			m.ResetContent()
			return next
		}

		if article.Node == nil {
			m.Log.Error("could not extract content")
			m.ResetContent()
			return next
		}

		m.Log.Debug("readability on contents")

		doc := &html.Node{Type: html.DocumentNode}
		body := dom.CreateElement("body")
		doc.AppendChild(body)
		dom.AppendChild(body, article.Node)
		// final cleanup
		removeEmbeds(body)
		fixImages(body, m)

		// Simplify the top hierarchy
		node := findFirstContentNode(body)
		if node != body.FirstChild {
			dom.ReplaceChild(body, node, body.FirstChild)
		}

		// Ensure we always start with a <section>
		encloseArticle(body)

		m.Dom = doc

		return next
	}
}

// Text is a processor that sets the pure text content of the final HTML.
func Text(m *extract.ProcessMessage, next extract.Processor) extract.Processor {
	if m.Step() != extract.StepPostProcess {
		return next
	}

	if len(m.Extractor.HTML) == 0 {
		return next
	}
	if !m.Extractor.Drop().IsHTML() {
		return next
	}

	m.Log.Debug("get text content")

	doc, _ := html.Parse(bytes.NewReader(m.Extractor.HTML))
	text := dom.TextContent(doc)

	text = rxSpace.ReplaceAllString(text, " ")
	text = rxNewLine.ReplaceAllString(text, "\n\n")
	text = strings.TrimSpace(text)

	m.Extractor.Text = text
	return next
}

func findFirstContentNode(node *html.Node) *html.Node {
	children := dom.ChildNodes(node)
	count := 0
	for _, x := range children {
		if x.Type == html.TextNode && strings.TrimSpace(x.Data) != "" {
			count++
		} else if x.Type == html.ElementNode {
			count++
		}

	}

	if count > 1 || dom.FirstElementChild(node) == nil {
		return node
	}

	return findFirstContentNode(dom.FirstElementChild(node))
}

func encloseArticle(top *html.Node) {
	children := dom.ChildNodes(top)

	if len(children) == 1 {
		node := children[0]
		switch node.Type {
		case html.TextNode:
			section := dom.CreateElement("section")
			dom.AppendChild(node.Parent, section)
			dom.AppendChild(section, node)
		case html.ElementNode:
			if node.Data == "div" {
				node.Data = "section"
			} else {
				section := dom.CreateElement("section")
				dom.AppendChild(node.Parent, section)
				dom.AppendChild(section, node)
			}
		}
	} else {
		section := dom.CreateElement("section")
		dom.AppendChild(top, section)
		for _, x := range children {
			dom.AppendChild(section, x)
		}
	}
}

func removeEmbeds(top *html.Node) {
	dom.RemoveNodes(dom.GetAllNodesWithTag(top, "object", "embed", "iframe", "video", "audio"), nil)
}

func fixNoscriptImages(top *html.Node) {
	// A bug in readability prevents us to extract images.
	// It does move the noscript content when it's a single image
	// but only when the noscript previous sibling is an image.
	// This will replace the noscript content with the image
	// in the other case.

	noscripts := dom.GetElementsByTagName(top, "noscript")
	dom.ForEachNode(noscripts, func(noscript *html.Node, _ int) {
		noscriptContent := dom.TextContent(noscript)
		tmpDoc, err := html.Parse(strings.NewReader(noscriptContent))
		if err != nil {
			return
		}

		tmpBody := dom.GetElementsByTagName(tmpDoc, "body")[0]
		if !isSingleImage(tmpBody) {
			return
		}

		// Sometimes, the image is *after* the noscript tag.
		// Let's move it before so the next step can detect it.
		nextElement := dom.NextElementSibling(noscript)
		if nextElement != nil && isSingleImage(nextElement) {
			if noscript.Parent != nil {
				noscript.Parent.InsertBefore(dom.Clone(nextElement, true), noscript)
				noscript.Parent.RemoveChild(nextElement)
			}
		}

		prevElement := dom.PreviousElementSibling(noscript)
		if prevElement == nil || !isSingleImage(prevElement) {
			dom.ReplaceChild(noscript.Parent, dom.FirstElementChild(tmpBody), noscript)
		}
	})
}

func isSingleImage(node *html.Node) bool {
	if dom.TagName(node) == "img" {
		return true
	}
	children := dom.Children(node)
	textContent := dom.TextContent(node)
	if len(children) != 1 || strings.TrimSpace(textContent) != "" {
		return false
	}

	return isSingleImage(children[0])
}

func convertPictureNodes(top *html.Node, _ *extract.ProcessMessage) {
	nodes := dom.GetElementsByTagName(top, "picture")
	dom.ForEachNode(nodes, func(node *html.Node, _ int) {
		// A picture tag contains zero or more <source> elements
		// and an <img> element. We take all the srcset values from
		// each <source>, add them to the <img> srcset and then replace
		// the picture element with the img.

		// First get or create an img element
		imgs := dom.GetElementsByTagName(node, "img")
		var img *html.Node
		if len(imgs) == 0 {
			img = dom.CreateElement("img")
		} else {
			img = imgs[0]
		}

		// Collect all the srcset attributes
		set := []string{}
		sources := dom.GetElementsByTagName(node, "source")
		for _, n := range sources {
			if dom.HasAttribute(n, "srcset") {
				set = append(set, dom.GetAttribute(n, "srcset"))
			}
		}

		// Including the one in the <img> if present
		if dom.HasAttribute(img, "srcset") {
			set = append(set, dom.GetAttribute(img, "srcset"))
		}

		// Now mix them all together and replace the picture
		// element.
		dom.SetAttribute(img, "srcset", strings.Join(set, ", "))

		dom.ReplaceChild(node.Parent, img, node)
	})
}

func fixImages(top *html.Node, m *extract.ProcessMessage) {
	// Fix images with an srcset attribute and only keep the
	// best one.
	m.Log.Debug("fixing images")
	nodes, err := htmlquery.QueryAll(top, "//*[@srcset]")
	if err != nil {
		m.Log.WithError(err).Warn()
	}

	dom.ForEachNode(nodes, func(node *html.Node, _ int) {
		sourceSet := srcset.SourceSet{}
		for _, x := range srcset.Parse(dom.GetAttribute(node, "srcset")) {
			if x.Height > 3072 || x.Width > 3072 {
				continue
			}
			sourceSet = append(sourceSet, x)
		}
		sort.SliceStable(sourceSet, func(i, j int) bool {
			return sourceSet[i].Width > sourceSet[j].Width
		})

		if len(sourceSet) > 0 {
			dom.SetAttribute(node, "src", sourceSet[0].URL)
			dom.RemoveAttribute(node, "srcset")
			dom.RemoveAttribute(node, "width")
			dom.RemoveAttribute(node, "height")
		}
	})
}

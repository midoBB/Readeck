// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bleach

import "strings"

// elementMap is the map of all known elements
// and what they can be transformed to.
// A value of "-" means the elements must be removed.
// As per https://developer.mozilla.org/en-US/docs/Web/HTML/Element
var elementMap = map[string]string{
	"a":          "",
	"abbr":       "",
	"acronym":    "",
	"address":    "",
	"applet":     "-", // remove
	"area":       "",
	"article":    "",
	"aside":      "",
	"audio":      "-", // remove
	"b":          "",
	"base":       "-", // remove
	"bdi":        "",
	"bdo":        "",
	"big":        "",
	"blockquote": "",
	"body":       "",
	"br":         "",
	"button":     "-", // remove
	"canvas":     "-", // remove
	"caption":    "",
	"center":     "",
	"cite":       "",
	"code":       "",
	"col":        "",
	"colgroup":   "",
	"data":       "",
	"datalist":   "",
	"dd":         "",
	"del":        "",
	"details":    "",
	"dfn":        "",
	"dialog":     "-", // remove
	"dir":        "",
	"div":        "",
	"dl":         "",
	"dt":         "",
	"em":         "",
	"embed":      "-", // remove
	"fieldset":   "div",
	"figcaption": "",
	"figure":     "",
	"font":       "span",
	"footer":     "",
	"form":       "div",
	"frame":      "-", // remove
	"frameset":   "-", // remove
	"h1":         "",
	"head":       "-", // remove
	"header":     "",
	"hgroup":     "",
	"hr":         "",
	"html":       "",
	"i":          "",
	"iframe":     "-", // remove
	"image":      "",
	"img":        "",
	"input":      "-", // remove
	"ins":        "",
	"kbd":        "",
	"label":      "",
	"legend":     "",
	"li":         "",
	"link":       "-", // remove
	"main":       "",
	"map":        "",
	"mark":       "",
	"marquee":    "",
	"menu":       "",
	"menuitem":   "",
	"meta":       "-", // remove
	"meter":      "",
	"nav":        "",
	"nobr":       "",
	"noembed":    "div",
	"noframes":   "div",
	"noscript":   "div",
	"object":     "-", // remove
	"ol":         "",
	"optgroup":   "",
	"option":     "",
	"output":     "",
	"p":          "",
	"param":      "-", // remove
	"picture":    "",
	"plaintext":  "",
	"portal":     "-", // remove
	"pre":        "",
	"progress":   "",
	"q":          "",
	"rb":         "",
	"rp":         "",
	"rt":         "",
	"rtc":        "",
	"ruby":       "",
	"s":          "",
	"samp":       "",
	"script":     "-", // remove
	"search":     "",
	"section":    "",
	"select":     "-", // remove
	"slot":       "-", // remove
	"small":      "",
	"source":     "-", // remove
	"span":       "",
	"strike":     "",
	"strong":     "",
	"style":      "-", // remove
	"sub":        "",
	"summary":    "",
	"sup":        "",
	"table":      "",
	"tbody":      "",
	"td":         "",
	"template":   "-", // remove
	"textarea":   "-", // remove
	"tfoot":      "",
	"th":         "",
	"thead":      "",
	"time":       "",
	"title":      "-", // remove
	"tr":         "",
	"track":      "-", // remove
	"tt":         "",
	"u":          "",
	"ul":         "",
	"var":        "",
	"video":      "-", // remove
	"wbr":        "",
	"xmp":        "",
}

// ctrlReplacer is a string replacer for all control characters
var ctrlReplacer = func() *strings.Replacer {
	oldnew := []string{}
	for i := 0; i <= 31; i++ {
		oldnew = append(oldnew, string(rune(i)), "")
	}
	for i := 127; i <= 159; i++ {
		oldnew = append(oldnew, string(rune(i)), "")
	}

	return strings.NewReplacer(oldnew...)
}()

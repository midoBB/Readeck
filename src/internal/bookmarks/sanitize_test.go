// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bookmarks

import (
	"strconv"
	"strings"
	"testing"

	"github.com/go-shiori/dom"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		fn       func(*html.Node)
		fragment string
		expected string
	}{
		{
			bleach.clean,
			`<p id="id-p" class="test" data-test="1" onClick="bar" rel="link" srcset="url" sizes="x1">foo</p>`,
			`<body><p id="id-p">foo</p></body>`,
		},
		{
			bleach.removeEmptyNodes,
			`<p>test</p><span></span><br /><p>test 2</p>`,
			`<body><p>test</p><br/><p>test 2</p></body>`,
		},
		{
			bleach.removeEmptyNodes,
			`<p>test</p><div><span>
			</span></div><br /><p>test 😺</p>`,
			`<body><p>test</p><br/><p>test 😺</p></body>`,
		},
		{
			bleach.removeEmptyNodes,
			`<video controls><source src="foo"></video>`,
			`<body><video controls=""><source src="foo"/></video></body>`,
		},
		{
			bleach.removeEmptyNodes,
			`<p><a name="foo"></a></p><p>test</p>`,
			`<body><p><a name="foo"></a></p><p>test</p></body>`,
		},
		{
			bleach.setLinkRel,
			`<p><a href="foo">link</a></p>`,
			`<body><p><a href="foo" rel="nofollow noopener noreferrer">link</a></p></body>`,
		},
		{
			bleach.setLinkRel,
			`<p><a name="foo"></a></p>`,
			`<body><p><a name="foo"></a></p></body>`,
		},
		{
			func(n *html.Node) {
				bleach.clean(n)
				bleach.removeEmptyNodes(n)
				bleach.setLinkRel(n)
			},
			`<p id="id-p" class="test" data-test="1" onClick="bar" rel="link" srcset="url" sizes="x1"><a name="foo"></a></p><p><a href="foo">link</a><span></span><hr></p>`,
			`<body><p id="id-p"><a name="foo"></a></p><p><a href="foo" rel="nofollow noopener noreferrer">link</a></p><hr/></body>`,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r := strings.NewReader(test.fragment)
			node, err := html.Parse(r)
			if err != nil {
				panic(err)
			}

			test.fn(node)
			assert.Equal(t, test.expected, dom.OuterHTML(dom.QuerySelector(node, "body")))
		})
	}

}

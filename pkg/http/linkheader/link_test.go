// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package linkheader_test

import (
	"net/http"
	"strconv"
	"testing"

	"codeberg.org/readeck/readeck/pkg/http/linkheader"
	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	tests := []struct {
		headers  []string
		expected []linkheader.Link
	}{
		{
			[]string{"</>"},
			[]linkheader.Link{
				{URL: "/"},
			},
		},
		{
			[]string{` <https://example.org/?test=1>; rel=next; title="test title" `},
			[]linkheader.Link{
				{URL: "https://example.org/?test=1", Rel: "next", Title: "test title"},
			},
		},
		{
			[]string{` <https://example.org/?test=1>; rel=next; title="test title" ,  <https://example.net/>; rel=original; foo="test"  `},
			[]linkheader.Link{
				{URL: "https://example.org/?test=1", Rel: "next", Title: "test title"},
				{URL: "https://example.net/", Rel: "original"},
			},
		},
		{
			[]string{
				`<https://example.net/>; rel=original; foo="test"  `,
				`<https://example.org/>; rel=alternate; type="text/xml"; garbage`,
			},
			[]linkheader.Link{
				{URL: "https://example.net/", Rel: "original"},
				{URL: "https://example.org/", Rel: "alternate", Type: "text/xml"},
			},
		},
		{
			[]string{
				`<https://example.net/>; rel=original; foo="test", invalid;rel="next"  `,
				`<https://example.org/>;rel=alternate;type="text/xml";garbage`,
			},
			[]linkheader.Link{
				{URL: "https://example.net/", Rel: "original"},
				{URL: "https://example.org/", Rel: "alternate", Type: "text/xml"},
			},
		},
		{
			[]string{
				`<https://example.net/>; rel=original; foo="test",`,
				`<https://example.org/>;rel=alternate;type="text/xml";garbage`,
			},
			[]linkheader.Link{
				{URL: "https://example.net/", Rel: "original"},
				{URL: "https://example.org/", Rel: "alternate", Type: "text/xml"},
			},
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			h := http.Header{}
			for _, l := range test.headers {
				h.Add("link", l)
			}
			links := linkheader.ParseLink(h)
			require.Equal(t, test.expected, links)
		})
	}
}

func TestNoHeader(t *testing.T) {
	h := http.Header{}
	links := linkheader.ParseLink(h)
	require.Empty(t, links)
}

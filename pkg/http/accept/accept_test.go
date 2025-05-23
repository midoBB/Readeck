// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package accept

import (
	"net/http"
	"testing"
)

var negotiateContentEncodingTests = []struct {
	s      string
	offers []string
	expect string
}{
	{"", []string{"identity", "gzip"}, "identity"},
	{"*;q=0", []string{"identity", "gzip"}, ""},
	{"gzip", []string{"identity", "gzip"}, "gzip"},
	{"gzip,br", []string{"br", "gzip"}, "br"},
}

func TestNegotiateContentEnoding(t *testing.T) {
	for _, tt := range negotiateContentEncodingTests {
		h := http.Header{"Accept-Encoding": {tt.s}}
		actual := NegotiateContentEncoding(h, tt.offers)
		if actual != tt.expect {
			t.Errorf("NegotiateContentEncoding(%q, %#v)=%q, want %q", tt.s, tt.offers, actual, tt.expect)
		}
	}
}

var negotiateContentTypeTests = []struct {
	s            string
	offers       []string
	defaultOffer string
	expect       string
}{
	{"text/html, */*;q=0", []string{"x/y"}, "", ""},
	{"text/html, */*", []string{"x/y"}, "", "x/y"},
	{"text/html, image/png", []string{"text/html", "image/png"}, "", "text/html"},
	{"text/html, image/png", []string{"image/png", "text/html"}, "", "image/png"},
	{"text/html, image/png; q=0.5", []string{"image/png"}, "", "image/png"},
	{"text/html, image/png; q=0.5", []string{"text/html"}, "", "text/html"},
	{"text/html, image/png; q=0.5", []string{"foo/bar"}, "", ""},
	{"text/html, image/png; q=0.5", []string{"image/png", "text/html"}, "", "text/html"},
	{"text/html, image/png; q=0.5", []string{"text/html", "image/png"}, "", "text/html"},
	{"text/html;q=0.5, image/png", []string{"image/png"}, "", "image/png"},
	{"text/html;q=0.5, image/png", []string{"text/html"}, "", "text/html"},
	{"text/html;q=0.5, image/png", []string{"image/png", "text/html"}, "", "image/png"},
	{"text/html;q=0.5, image/png", []string{"text/html", "image/png"}, "", "image/png"},
	{"image/png, image/*;q=0.5", []string{"image/jpg", "image/png"}, "", "image/png"},
	{"image/png, image/*;q=0.5", []string{"image/jpg"}, "", "image/jpg"},
	{"image/png, image/*;q=0.5", []string{"image/jpg", "image/gif"}, "", "image/jpg"},
	{"image/png, image/*", []string{"image/jpg", "image/gif"}, "", "image/jpg"},
	{"image/png, image/*", []string{"image/gif", "image/jpg"}, "", "image/gif"},
	{"image/png, image/*", []string{"image/gif", "image/png"}, "", "image/png"},
	{"image/png, image/*", []string{"image/png", "image/gif"}, "", "image/png"},
}

func TestNegotiateContentType(t *testing.T) {
	for _, tt := range negotiateContentTypeTests {
		h := http.Header{"Accept": {tt.s}}
		actual := NegotiateContentType(h, tt.offers, tt.defaultOffer)
		if actual != tt.expect {
			t.Errorf("NegotiateContentType(%q, %#v, %q)=%q, want %q", tt.s, tt.offers, tt.defaultOffer, actual, tt.expect)
		}
	}
}

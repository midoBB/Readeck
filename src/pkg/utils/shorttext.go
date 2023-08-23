// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package utils

import (
	"net/url"
	"path"
	"strings"
	"unicode"
)

// ShortText returns a string of maxChars maximum length. It attempts to cut between words
// when possible.
func ShortText(s string, maxChars int) string {
	runes := []rune(strings.TrimSpace(strings.Join(strings.Fields(s), " ")))
	if len(runes) <= maxChars {
		return string(runes)
	}

	res := &strings.Builder{}
	j := 0
	for i, word := range strings.FieldsFunc(s, unicode.IsSpace) {
		j += len(word)
		if j >= maxChars {
			if len(word) > maxChars {
				res.WriteString(word[0:maxChars])
			}
			break
		}
		if i > 0 {
			res.WriteString(" ")
		}
		res.WriteString(word)
	}
	res.WriteString("...")

	return res.String()
}

// ShortURL returns a string of maxChars maximum length where src is a URL. It attempts
// to nicely cut the path parts.
func ShortURL(s string, maxChars int) string {
	src, err := url.Parse(s)
	if err != nil {
		return ShortText(s, maxChars)
	}

	res := &strings.Builder{}
	maxChars = max(5, maxChars-len(src.Hostname()))
	res.WriteString(src.Hostname())
	res.WriteString("/")
	dir, file := path.Split(strings.Trim(src.Path, "/"))
	if len(dir+file) <= maxChars {
		res.WriteString(dir + file)
	} else {
		res.WriteString(".../" + ShortText(file, maxChars))
	}
	return res.String()
}

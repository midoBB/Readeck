package utils

import (
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

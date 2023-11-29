// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package utils_test

import (
	"strconv"
	"testing"

	"codeberg.org/readeck/readeck/pkg/utils"
	"github.com/stretchr/testify/require"
)

func TestShortText(t *testing.T) {
	tests := []struct {
		Text     string
		Expected string
	}{
		{"abcd", "abcd"},
		{"abcdefghij", "abcdefghij"},
		{"abcd abcd abcde", "abcd abcd..."},
		{"abcde abcde abcde", "abcde..."},
		{"abcdeabcdeabcde", "abcdeabcde..."},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res := utils.ShortText(test.Text, 10)
			require.Equal(t, test.Expected, res)
		})
	}
}

func TestShortURL(t *testing.T) {
	tests := []struct {
		Src      string
		Expected string
	}{
		{"https://example.net/abcd/abcd", "example.net/abcd/abcd"},
		{"https://example.net/abcd/abcd/efgh/ijkl/mnop/qrst/uvw/xyz", "example.net/.../xyz"},
		{"https://example.net/abcd/abcd/verylongpathpart/abcd", "example.net/.../abcd"},
		{"/test", "/test"},
		{"\b/test", "\b/test"},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res := utils.ShortURL(test.Src, 40)
			require.Equal(t, test.Expected, res)
		})
	}
}

func TestSlug(t *testing.T) {
	tests := []struct {
		Text     string
		Expected string
	}{
		{"abcd", "abcd"},
		{"abcd efgh _  xyz", "abcd-efgh-xyz"},
		{"c'est intéressant comme ça ?", "c-est-interessant-comme-ca"},
		{"Ogólnie znana teza głosi", "ogolnie-znana-teza-głosi"},
		{"Είναι πλέον κοινά παραδεκτό", "ειναι-πλεον-κοινα-παραδεκτο"},
		{"هناك حقيقة مثبتة منذ", "هناك-حقيقة-مثبتة-منذ"},
		{"🙂 happy 🐈", "happy"},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res := utils.Slug(test.Text)
			require.Equal(t, test.Expected, res)
		})
	}
}

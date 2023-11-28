// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package timetoken_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"codeberg.org/readeck/readeck/pkg/timetoken"
)

func TestRelativeTo(t *testing.T) {
	now, _ := time.Parse(time.DateTime, "2006-01-02 15:04:05")

	tests := []struct {
		s        string
		expected string
	}{
		{"now", "2006-01-02 15:04:05"},
		{"", "2006-01-02 15:04:05"},
		{"-2d", "2005-12-31 15:04:05"},
		{"+4w", "2006-01-30 15:04:05"},
		{"-2m", "2005-11-02 15:04:05"},
		{"-1y", "2005-01-02 15:04:05"},
		{"+2y", "2008-01-02 15:04:05"},
		{"+11d", "2006-01-13 15:04:05"},
		{"-11d", "2005-12-22 15:04:05"},
		{"2023-08-21 12:34:23", "2023-08-21 12:34:23"},
		{"2023-08-21", "2023-08-21 00:00:00"},
		{"21/8/2023", "2023-08-21 00:00:00"},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			token, err := timetoken.New(test.s)
			require.NoError(t, err)
			require.Equal(t, test.expected, token.RelativeTo(&now).Format(time.DateTime))
		})
	}
}

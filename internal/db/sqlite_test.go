// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package db

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSqliteDSN(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	tests := []struct {
		dsn      string
		expected string
		error    string
	}{
		{
			"sqlite3::memory:",
			"memory:/memory.db",
			"",
		},
		{
			"sqlite3:test.db",
			fmt.Sprintf("file:%s", filepath.Join(cwd, "test.db")),
			"",
		},
		{
			"sqlite3:/path/to/test.db",
			"file:/path/to/test.db",
			"",
		},
		{
			"sqlite3:///path/to/test.db",
			"file:/path/to/test.db",
			"",
		},
		{
			"sqlite3:///path/to/test.db?test=1",
			"file:/path/to/test.db",
			"",
		},
		{
			"sqlite3:",
			"",
			"sqlite3: is not a valid database URI",
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			assert := require.New(t)
			uri, err := url.Parse(test.dsn)
			assert.NoError(err)

			res, err := getSqliteDsn(uri)
			if test.error != "" {
				assert.Equal(test.error, err.Error())
			} else {
				assert.NoError(err)
				assert.Equal(test.expected, res.String())
			}
		})
	}
}

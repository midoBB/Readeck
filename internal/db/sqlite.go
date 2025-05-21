// SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package db

import (
	"fmt"
	"net/url"
	"path/filepath"
)

func getSqliteDsn(dsn *url.URL) (*url.URL, error) {
	if dsn.Opaque == ":memory:" {
		return &url.URL{
			Scheme: "memory",
			Opaque: "/memory.db",
		}, nil
	}

	var err error
	uri := &url.URL{Scheme: "file"}

	// Support initial dsn in several forms
	switch {
	case dsn.Opaque != "":
		// could be sqlite3:some/path
		uri.Path, err = filepath.Abs(dsn.Opaque)
	case dsn.Path != "":
		// or sqlite3:///some/path
		uri.Path, err = filepath.Abs(dsn.Path)
	default:
		err = fmt.Errorf("%s is not a valid database URI", dsn)
	}
	if err != nil {
		return nil, err
	}

	// Convert it to file:<path> (without // path prefix)
	uri.Opaque = uri.Path
	uri.Path = ""

	return uri, nil
}

// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package importer

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"slices"

	"codeberg.org/readeck/readeck/pkg/forms"
)

type textAdapter struct {
	idx  int
	URLs []string `json:"url_list"`
}

func (adapter *textAdapter) Form() forms.Binder {
	return newMultipartForm()
}

func (adapter *textAdapter) Params(form forms.Binder) ([]byte, error) {
	f := form.(*multipartForm).dataReader()
	if f == nil {
		return nil, errors.New("unable to load content")
	}
	defer f.Close() //nolint:errcheck

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		adapter.URLs = append(adapter.URLs, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	slices.Reverse(adapter.URLs)
	return json.Marshal(adapter)
}

func (adapter *textAdapter) LoadData(data []byte) error {
	return json.Unmarshal(data, adapter)
}

func (adapter *textAdapter) Next() (bookmarkImporter, error) {
	if adapter.idx+1 > len(adapter.URLs) {
		return nil, io.EOF
	}

	adapter.idx++
	return newURLBookmark(adapter.URLs[adapter.idx-1])
}

// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package importer

import (
	"encoding/json"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	log "github.com/sirupsen/logrus"

	"codeberg.org/readeck/readeck/internal/auth/users"
	"codeberg.org/readeck/readeck/internal/bus"
	"codeberg.org/readeck/readeck/pkg/superbus"
)

var ImportBookmarksTask superbus.Task // ImportBookmarksTask is the bookmark import task.

// ImportParams contains the ImportBookmarksTask parameters.
type ImportParams struct {
	Source          string `json:"source"`
	Data            []byte `json:"data"`
	UserID          int    `json:"user_id"`
	RequestID       string `json:"request_id"`
	AllowDuplicates bool   `json:"allow_duplicates"`
}

func init() {
	bus.OnReady(func() {
		ImportBookmarksTask = bus.Tasks().NewTask(
			"bookmarks.import",
			superbus.WithUnmarshall(func(data []byte) interface{} {
				var res ImportParams
				err := json.Unmarshal(data, &res)
				if err != nil {
					panic(err)
				}
				return res
			}),
			superbus.WithTaskHandler(importBookmarksHandler),
		)
	})
}

func importBookmarksHandler(data interface{}) {
	params := data.(ImportParams)

	adapter := LoadAdapter(params.Source)
	if adapter == nil {
		panic(fmt.Errorf(`adapter "%s" not found`, params.Source))
	}

	worker, ok := adapter.(importWorker)
	if !ok {
		panic(fmt.Errorf(`loader "%s" does not implement worker`, params.Source))
	}

	var err error
	if err = worker.LoadData(params.Data); err != nil {
		panic(err)
	}

	imp := importer{
		worker:          worker,
		requestID:       params.RequestID,
		allowDuplicates: params.AllowDuplicates,
	}

	if imp.user, err = users.Users.GetOne(goqu.C("id").Eq(params.UserID)); err != nil {
		panic(err)
	}

	imp.log = log.WithFields(log.Fields{"user": imp.user.ID, "@id": imp.requestID})

	if err = imp.Import(); err != nil {
		panic(err)
	}
}

// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package profile

import (
	"encoding/json"
	"log/slog"

	"github.com/doug-martin/goqu/v9"

	"codeberg.org/readeck/readeck/internal/auth/credentials"
	"codeberg.org/readeck/readeck/internal/auth/tokens"
	"codeberg.org/readeck/readeck/internal/bus"
	"codeberg.org/readeck/readeck/pkg/superbus"
)

var (
	deleteCredentialTask superbus.Task
	deleteTokenTask      superbus.Task
)

func init() {
	bus.OnReady(func() {
		deleteCredentialTask = bus.Tasks().NewTask(
			"credential.delete",
			superbus.WithTaskDelay(20),
			superbus.WithUnmarshall(func(data []byte) interface{} {
				var res int
				err := json.Unmarshal(data, &res)
				if err != nil {
					panic(err)
				}
				return res
			}),
			superbus.WithTaskHandler(deleteCredentialHandler),
		)
		deleteTokenTask = bus.Tasks().NewTask(
			"token.delete",
			superbus.WithTaskDelay(20),
			superbus.WithUnmarshall(func(data []byte) interface{} {
				var res int
				err := json.Unmarshal(data, &res)
				if err != nil {
					panic(err)
				}
				return res
			}),
			superbus.WithTaskHandler(deleteTokenHandler),
		)
	})
}

func deleteCredentialHandler(data interface{}) {
	id := data.(int)
	logger := slog.With(slog.Int("id", id))

	c, err := credentials.Credentials.GetOne(goqu.C("id").Eq(id))
	if err != nil {
		logger.Error("credential retrieve", slog.Any("err", err))
		return
	}

	if err := c.Delete(); err != nil {
		logger.Error("token removal", slog.Any("err", err))
		return
	}

	logger.Info("credential removed")
}

func deleteTokenHandler(data interface{}) {
	id := data.(int)
	logger := slog.With(slog.Int("id", id))

	t, err := tokens.Tokens.GetOne(goqu.C("id").Eq(id))
	if err != nil {
		logger.Error("token retrieve", slog.Any("err", err))
		return
	}

	if err := t.Delete(); err != nil {
		logger.Error("token removal", slog.Any("err", err))
		return
	}

	logger.Info("token removed")
}

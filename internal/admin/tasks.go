package admin

import (
	"encoding/json"

	"github.com/doug-martin/goqu/v9"
	log "github.com/sirupsen/logrus"

	"github.com/readeck/readeck/internal/auth/users"
	"github.com/readeck/readeck/internal/bus"
	"github.com/readeck/readeck/pkg/superbus"
)

var deleteUserTask superbus.Task

func init() {
	bus.OnReady(func() {
		deleteUserTask = bus.Tasks().NewTask(
			"user.delete",
			superbus.WithTaskDelay(20),
			superbus.WithUnmarshall(func(data []byte) interface{} {
				var res int
				err := json.Unmarshal(data, &res)
				if err != nil {
					panic(err)
				}
				return res
			}),
			superbus.WithTaskHandler(deleteUserHandler),
		)
	})
}

func deleteUserHandler(data interface{}) {
	id := data.(int)
	logger := log.WithField("id", id)

	u, err := users.Users.GetOne(goqu.C("id").Eq(id))
	if err != nil {
		logger.WithError(err).Error("user retrieve")
		return
	}

	if err := deleteUser(u); err != nil {
		logger.WithError(err).Error("user removal")
		return
	}

	logger.Info("user removed")
}

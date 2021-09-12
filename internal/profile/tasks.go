package profile

import (
	"encoding/json"

	"github.com/doug-martin/goqu/v9"
	log "github.com/sirupsen/logrus"

	"github.com/readeck/readeck/internal/auth/tokens"
	"github.com/readeck/readeck/internal/bus"
	"github.com/readeck/readeck/pkg/superbus"
)

var deleteTokenTask superbus.Task

func init() {
	bus.OnReady(func() {
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

func deleteTokenHandler(data interface{}) {
	id := data.(int)
	logger := log.WithField("id", id)

	t, err := tokens.Tokens.GetOne(goqu.C("id").Eq(id))
	if err != nil {
		logger.WithError(err).Error("token retrieve")
		return
	}

	if err := t.Delete(); err != nil {
		logger.WithError(err).Error("token removal")
		return
	}

	logger.Info("token removed")
}

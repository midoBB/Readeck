// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/cristalhq/acmd"
	log "github.com/sirupsen/logrus"

	"codeberg.org/readeck/readeck/configs"
	"codeberg.org/readeck/readeck/internal/bus"
	"codeberg.org/readeck/readeck/internal/metrics"
)

func init() {
	commands = append(commands, acmd.Command{
		Name:        "worker",
		Description: "Start Readeck jobs worker",
		ExecFunc:    runWorker,
	})
}

func runWorker(_ context.Context, args []string) error {
	var flags appFlags
	if err := flags.Flags().Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	// Init application
	if err := appPreRun(&flags); err != nil {
		return err
	}
	defer appPostRun()

	// Start the metrics HTTP server
	startMetrics := configs.Config.Worker.Metrics.Port > 0
	if startMetrics {
		log.WithField("host", configs.Config.Worker.Metrics.Host).
			WithField("port", configs.Config.Worker.Metrics.Port).
			Info("metrics endpoint")
		go func() {
			if err := metrics.ListenAndServe(
				configs.Config.Worker.Metrics.Host,
				configs.Config.Worker.Metrics.Port,
			); err != nil {
				log.WithError(err).Error("cannot start the metrics server")
				os.Exit(1)
			}
		}()
	}

	// Start jobs worker
	if err := bus.Load(); err != nil {
		return err
	}

	if bus.Protocol() == "memory" {
		return errors.New("cannot start an in memory worker")
	}

	stop := make(chan os.Signal, 2)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	bus.Tasks().Start()
	log.Info("workers started")

	// Shutdown
	<-stop
	log.Info("shutting down...")

	log.Info("stopping workers...")
	bus.Tasks().Stop()
	log.Info("workers stopped")
	return nil
}

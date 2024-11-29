// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/cristalhq/acmd"

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
	startMetrics := configs.Config.Metrics.Port > 0
	if startMetrics {
		slog.Info("metrics endpoint",
			slog.String("host", configs.Config.Metrics.Host),
			slog.Int("port", configs.Config.Metrics.Port),
		)
		go func() {
			if err := metrics.ListenAndServe(
				configs.Config.Metrics.Host,
				configs.Config.Metrics.Port,
			); err != nil {
				fatal("cannot start the metrics server", err)
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
	slog.Info("workers started")

	// Shutdown
	<-stop
	slog.Info("shutting down...")

	slog.Info("stopping workers...")
	bus.Tasks().Stop()
	slog.Info("workers stopped")
	return nil
}

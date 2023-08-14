// SPDX-FileCopyrightText: © 2021 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"errors"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/readeck/readeck/internal/bus"
)

func init() {
	rootCmd.AddCommand(workerCmd)
}

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "start jobs worker",
	RunE:  runWorker,
}

func runWorker(_ *cobra.Command, _ []string) error {
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
	defer cleanup()

	log.Info("stopping workers...")
	bus.Tasks().Stop()
	log.Info("workers stopped")
	return nil
}

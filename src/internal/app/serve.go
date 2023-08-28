// SPDX-FileCopyrightText: © 2020 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"codeberg.org/readeck/readeck/configs"
	"codeberg.org/readeck/readeck/docs"
	"codeberg.org/readeck/readeck/internal/admin"
	"codeberg.org/readeck/readeck/internal/assets"
	"codeberg.org/readeck/readeck/internal/auth/onboarding"
	"codeberg.org/readeck/readeck/internal/auth/signin"
	"codeberg.org/readeck/readeck/internal/bookmarks"
	"codeberg.org/readeck/readeck/internal/bus"
	"codeberg.org/readeck/readeck/internal/cookbook"
	"codeberg.org/readeck/readeck/internal/dashboard"
	"codeberg.org/readeck/readeck/internal/opds"
	"codeberg.org/readeck/readeck/internal/profile"
	"codeberg.org/readeck/readeck/internal/server"
)

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.PersistentFlags().StringVarP(
		&configs.Config.Server.Host, "host", "H",
		configs.Config.Server.Host, "server host")
	serveCmd.PersistentFlags().IntVarP(
		&configs.Config.Server.Port, "port", "p",
		configs.Config.Server.Port, "server host")
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start HTTP server",
	RunE:  runServe,
}

func runServe(_ *cobra.Command, _ []string) error {
	if !configs.Config.Main.DevMode && len(configs.Config.Server.AllowedHosts) == 0 {
		return fmt.Errorf("The server.allowed_hosts setting is not set")
	}

	// Prepare HTTP server
	s := server.New(configs.Config.Server.Prefix)
	if err := InitServer(s); err != nil {
		return err
	}

	srv := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", configs.Config.Server.Host, configs.Config.Server.Port),
		Handler:        s.Router,
		MaxHeaderBytes: 1 << 20,
	}

	if err := bus.Load(); err != nil {
		return err
	}

	if err := onboarding.CLI(); err != nil {
		log.WithError(err).Fatal()
	}

	ready := make(chan bool)
	stop := make(chan os.Signal, 2)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start the embed standalone worker.
	startBus := configs.Config.Worker.StartWorker || bus.Protocol() == "memory"
	if startBus {
		go func() {
			bus.Tasks().Start()
			log.Info("workers started")
		}()
	}

	// Start the HTTP server
	go func() {
		ln, err := net.Listen("tcp", srv.Addr)
		if err != nil {
			log.WithError(err).Error("cannot start the server")
			os.Exit(1)
		}

		ready <- true
		if err = srv.Serve(ln); err != nil {
			if err == http.ErrServerClosed {
				log.Info("stopping server...")
				return
			}
			log.WithError(err).Error("server error")
		}
	}()

	// Server is ready to accept requests
	<-ready
	log.WithField("url", fmt.Sprintf("http://%s:%d%s",
		configs.Config.Server.Host, configs.Config.Server.Port, s.BasePath),
	).Info("server started")

	// Server shutdown
	<-stop
	log.Info("shutting down...")
	defer cleanup()

	// Graceful http shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		panic(err)
	}
	log.Info("server stopped")

	if startBus {
		log.Info("stopping workers...")
		bus.Tasks().Stop()
		log.Info("workers stopped")
	}

	return nil
}

// InitServer setups all the routes.
func InitServer(s *server.Server) error {
	// Init session store
	if err := s.InitSession(); err != nil {
		return err
	}

	// Static asserts
	assets.SetupRoutes(s)

	// Auth routes
	signin.SetupRoutes(s)

	// Dashboard routes
	dashboard.SetupRoutes(s)

	// Bookmark routes
	// - /bookmarks/*
	// - /bm/* (for bookmark media files)
	bookmarks.SetupRoutes(s)

	// OPDS routes
	opds.SetupRoutes(s)

	// User routes
	profile.SetupRoutes(s)

	// Admin routes
	admin.SetupRoutes(s)

	// Help routes
	docs.SetupRoutes(s)

	// Only in dev mode
	if configs.Config.Main.DevMode {
		// Cookbook routes
		cookbook.SetupRoutes(s)
	}

	s.Init()
	return nil
}

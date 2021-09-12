package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/readeck/readeck/configs"
	"github.com/readeck/readeck/internal/admin"
	"github.com/readeck/readeck/internal/assets"
	"github.com/readeck/readeck/internal/auth/signin"
	"github.com/readeck/readeck/internal/bookmarks"
	"github.com/readeck/readeck/internal/bus"
	"github.com/readeck/readeck/internal/cookbook"
	"github.com/readeck/readeck/internal/dashboard"
	"github.com/readeck/readeck/internal/profile"
	"github.com/readeck/readeck/internal/server"
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

	stop := make(chan os.Signal, 2)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start the HTTP server
	go func() {
		log.WithField("url", fmt.Sprintf("http://%s:%d%s",
			configs.Config.Server.Host, configs.Config.Server.Port, s.BasePath),
		).Info("starting server")
		if err := srv.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				log.Info("stopping server...")
				return
			}
			panic(err)
		}
	}()

	// Start the embed standalone worker.
	startBus := configs.Config.Worker.StartWorker || bus.Protocol() == "memory"
	if startBus {
		go func() {
			bus.Tasks().Start()
			log.Info("workers started")
		}()
	}

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

	// User routes
	profile.SetupRoutes(s)

	// Admin routes
	admin.SetupRoutes(s)

	// Only in dev mode
	if configs.Config.Main.DevMode {
		// Cookbook routes
		cookbook.SetupRoutes(s)
	}

	s.Init()
	return nil
}

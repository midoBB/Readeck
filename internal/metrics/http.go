// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// Package metrics provides a prometheus/open-metrics route.
package metrics

import (
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ListenAndServe start an HTTP server only to serve the /metrics and /debug routes.
func ListenAndServe(host string, port int) error {
	r := chi.NewRouter()

	// Metrics
	r.Handle("/metrics", promhttp.Handler())

	// Profiler
	r.Mount("/debug", middleware.Profiler())

	s := &http.Server{
		Addr:              net.JoinHostPort(host, strconv.Itoa(port)),
		Handler:           r,
		ReadHeaderTimeout: time.Second * 1,
	}

	return s.ListenAndServe()
}

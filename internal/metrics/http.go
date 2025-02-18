// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

// Package metrics provides a prometheus/open-metrics route.
package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ListenAndServe start an HTTP server only to serve the /metrics route.
func ListenAndServe(host string, port int) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	s := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", host, port),
		Handler:           mux,
		ReadHeaderTimeout: time.Second * 1,
	}

	return s.ListenAndServe()
}

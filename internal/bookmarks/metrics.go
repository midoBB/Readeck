// SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package bookmarks

import "github.com/prometheus/client_golang/prometheus"

func init() {
	prometheus.MustRegister(metricCreation)
	prometheus.MustRegister(metricTiming)
	prometheus.MustRegister(metricResources)
}

var metricCreation = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name:      "bookmark_creation_total",
	Namespace: "readeck",
	Help:      "Total of created bookmarks",
}, []string{"status"})

var metricTiming = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:      "bookmark_creation_duration_seconds",
	Namespace: "readeck",
	Help:      "Time spent on bookmark retrieval and archiving",
	Buckets:   []float64{1, 2, 5, 10, 20, 60},
}, []string{"status"})

var metricResources = prometheus.NewHistogram(prometheus.HistogramOpts{
	Name:      "bookmark_resources_total",
	Namespace: "readeck",
	Help:      "Total of resources saved with a bookmark",
	Buckets:   []float64{0, 5, 10, 30, 50},
})

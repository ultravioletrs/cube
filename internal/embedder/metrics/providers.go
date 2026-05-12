// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	sourceSyncRuns = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cube_embedder_source_sync_runs_total",
			Help: "Total number of source sync runs by source/provider and result.",
		},
		[]string{"source_type", "provider_type", "result"},
	)
	sourceSyncDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cube_embedder_source_sync_duration_seconds",
			Help:    "Duration of source sync runs in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"source_type", "provider_type", "result"},
	)
	sourceSyncFiles = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cube_embedder_source_sync_files_total",
			Help: "Number of files discovered/queued/updated/unchanged/deleted during source sync.",
		},
		[]string{"source_type", "provider_type", "kind"},
	)
	sourceDownloadRuns = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cube_embedder_source_download_runs_total",
			Help: "Total number of record download attempts by source/provider and result.",
		},
		[]string{"source_type", "provider_type", "result"},
	)
	sourceDownloadDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cube_embedder_source_download_duration_seconds",
			Help:    "Duration of record downloads in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"source_type", "provider_type", "result"},
	)
)

func init() {
	prometheus.MustRegister(
		sourceSyncRuns,
		sourceSyncDuration,
		sourceSyncFiles,
		sourceDownloadRuns,
		sourceDownloadDuration,
	)
}

// ObserveSourceSync captures the outcome and duration of a single sync run.
func ObserveSourceSync(sourceType, providerType string, duration time.Duration, err error) {
	result := "ok"
	if err != nil {
		result = "error"
	}
	labels := []string{normalizeLabel(sourceType), normalizeLabel(providerType), result}
	sourceSyncRuns.WithLabelValues(labels...).Inc()
	sourceSyncDuration.WithLabelValues(labels...).Observe(duration.Seconds())
}

// AddSourceSyncFiles captures file-count outcomes from a sync run.
func AddSourceSyncFiles(
	sourceType, providerType string,
	discovered, queued, updated, unchanged, deleted uint64,
) {
	if discovered == 0 && queued == 0 && updated == 0 && unchanged == 0 && deleted == 0 {
		return
	}
	st := normalizeLabel(sourceType)
	pt := normalizeLabel(providerType)
	addSyncFiles(st, pt, "discovered", discovered)
	addSyncFiles(st, pt, "queued", queued)
	addSyncFiles(st, pt, "updated", updated)
	addSyncFiles(st, pt, "unchanged", unchanged)
	addSyncFiles(st, pt, "deleted", deleted)
}

// ObserveSourceDownload captures the outcome and duration of a single provider download attempt.
func ObserveSourceDownload(sourceType, providerType string, duration time.Duration, err error) {
	result := "ok"
	if err != nil {
		result = "error"
	}
	labels := []string{normalizeLabel(sourceType), normalizeLabel(providerType), result}
	sourceDownloadRuns.WithLabelValues(labels...).Inc()
	sourceDownloadDuration.WithLabelValues(labels...).Observe(duration.Seconds())
}

func addSyncFiles(sourceType, providerType, kind string, n uint64) {
	if n == 0 {
		return
	}
	sourceSyncFiles.WithLabelValues(sourceType, providerType, kind).Add(float64(n))
}

func normalizeLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}

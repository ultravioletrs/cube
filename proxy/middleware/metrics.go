// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package middleware

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/ultraviolet/cube/proxy"
)

var _ proxy.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     proxy.Service
}

func NewMetricsMiddleware(counter metrics.Counter, latency metrics.Histogram, svc proxy.Service) proxy.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (m *metricsMiddleware) Identify(ctx context.Context, token string) (err error) {
	defer func(begin time.Time) {
		m.counter.With("method", "Identify").Add(1)
		m.latency.With("method", "Identify").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.Identify(ctx, token)
}

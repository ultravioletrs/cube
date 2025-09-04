// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package middleware

import (
	"net/http/httputil"
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

// Proxy implements proxy.Service.
func (m *metricsMiddleware) Proxy() *httputil.ReverseProxy {
	// todo : add metrics to the proxy transport
	defer func(begin time.Time) {
		m.counter.With("method", "proxy").Add(1)
		m.latency.With("method", "proxy").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.Proxy()
}

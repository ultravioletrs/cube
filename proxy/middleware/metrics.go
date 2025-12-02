// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/supermq/pkg/authn"
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

func (m *metricsMiddleware) ProxyRequest(ctx context.Context, session *authn.Session, path string) (err error) {
	defer func(begin time.Time) {
		m.counter.With("method", "proxy_request").Add(1)
		m.latency.With("method", "proxy_request").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.ProxyRequest(ctx, session, path)
}

func (m *metricsMiddleware) Secure() string {
	return m.svc.Secure()
}

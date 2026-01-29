// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/go-kit/kit/metrics"
	"github.com/ultravioletrs/cube/proxy"
	"github.com/ultravioletrs/cube/proxy/router"
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

// GetAttestationPolicy implements proxy.Service.
func (m *metricsMiddleware) GetAttestationPolicy(ctx context.Context, session *authn.Session) ([]byte, error) {
	return m.svc.GetAttestationPolicy(ctx, session)
}

// UpdateAttestationPolicy implements proxy.Service.
func (m *metricsMiddleware) UpdateAttestationPolicy(ctx context.Context, session *authn.Session, policy []byte) error {
	return m.svc.UpdateAttestationPolicy(ctx, session, policy)
}

// CreateRoute implements proxy.Service.
func (m *metricsMiddleware) CreateRoute(
	ctx context.Context, session *authn.Session, route *router.RouteRule,
) (*router.RouteRule, error) {
	defer func(begin time.Time) {
		m.counter.With("method", "create_route").Add(1)
		m.latency.With("method", "create_route").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.CreateRoute(ctx, session, route)
}

// UpdateRoute implements proxy.Service.
func (m *metricsMiddleware) UpdateRoute(
	ctx context.Context, session *authn.Session, route *router.RouteRule,
) (*router.RouteRule, error) {
	defer func(begin time.Time) {
		m.counter.With("method", "update_route").Add(1)
		m.latency.With("method", "update_route").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.UpdateRoute(ctx, session, route)
}

// DeleteRoute implements proxy.Service.
func (m *metricsMiddleware) DeleteRoute(ctx context.Context, session *authn.Session, name string) (err error) {
	defer func(begin time.Time) {
		m.counter.With("method", "delete_route").Add(1)
		m.latency.With("method", "delete_route").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return m.svc.DeleteRoute(ctx, session, name)
}

// GetRoute implements proxy.Service.
func (m *metricsMiddleware) GetRoute(
	ctx context.Context, session *authn.Session, name string,
) (*router.RouteRule, error) {
	return m.svc.GetRoute(ctx, session, name)
}

// ListRoutes implements proxy.Service.
func (m *metricsMiddleware) ListRoutes(ctx context.Context, session *authn.Session) ([]router.RouteRule, error) {
	return m.svc.ListRoutes(ctx, session)
}

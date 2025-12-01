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

func (m *metricsMiddleware) ProxyRequest(ctx context.Context, session authn.Session, domainID, path string) (err error) {
	defer func(begin time.Time) {
		m.counter.With("method", "proxy_request").Add(1)
		m.latency.With("method", "proxy_request").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return m.svc.ProxyRequest(ctx, session, domainID, path)
}

func (m *metricsMiddleware) ListAuditLogs(ctx context.Context, session authn.Session, domainID string, query proxy.AuditLogQuery) (logs map[string]interface{}, err error) {
	defer func(begin time.Time) {
		m.counter.With("method", "list_audit_logs").Add(1)
		m.latency.With("method", "list_audit_logs").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return m.svc.ListAuditLogs(ctx, session, domainID, query)
}

func (m *metricsMiddleware) ExportAuditLogs(ctx context.Context, session authn.Session, domainID string, query proxy.AuditLogQuery) (content []byte, contentType string, err error) {
	defer func(begin time.Time) {
		m.counter.With("method", "export_audit_logs").Add(1)
		m.latency.With("method", "export_audit_logs").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return m.svc.ExportAuditLogs(ctx, session, domainID, query)
}

func (m *metricsMiddleware) Secure() string {
	return m.svc.Secure()
}

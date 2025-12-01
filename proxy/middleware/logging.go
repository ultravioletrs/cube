// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/ultraviolet/cube/proxy"
)

var _ proxy.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    proxy.Service
}

func NewLoggingMiddleware(logger *slog.Logger, svc proxy.Service) proxy.Service {
	return &loggingMiddleware{
		logger: logger,
		svc:    svc,
	}
}

func (l *loggingMiddleware) ProxyRequest(ctx context.Context, session authn.Session, domainID, path string) (err error) {
	defer func(begin time.Time) {
		l.logger.Info("ProxyRequest", "domain_id", domainID, "path", path, "took", time.Since(begin), "error", err)
	}(time.Now())
	return l.svc.ProxyRequest(ctx, session, domainID, path)
}

func (l *loggingMiddleware) ListAuditLogs(ctx context.Context, session authn.Session, domainID string, query proxy.AuditLogQuery) (logs map[string]interface{}, err error) {
	defer func(begin time.Time) {
		l.logger.Info("ListAuditLogs", "domain_id", domainID, "took", time.Since(begin), "error", err)
	}(time.Now())
	return l.svc.ListAuditLogs(ctx, session, domainID, query)
}

func (l *loggingMiddleware) ExportAuditLogs(ctx context.Context, session authn.Session, domainID string, query proxy.AuditLogQuery) (content []byte, contentType string, err error) {
	defer func(begin time.Time) {
		l.logger.Info("ExportAuditLogs", "domain_id", domainID, "took", time.Since(begin), "error", err)
	}(time.Now())
	return l.svc.ExportAuditLogs(ctx, session, domainID, query)
}

func (l *loggingMiddleware) Secure() string {
	return l.svc.Secure()
}

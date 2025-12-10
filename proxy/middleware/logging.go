// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/ultravioletrs/cube/proxy"
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

func (l *loggingMiddleware) ProxyRequest(ctx context.Context, session *authn.Session, path string) (err error) {
	defer func(begin time.Time) {
		l.logger.Info("ProxyRequest", "path", path, "took", time.Since(begin), "error", err)
	}(time.Now())

	return l.svc.ProxyRequest(ctx, session, path)
}

func (l *loggingMiddleware) Secure() string {
	return l.svc.Secure()
}

// GetAttestationPolicy implements proxy.Service.
// GetAttestationPolicy implements proxy.Service.
func (l *loggingMiddleware) GetAttestationPolicy(ctx context.Context, session *authn.Session) ([]byte, error) {
	return l.svc.GetAttestationPolicy(ctx, session)
}

// UpdateAttestationPolicy implements proxy.Service.
func (l *loggingMiddleware) UpdateAttestationPolicy(ctx context.Context, session *authn.Session, policy []byte) error {
	return l.svc.UpdateAttestationPolicy(ctx, session, policy)
}

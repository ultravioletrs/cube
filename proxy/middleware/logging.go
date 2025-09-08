// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package middleware

import (
	"log/slog"
	"net/http/httputil"

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

// Proxy implements proxy.Service.
func (l *loggingMiddleware) Proxy() *httputil.ReverseProxy {
	// todo: add logging to the proxy transport
	l.logger.Info("Proxy initialized", "service", "loggingMiddleware")

	return l.svc.Proxy()
}

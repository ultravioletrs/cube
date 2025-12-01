// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/absmach/supermq"
	api "github.com/absmach/supermq/api/http"
	mgauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/authz"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ultraviolet/cube/agent/audit"
	"github.com/ultraviolet/cube/proxy"
)

const ContentType = "application/json"

func MakeHandler(
	svc proxy.Service, instanceID string, auditSvc audit.Service, authn mgauthn.AuthNMiddleware, authz authz.Authorization, idp supermq.IDProvider, opensearchURL string,
) http.Handler {
	mux := chi.NewRouter()

	// Initialize audit handler
	auditHandler := NewAuditHandler(opensearchURL)

	mux.Get("/health", supermq.Health("cube-proxy", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	mux.Route("/{domainID}", func(r chi.Router) {
		r.Use(authn.Middleware(), api.RequestIDMiddleware(idp))
		r.Use(AuthorizationMiddleware(authz))
		r.Use(auditSvc.Middleware)

		// Audit log routes
		r.Get("/audit/logs", auditHandler.FetchAuditLogs)
		r.Post("/audit/export", auditHandler.ExportAuditLogs)

		// Proxy all other requests to the agent
		r.Handle("/*", svc.Proxy())
	})

	return mux
}

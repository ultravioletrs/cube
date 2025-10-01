// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/absmach/supermq"
	api "github.com/absmach/supermq/api/http"
	mgauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ultraviolet/cube/agent"
	"github.com/ultraviolet/cube/agent/audit"
)

const ContentType = "application/json"

func MakeHandler(
	svc agent.Service, instanceID string, auditSvc audit.Service, authn mgauthn.AuthNMiddleware, idp supermq.IDProvider,
) http.Handler {
	mux := chi.NewRouter()

	mux.Get("/health", supermq.Health("cube-agent", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	mux.Route("/", func(r chi.Router) {
		r.Use(authn.Middleware(), api.RequestIDMiddleware(idp))

		r.Use(auditSvc.Middleware)

		r.Handle("/*", svc.Proxy())
	})

	return mux
}

// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/absmach/supermq"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ultraviolet/cube/proxy"
)

const ContentType = "application/json"

func MakeHandler(svc proxy.Service, instanceID string) http.Handler {
	mux := chi.NewRouter()

	mux.Handle("/", svc.Proxy())

	mux.Get("/health", supermq.Health("cube-proxy", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

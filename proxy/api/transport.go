// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/absmach/supermq"
	api "github.com/absmach/supermq/api/http"
	mgauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/go-chi/chi/v5"
	kitendpoint "github.com/go-kit/kit/endpoint"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ultraviolet/cube/agent/audit"
	"github.com/ultraviolet/cube/proxy"
	"github.com/ultraviolet/cube/proxy/endpoint"
)

const ContentType = "application/json"

func MakeHandler(
	svc proxy.Service,
	instanceID string,
	auditSvc audit.Service,
	authn mgauthn.AuthNMiddleware,
	idp supermq.IDProvider,
	proxyTransport http.RoundTripper,
	proxyURL string,
	opensearchURL string,
) http.Handler {
	endpoints := endpoint.MakeEndpoints(svc)

	mux := chi.NewRouter()

	mux.Get("/health", supermq.Health("cube-proxy", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	mux.Route("/{domainID}", func(r chi.Router) {
		r.Use(authn.Middleware(), api.RequestIDMiddleware(idp))
		r.Use(auditSvc.Middleware)

		// Proxy audit log requests to OpenSearch
		r.Handle("/audit/*", makeAuditProxyHandler(endpoints.ProxyRequest, http.DefaultTransport, opensearchURL))

		// Proxy all other requests to the agent
		r.Handle("/*", makeProxyHandler(endpoints.ProxyRequest, proxyTransport, proxyURL))
	})

	return mux
}

func makeProxyHandler(
	proxyEndpoint kitendpoint.Endpoint, transport http.RoundTripper, targetURL string,
) http.HandlerFunc {
	target, _ := url.Parse(targetURL)
	prxy := httputil.NewSingleHostReverseProxy(target)
	prxy.Transport = transport

	originalDirector := prxy.Director
	prxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Strip domainID prefix
		if domainID := chi.URLParam(req, "domainID"); domainID != "" {
			prefix := "/" + domainID

			req.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
			if req.URL.RawPath != "" {
				req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, prefix)
			}
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, ok := ctx.Value(mgauthn.SessionKey).(mgauthn.Session)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)

			return
		}

		domainID := chi.URLParam(r, "domainID")

		// Check authorization via endpoint
		req := endpoint.ProxyRequestRequest{
			Session:  session,
			DomainID: domainID,
			Path:     r.URL.Path,
		}

		if _, err := proxyEndpoint(ctx, req); err != nil {
			encodeError(ctx, err, w)

			return
		}

		// Proceed to proxy
		prxy.ServeHTTP(w, r)
	}
}

func makeAuditProxyHandler(
	proxyEndpoint kitendpoint.Endpoint, transport http.RoundTripper, opensearchURL string,
) http.HandlerFunc {
	target, _ := url.Parse(opensearchURL)
	prxy := httputil.NewSingleHostReverseProxy(target)
	prxy.Transport = transport

	originalDirector := prxy.Director
	prxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Strip domainID and /audit prefix
		if domainID := chi.URLParam(req, "domainID"); domainID != "" {
			prefix := "/" + domainID + "/audit"
			req.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
			if req.URL.RawPath != "" {
				req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, prefix)
			}
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, ok := ctx.Value(mgauthn.SessionKey).(mgauthn.Session)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)

			return
		}

		domainID := chi.URLParam(r, "domainID")

		// Check authorization via endpoint
		req := endpoint.ProxyRequestRequest{
			Session:  session,
			DomainID: domainID,
			Path:     r.URL.Path,
		}

		if _, err := proxyEndpoint(ctx, req); err != nil {
			encodeError(ctx, err, w)

			return
		}

		// Proceed to proxy
		prxy.ServeHTTP(w, r)
	}
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", ContentType)

	if errors.Is(err, errUnauthorized) {
		w.WriteHeader(http.StatusUnauthorized)
	} else {
		w.WriteHeader(http.StatusForbidden) // Default to forbidden for auth errors
	}

	if err := json.NewEncoder(w).Encode(map[string]any{
		"error": err.Error(),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var errUnauthorized = errors.New("unauthorized")

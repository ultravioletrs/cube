// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/absmach/supermq"
	api "github.com/absmach/supermq/api/http"
	mgauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/go-chi/chi/v5"
	kitendpoint "github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ultravioletrs/cube/agent/audit"
	"github.com/ultravioletrs/cube/proxy"
	"github.com/ultravioletrs/cube/proxy/endpoint"
	"github.com/ultravioletrs/cube/proxy/router"
)

const ContentType = "application/json"

// GuardrailsConfig holds configuration for the guardrails sidecar.
type GuardrailsConfig struct {
	Enabled  bool
	URL      string
	AgentURL string
}

func MakeHandler(
	svc proxy.Service,
	instanceID string,
	auditSvc audit.Service,
	authn mgauthn.AuthNMiddleware,
	idp supermq.IDProvider,
	proxyTransport http.RoundTripper,
	rter *router.Router,
	guardrailsCfg GuardrailsConfig,
) http.Handler {
	endpoints := endpoint.MakeEndpoints(svc)

	mux := chi.NewRouter()

	mux.Get("/health", supermq.Health("cube-proxy", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	// Route management endpoints (public, requires authentication)
	mux.Post("/api/routes", authn.Middleware()(kithttp.NewServer(
		endpoints.CreateRoute,
		decodeCreateRouteRequest,
		encodeCreateRouteResponse,
	)).ServeHTTP)

	mux.Get("/api/routes", authn.Middleware()(kithttp.NewServer(
		endpoints.ListRoutes,
		decodeListRoutesRequest,
		encodeListRoutesResponse,
	)).ServeHTTP)

	mux.Get("/api/routes/{name}", authn.Middleware()(kithttp.NewServer(
		endpoints.GetRoute,
		decodeGetRouteRequest,
		encodeGetRouteResponse,
	)).ServeHTTP)

	mux.Put("/api/routes/{name}", authn.Middleware()(kithttp.NewServer(
		endpoints.UpdateRoute,
		decodeUpdateRouteRequest,
		encodeUpdateRouteResponse,
	)).ServeHTTP)

	mux.Delete("/api/routes/{name}", authn.Middleware()(kithttp.NewServer(
		endpoints.DeleteRoute,
		decodeDeleteRouteRequest,
		encodeDeleteRouteResponse,
	)).ServeHTTP)

	mux.Route("/{domainID}", func(r chi.Router) {
		r.Use(authn.Middleware(), api.RequestIDMiddleware(idp))
		r.Use(auditSvc.Middleware)

		r.Get("/attestation/policy", kithttp.NewServer(
			endpoints.GetAttestationPolicy,
			decodeGetAttestationPolicyRequest,
			encodeGetAttestationPolicyResponse,
		).ServeHTTP)

		// Chat completions with guardrails orchestration
		if guardrailsCfg.Enabled {
			r.Post("/api/chat", guardrailsHandler(
				endpoints.ProxyRequest,
				proxyTransport,
				guardrailsCfg,
			))
		}

		// Proxy all other requests using the router
		r.Handle("/*", makeProxyHandler(endpoints.ProxyRequest, proxyTransport, rter))
	})

	mux.Post("/attestation/policy", kithttp.NewServer(
		endpoints.UpdateAttestationPolicy,
		decodeUpdateAttestationPolicyRequest,
		encodeUpdateAttestationPolicyResponse,
	).ServeHTTP)

	return mux
}

// makeProxyHandler creates a http.HandlerFunc that proxies requests.
func makeProxyHandler(
	proxyEndpoint kitendpoint.Endpoint, transport http.RoundTripper, rter *router.Router,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), proxy.MethodContextKey, r.Method)

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

		// Determine target using router
		targetURL, stripPrefix, err := rter.DetermineTarget(r)
		if err != nil {
			log.Printf("Failed to determine target: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)

			return
		}

		target, err := url.Parse(targetURL)
		if err != nil {
			log.Printf("Invalid target URL %s: %v", targetURL, err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)

			return
		}

		prxy := httputil.NewSingleHostReverseProxy(target)
		prxy.Transport = transport

		prxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
			log.Printf("Proxy error: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		}

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

			// Strip configured prefix
			if stripPrefix != "" {
				req.URL.Path = strings.TrimPrefix(req.URL.Path, stripPrefix)
				if req.URL.RawPath != "" {
					req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, stripPrefix)
				}
			}
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

func guardrailsHandler(
	proxyEndpoint kitendpoint.Endpoint,
	transport http.RoundTripper,
	cfg GuardrailsConfig,
) http.HandlerFunc {
	client := &http.Client{
		Transport: transport,
		Timeout:   120 * time.Second,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, ok := ctx.Value(mgauthn.SessionKey).(mgauthn.Session)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)

			return
		}

		domainID := chi.URLParam(r, "domainID")

		if _, err := proxyEndpoint(ctx, endpoint.ProxyRequestRequest{
			Session:  session,
			DomainID: domainID,
			Path:     r.URL.Path,
		}); err != nil {
			encodeError(ctx, err, w)

			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)

			return
		}

		r.Body.Close()

		// Forward to guardrails
		guardrailsURL := cfg.URL + "/guardrails/messages"
		log.Printf("guardrailsHandler: Forwarding to guardrails - URL: %s", guardrailsURL)

		guardrailsBody, guardrailsStatus, err := forwardRequest(ctx, client, guardrailsURL, bodyBytes)
		if err != nil {
			log.Printf(" guardrailsHandler: Failed to call guardrails: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)

			return
		}

		w.Header().Set("Content-Type", ContentType)
		w.WriteHeader(guardrailsStatus)
		_, _ = w.Write(guardrailsBody)
	}
}

func forwardRequest(
	ctx context.Context,
	client *http.Client,
	railsURL string,
	body []byte,
) (b []byte, status int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, railsURL, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Content-Type", ContentType)

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	return respBody, resp.StatusCode, nil
}

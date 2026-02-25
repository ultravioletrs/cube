// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

// Package api provides HTTP transport layer for the proxy service.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/absmach/supermq"
	api "github.com/absmach/supermq/api/http"
	mgauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/go-chi/chi/v5"
	kitendpoint "github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ultravioletrs/cube/agent/audit"
	"github.com/ultravioletrs/cube/proxy"
	"github.com/ultravioletrs/cube/proxy/endpoint"
	"github.com/ultravioletrs/cube/proxy/router"
)

const ContentType = "application/json"

// GuardrailsConfig holds configuration for the guardrails sidecar service.
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
	domainAuthn mgauthn.AuthNMiddleware,
	idp supermq.IDProvider,
	proxyTransport http.RoundTripper,
	rter *router.Router,
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
		r.Use(domainAuthn.Middleware(), api.RequestIDMiddleware(idp))
		r.Use(auditSvc.Middleware)

		r.Get("/attestation/policy", kithttp.NewServer(
			endpoints.GetAttestationPolicy,
			decodeGetAttestationPolicyRequest,
			encodeGetAttestationPolicyResponse,
		).ServeHTTP)

		// Proxy all other requests using the router
		// When guardrails is enabled, /api/chat is routed to guardrails service via config.json
		r.Handle("/*", makeProxyHandler(endpoints.ProxyRequest, proxyTransport, rter))
	})

	mux.Post("/attestation/policy", authn.Middleware()(kithttp.NewServer(
		endpoints.UpdateAttestationPolicy,
		decodeUpdateAttestationPolicyRequest,
		encodeUpdateAttestationPolicyResponse,
	)).ServeHTTP)

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

		serveReverseProxy(w, r, transport, rter)
	}
}

// copyAttestationHeaders copies attestation and TLS-related headers from the upstream response
// to the response writer for audit logging purposes.
func copyAttestationHeaders(w http.ResponseWriter, resp *http.Response) {
	auditHeaders := []string{
		// TLS details
		"X-TLS-Version",
		"X-TLS-Cipher-Suite",
		"X-TLS-Peer-Cert-Issuer",
		// Attestation details
		"X-Attestation-Type",
		"X-Attestation-OK",
		"X-Attestation-Error",
		"X-Attestation-Nonce",
		"X-Attestation-Report",
		"X-ATLS-Handshake",
		"X-ATLS-Handshake-Ms",
	}

	for _, h := range auditHeaders {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
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

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")

	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}

	return a + b
}

func serveReverseProxy(
	w http.ResponseWriter, r *http.Request, transport http.RoundTripper, rter *router.Router,
) {
	rule, err := rter.DetermineTarget(r)
	if err != nil {
		log.Printf("Failed to determine target: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)

		return
	}

	targetURL := rule.TargetURL
	stripPrefix := rule.StripPrefix

	// Inject metadata for audit logging
	ctx := r.Context()
	ctx = context.WithValue(ctx, audit.ATLSExpectedCtxKey, rule.ATLS)
	r = r.WithContext(ctx)

	if rule.EventType != "" {
		w.Header().Set("X-Event-Type", rule.EventType)
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

	// Add ModifyResponse hook to inject attestation headers for audit logging
	prxy.ModifyResponse = func(resp *http.Response) error {
		copyAttestationHeaders(w, resp)

		return nil
	}

	prxy.Director = func(req *http.Request) {
		domainID := chi.URLParam(req, "domainID")
		prepareProxyRequest(req, target, rule, domainID, stripPrefix)
	}

	prxy.ServeHTTP(w, r)
}

func prepareProxyRequest(req *http.Request, target *url.URL, rule *router.RouteRule, domainID, stripPrefix string) {
	if domainID != "" {
		if err := uuid.Validate(domainID); err == nil {
			prefix := "/" + domainID

			req.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
			if req.URL.RawPath != "" {
				req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, prefix)
			}
		}
	}

	if stripPrefix != "" {
		req.URL.Path = strings.TrimPrefix(req.URL.Path, stripPrefix)
		if req.URL.RawPath != "" {
			req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, stripPrefix)
		}
	}

	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host

	remainingPath := req.URL.Path
	if target.Path != "" {
		if remainingPath == "" || remainingPath == "/" {
			req.URL.Path = target.Path
		} else {
			req.URL.Path = singleJoiningSlash(target.Path, remainingPath)
		}
	}

	originalQuery := req.URL.RawQuery

	req.URL.RawQuery = target.RawQuery
	if originalQuery != "" {
		if req.URL.RawQuery == "" {
			req.URL.RawQuery = originalQuery
		} else {
			req.URL.RawQuery = req.URL.RawQuery + "&" + originalQuery
		}
	}

	if rule.Name == "audit" && domainID != "" {
		injectAuditFilter(req, domainID)
	}

	req.Host = target.Host
}

// injectAuditFilter modifies the request to filter audit logs by event.session.DomainID.keyword.
// It handles both query string (q parameter) and JSON body queries.
func injectAuditFilter(req *http.Request, domainID string) {
	filter := "event.session.DomainID.keyword:" + domainID

	if req.Body != nil && req.ContentLength > 0 {
		if err := injectAuditFilterIntoBody(req, domainID); err != nil {
			slog.Error("Failed to inject audit filter into body", "error", err)
		}
	}

	injectAuditFilterIntoQuery(req, filter)
}

func injectAuditFilterIntoBody(req *http.Request, domainID string) error {
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	req.Body.Close()

	if err := tryInjectFilter(req, bodyBytes, domainID); err == nil {
		return nil
	}

	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return nil
}

func tryInjectFilter(req *http.Request, bodyBytes []byte, domainID string) error {
	var bodyMap map[string]any
	if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
		return err
	}

	termFilter := map[string]any{
		"term": map[string]any{
			"event.session.DomainID.keyword": domainID,
		},
	}

	if query, ok := bodyMap["query"].(map[string]any); ok {
		boolQuery, isBool := query["bool"].(map[string]any)
		if !isBool {
			boolQuery = map[string]any{
				"must": []any{query},
			}
			bodyMap["query"] = map[string]any{
				"bool": boolQuery,
			}
		}

		filterArr, ok := boolQuery["filter"].([]any)
		if !ok {
			filterArr = []any{}
		}

		filterArr = append(filterArr, termFilter)
		boolQuery["filter"] = filterArr
	} else {
		bodyMap["query"] = map[string]any{
			"bool": map[string]any{
				"filter": []any{termFilter},
			},
		}
	}

	newBody, err := json.Marshal(bodyMap)
	if err != nil {
		return err
	}

	req.Body = io.NopCloser(bytes.NewBuffer(newBody))
	req.ContentLength = int64(len(newBody))
	req.Header.Set("Content-Length", strconv.Itoa(len(newBody)))

	return nil
}

func injectAuditFilterIntoQuery(req *http.Request, filter string) {
	q := req.URL.Query()

	existingQ := q.Get("q")
	if existingQ != "" {
		q.Set("q", fmt.Sprintf("%s AND %s", existingQ, filter))
	} else {
		q.Set("q", filter)
	}

	req.URL.RawQuery = q.Encode()
}

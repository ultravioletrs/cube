// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/absmach/supermq"
	api "github.com/absmach/supermq/api/http"
	mgauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/go-chi/chi/v5"
	kitendpoint "github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
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
) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	endpoints := endpoint.MakeEndpoints(svc)

	listAuditLogsHandler := kithttp.NewServer(
		endpoints.ListAuditLogs,
		decodeListAuditLogsRequest,
		encodeListAuditLogsResponse,
		opts...,
	)

	exportAuditLogsHandler := kithttp.NewServer(
		endpoints.ExportAuditLogs,
		decodeExportAuditLogsRequest,
		encodeExportAuditLogsResponse,
		opts...,
	)

	mux := chi.NewRouter()

	mux.Get("/health", supermq.Health("cube-proxy", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	mux.Route("/{domainID}", func(r chi.Router) {
		r.Use(authn.Middleware(), api.RequestIDMiddleware(idp))
		r.Use(auditSvc.Middleware)

		// Audit log routes
		r.Get("/audit/logs", listAuditLogsHandler.ServeHTTP)
		r.Post("/audit/export", exportAuditLogsHandler.ServeHTTP)

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

func decodeListAuditLogsRequest(_ context.Context, r *http.Request) (any, error) {
	session, ok := r.Context().Value(mgauthn.SessionKey).(mgauthn.Session)
	if !ok {
		return nil, errUnauthorized
	}

	domainID := chi.URLParam(r, "domainID")
	query := parseQueryParams(r.URL.Query())

	return endpoint.ListAuditLogsRequest{
		Session:  session,
		DomainID: domainID,
		Query:    query,
	}, nil
}

func encodeListAuditLogsResponse(_ context.Context, w http.ResponseWriter, response any) error {
	resp, ok := response.(endpoint.ListAuditLogsResponse)
	if !ok {
		return errInvalidResponseType
	}

	if resp.Err != nil {
		return resp.Err
	}

	w.Header().Set("Content-Type", ContentType)

	return json.NewEncoder(w).Encode(resp.Logs)
}

func decodeExportAuditLogsRequest(_ context.Context, r *http.Request) (any, error) {
	session, ok := r.Context().Value(mgauthn.SessionKey).(mgauthn.Session)
	if !ok {
		return nil, errUnauthorized
	}

	domainID := chi.URLParam(r, "domainID")
	query := parseQueryParams(r.URL.Query())

	return endpoint.ExportAuditLogsRequest{
		Session:  session,
		DomainID: domainID,
		Query:    query,
	}, nil
}

func encodeExportAuditLogsResponse(_ context.Context, w http.ResponseWriter, response any) error {
	resp, ok := response.(endpoint.ExportAuditLogsResponse)
	if !ok {
		return errInvalidResponseType
	}

	if resp.Err != nil {
		return resp.Err
	}

	w.Header().Set("Content-Type", resp.ContentType)
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=audit-logs-%s.json", time.Now().Format("2006-01-02")))

	if _, err := w.Write(resp.Content); err != nil {
		return err
	}

	return nil
}

func parseQueryParams(params url.Values) proxy.AuditLogQuery {
	query := proxy.AuditLogQuery{
		StartTime: time.Now().Add(-24 * time.Hour),
		EndTime:   time.Now(),
		Limit:     100,
		Offset:    0,
	}

	if startStr := params.Get("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			query.StartTime = t
		}
	}

	if endStr := params.Get("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			query.EndTime = t
		}
	}

	query.UserID = params.Get("user_id")
	query.EventType = params.Get("event_type")

	if limitStr := params.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			query.Limit = limit
		}
	}

	if offsetStr := params.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			query.Offset = offset
		}
	}

	return query
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

var (
	errUnauthorized        = errors.New("unauthorized")
	errInvalidResponseType = errors.New("invalid response type")
)

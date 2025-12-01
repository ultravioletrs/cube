// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/absmach/supermq/pkg/authn"
)

// AuditLogQuery represents query parameters for fetching audit logs
type AuditLogQuery struct {
	StartTime time.Time
	EndTime   time.Time
	UserID    string
	EventType string
	Limit     int
	Offset    int
}

// AuditHandler handles audit log requests
type AuditHandler struct {
	opensearchURL string
	httpClient    *http.Client
}

// NewAuditHandler creates a new audit log handler
func NewAuditHandler(opensearchURL string) *AuditHandler {
	return &AuditHandler{
		opensearchURL: opensearchURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchAuditLogs handles GET requests to fetch audit logs from OpenSearch
func (h *AuditHandler) FetchAuditLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract session for filtering
	session, ok := ctx.Value(authn.SessionKey).(authn.Session)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	query := h.parseQueryParams(r.URL.Query(), session)

	// Build OpenSearch query
	searchQuery := h.buildOpenSearchQuery(query)

	// Make request to OpenSearch
	resp, err := h.queryOpenSearch(ctx, searchQuery)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch audit logs: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Stream the response back to the client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// ExportAuditLogs handles POST requests to export audit logs
func (h *AuditHandler) ExportAuditLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract session for filtering
	session, ok := ctx.Value(authn.SessionKey).(authn.Session)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	query := h.parseQueryParams(r.URL.Query(), session)

	// Build OpenSearch query with larger size for export
	query.Limit = 10000 // Max export size
	searchQuery := h.buildOpenSearchQuery(query)

	// Make request to OpenSearch
	resp, err := h.queryOpenSearch(ctx, searchQuery)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to export audit logs: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Set headers for file download
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=audit-logs-%s.json", time.Now().Format("2006-01-02")))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// parseQueryParams extracts and validates query parameters
func (h *AuditHandler) parseQueryParams(params url.Values, session authn.Session) AuditLogQuery {
	query := AuditLogQuery{
		StartTime: time.Now().Add(-24 * time.Hour), // Default: last 24 hours
		EndTime:   time.Now(),
		Limit:     100,
		Offset:    0,
	}

	// Parse start_time
	if startStr := params.Get("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			query.StartTime = t
		}
	}

	// Parse end_time
	if endStr := params.Get("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			query.EndTime = t
		}
	}

	// Parse user_id (optional filter)
	query.UserID = params.Get("user_id")

	// Parse event_type
	query.EventType = params.Get("event_type")

	// Parse limit
	if limitStr := params.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			query.Limit = limit
		}
	}

	// Parse offset
	if offsetStr := params.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			query.Offset = offset
		}
	}

	return query
}

// buildOpenSearchQuery constructs an OpenSearch query DSL
func (h *AuditHandler) buildOpenSearchQuery(query AuditLogQuery) map[string]interface{} {
	mustClauses := []map[string]interface{}{
		{
			"range": map[string]interface{}{
				"timestamp": map[string]interface{}{
					"gte": query.StartTime.Format(time.RFC3339),
					"lte": query.EndTime.Format(time.RFC3339),
				},
			},
		},
	}

	// Add user_id filter if specified
	if query.UserID != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"session.user_id.keyword": query.UserID,
			},
		})
	}

	// Add event_type filter if specified
	if query.EventType != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"event_type": query.EventType,
			},
		})
	}

	return map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": mustClauses,
			},
		},
		"sort": []map[string]interface{}{
			{
				"timestamp": map[string]string{
					"order": "desc",
				},
			},
		},
		"size": query.Limit,
		"from": query.Offset,
	}
}

// queryOpenSearch executes a search query against OpenSearch
func (h *AuditHandler) queryOpenSearch(ctx context.Context, query map[string]interface{}) (*http.Response, error) {
	// Serialize query to JSON
	queryBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Create request to OpenSearch
	searchURL := fmt.Sprintf("%s/cube-audit-*/_search", h.opensearchURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, searchURL, bytes.NewReader(queryBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query OpenSearch: %w", err)
	}

	return resp, nil
}

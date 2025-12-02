// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/ultravioletrs/cocos/pkg/clients"
	httpclient "github.com/ultravioletrs/cocos/pkg/clients/http"
)

var errOpenSearchBadStatus = errors.New("opensearch returned bad status")

// AuditLogQuery represents query parameters for fetching audit logs.
type AuditLogQuery struct {
	StartTime time.Time
	EndTime   time.Time
	UserID    string
	EventType string
	Limit     int
	Offset    int
}

type Service interface {
	// ProxyRequest checks if the request is allowed.
	ProxyRequest(ctx context.Context, session *authn.Session, path string) error
	// ListAuditLogs fetches audit logs.
	ListAuditLogs(ctx context.Context, session *authn.Session, query *AuditLogQuery) (map[string]any, error)
	// ExportAuditLogs exports audit logs.
	ExportAuditLogs(ctx context.Context, session *authn.Session, query *AuditLogQuery) ([]byte, string, error)
	// Secure returns the secure connection type.
	Secure() string
}

type service struct {
	config        *clients.AttestedClientConfig
	transport     *http.Transport
	secure        string
	opensearchURL string
	httpClient    *http.Client
}

func New(config *clients.AttestedClientConfig, opensearchURL string) (Service, error) {
	client, err := httpclient.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &service{
		config:        config,
		transport:     client.Transport(),
		secure:        client.Secure(),
		opensearchURL: opensearchURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (s *service) ProxyRequest(_ context.Context, _ *authn.Session, _ string) error {
	return nil
}

func (s *service) ListAuditLogs(ctx context.Context, _ *authn.Session, query *AuditLogQuery) (map[string]any, error) {
	searchQuery := s.buildOpenSearchQuery(query)

	resp, err := s.queryOpenSearch(ctx, searchQuery)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

func (s *service) ExportAuditLogs(
	ctx context.Context, _ *authn.Session, query *AuditLogQuery,
) (body []byte, contentType string, err error) {
	query.Limit = 10000 // Max export size
	searchQuery := s.buildOpenSearchQuery(query)

	resp, err := s.queryOpenSearch(ctx, searchQuery)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	contentType = "application/json"

	return body, contentType, nil
}

func (s *service) Secure() string {
	return s.secure
}

func (s *service) buildOpenSearchQuery(query *AuditLogQuery) map[string]any {
	mustClauses := []map[string]any{
		{
			"range": map[string]any{
				"timestamp": map[string]any{
					"gte": query.StartTime.Format(time.RFC3339),
					"lte": query.EndTime.Format(time.RFC3339),
				},
			},
		},
	}

	if query.UserID != "" {
		mustClauses = append(mustClauses, map[string]any{
			"term": map[string]any{
				"session.user_id.keyword": query.UserID,
			},
		})
	}

	if query.EventType != "" {
		mustClauses = append(mustClauses, map[string]any{
			"term": map[string]any{
				"event_type": query.EventType,
			},
		})
	}

	return map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": mustClauses,
			},
		},
		"sort": []map[string]any{
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

func (s *service) queryOpenSearch(ctx context.Context, query map[string]any) (*http.Response, error) {
	queryBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	searchURL := s.opensearchURL + "/cube-audit-*/_search"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, searchURL, bytes.NewReader(queryBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query OpenSearch: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()

		return nil, fmt.Errorf("%w: %d", errOpenSearchBadStatus, resp.StatusCode)
	}

	return resp, nil
}

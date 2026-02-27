// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ultravioletrs/cube/proxy/api"
)

func TestInjectAuditFilterAddsQuotedQueryAndBodyFilter(t *testing.T) {
	t.Parallel()

	domainID := "878da198-6854-4194-8a3d-aab32e532b6c"
	body := `{"query":{"match_all":{}}}`
	req := httptest.NewRequest(http.MethodPost, "http://example.com/_search?q=status:ok", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	api.InjectAuditFilter(req, domainID)

	gotQ := req.URL.Query().Get("q")

	wantQ := `status:ok AND event.session.DomainID.keyword:"878da198-6854-4194-8a3d-aab32e532b6c"`
	if gotQ != wantQ {
		t.Fatalf("unexpected q filter: got %q, want %q", gotQ, wantQ)
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("failed reading body: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		t.Fatalf("failed unmarshalling body: %v", err)
	}

	query, ok := payload["query"].(map[string]any)
	if !ok {
		t.Fatalf("query missing or invalid type: %#v", payload["query"])
	}

	boolQuery, ok := query["bool"].(map[string]any)
	if !ok {
		t.Fatalf("bool query missing: %#v", query)
	}

	filters, ok := boolQuery["filter"].([]any)
	if !ok || len(filters) != 1 {
		t.Fatalf("expected exactly one filter, got %#v", boolQuery["filter"])
	}

	term, ok := filters[0].(map[string]any)["term"].(map[string]any)
	if !ok {
		t.Fatalf("term filter missing or invalid: %#v", filters[0])
	}

	if got := term["event.session.DomainID.keyword"]; got != domainID {
		t.Fatalf("unexpected term filter value: got %#v, want %q", got, domainID)
	}
}

func TestInjectAuditFilterWithInvalidJSONKeepsBodyAndAddsQueryFilter(t *testing.T) {
	t.Parallel()

	domainID := "878da198-6854-4194-8a3d-aab32e532b6c"
	originalBody := "not-json"
	req := httptest.NewRequest(http.MethodPost, "http://example.com/_search", strings.NewReader(originalBody))

	api.InjectAuditFilter(req, domainID)

	gotQ := req.URL.Query().Get("q")

	wantQ := `event.session.DomainID.keyword:"878da198-6854-4194-8a3d-aab32e532b6c"`
	if gotQ != wantQ {
		t.Fatalf("unexpected q filter: got %q, want %q", gotQ, wantQ)
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("failed reading body: %v", err)
	}

	if string(bodyBytes) != originalBody {
		t.Fatalf("body changed unexpectedly: got %q, want %q", string(bodyBytes), originalBody)
	}
}

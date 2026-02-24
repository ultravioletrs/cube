// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package audit_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ultravioletrs/cube/agent/audit"
)

type mockRoundTripper struct {
	lastReq *http.Request
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.lastReq = req

	return &http.Response{
		Header: make(http.Header),
		TLS:    &tls.ConnectionState{}, // Add TLS state to allow ATLSHandshake to be set
	}, nil
}

func TestInstrumentedTransportATLSExpected(t *testing.T) {
	t.Parallel()

	mock := &mockRoundTripper{}
	it := audit.NewInstrumentedTransport(mock, "aTLS")

	tests := []struct {
		name     string
		ctxValue any
		expected bool
	}{
		{
			name:     "No context value",
			ctxValue: nil,
			expected: true, // Default from "aTLS" type
		},
		{
			name:     "Override to false",
			ctxValue: false,
			expected: false,
		},
		{
			name:     "Override to true",
			ctxValue: true,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", http.NoBody)
			if tt.ctxValue != nil {
				req = req.WithContext(context.WithValue(req.Context(), audit.ATLSExpectedCtxKey, tt.ctxValue))
			}

			res, _ := it.RoundTrip(req)
			if res != nil && res.Body != nil {
				res.Body.Close()
			}

			result := it.GetLastResult()
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, result.ATLSHandshake, tt.name)
		})
	}
}

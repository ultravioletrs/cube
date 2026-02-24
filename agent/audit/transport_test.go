package audit

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
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
	mock := &mockRoundTripper{}
	it := NewInstrumentedTransport(mock, "aTLS")

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
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			if tt.ctxValue != nil {
				req = req.WithContext(context.WithValue(req.Context(), ATLSExpectedCtxKey, tt.ctxValue))
			}

			_, _ = it.RoundTrip(req)

			it.mu.RLock()
			assert.Equal(t, tt.expected, it.lastResult.ATLSHandshake, tt.name)
			it.mu.RUnlock()
		})
	}
}

package audit

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractEventType(t *testing.T) {
	am := &auditMiddleware{}

	tests := []struct {
		name     string
		headers  http.Header
		expected string
	}{
		{
			name:     "Default event type",
			headers:  http.Header{},
			expected: "llm_request",
		},
		{
			name: "Custom event type",
			headers: http.Header{
				HeaderXEventType: []string{"audit_event"},
			},
			expected: "audit_event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := am.ExtractEventType(tt.headers)
			assert.Equal(t, tt.expected, result)
		})
	}
}

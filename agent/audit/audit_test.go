// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package audit_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ultravioletrs/cube/agent/audit"
)

func TestExtractEventType(t *testing.T) {
	t.Parallel()

	am := &audit.Middleware{}

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
				audit.HeaderXEventType: []string{"audit_event"},
			},
			expected: "audit_event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := am.ExtractEventType(tt.headers)
			assert.Equal(t, tt.expected, result)
		})
	}
}

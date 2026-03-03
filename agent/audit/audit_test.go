// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package audit_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestGuardrailsResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   audit.GuardrailsResult
		expected map[string]any
	}{
		{
			name: "PII detection result",
			result: audit.GuardrailsResult{
				Type:        "pii_detected",
				Category:    "output_validation",
				Severity:    "medium",
				Description: "Email address detected and masked",
				Entity:      "EMAIL_ADDRESS",
				Confidence:  0.95,
				Action:      "masked",
				Rail:        "mask sensitive data on output",
			},
			expected: map[string]any{
				"type":        "pii_detected",
				"category":    "output_validation",
				"severity":    "medium",
				"description": "Email address detected and masked",
				"entity":      "EMAIL_ADDRESS",
				"confidence":  0.95,
				"action":      "masked",
				"rail":        "mask sensitive data on output",
			},
		},
		{
			name: "Prompt injection detection",
			result: audit.GuardrailsResult{
				Type:        "prompt_injection",
				Category:    "input_validation",
				Severity:    "high",
				Description: "Detected attempt to ignore previous instructions",
				Action:      "blocked",
			},
			expected: map[string]any{
				"type":        "prompt_injection",
				"category":    "input_validation",
				"severity":    "high",
				"description": "Detected attempt to ignore previous instructions",
				"action":      "blocked",
			},
		},
		{
			name: "Jailbreak attempt",
			result: audit.GuardrailsResult{
				Type:        "jailbreak_attempt",
				Category:    "input_validation",
				Severity:    "high",
				Description: "User attempted to bypass safety guidelines",
				Action:      "blocked",
				Rail:        "check jailbreak",
			},
			expected: map[string]any{
				"type":        "jailbreak_attempt",
				"category":    "input_validation",
				"severity":    "high",
				"description": "User attempted to bypass safety guidelines",
				"action":      "blocked",
				"rail":        "check jailbreak",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test JSON serialization
			data, err := json.Marshal(tt.result)
			require.NoError(t, err)

			var decoded map[string]any

			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			// Verify key fields are present
			assert.Equal(t, tt.expected["type"], decoded["type"])
			assert.Equal(t, tt.expected["category"], decoded["category"])
			assert.Equal(t, tt.expected["severity"], decoded["severity"])
			assert.Equal(t, tt.expected["action"], decoded["action"])
		})
	}
}

func TestEventGuardrailsFields(t *testing.T) {
	t.Parallel()

	event := audit.Event{
		TraceID:              "trace-123",
		RequestID:            "req-456",
		GuardrailsProcessed:  true,
		GuardrailsDecision:   "BLOCK",
		TriggeredInputRails:  []string{"check jailbreak", "check prompt injection"},
		TriggeredOutputRails: []string{"mask sensitive data on output"},
		GuardrailsViolations: []audit.GuardrailsResult{
			{
				Type:        "jailbreak_attempt",
				Category:    "input_validation",
				Severity:    "high",
				Description: "Jailbreak attempt detected",
				Action:      "blocked",
			},
		},
		SensitiveDataMasked: true,
		PromptInjection:     false,
		JailbreakAttempt:    true,
		ToxicContent:        false,
		OffTopicDetected:    false,
		HallucinationRisk:   false,
		GuardrailsLatencyMs: 125.5,
	}

	// Test JSON serialization
	data, err := json.Marshal(event)
	require.NoError(t, err)

	var decoded map[string]any

	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Verify guardrails fields are properly serialized
	assert.Equal(t, true, decoded["guardrails_processed"])
	assert.Equal(t, "BLOCK", decoded["guardrails_decision"])
	assert.InEpsilon(t, 125.5, decoded["guardrails_latency_ms"], 0.001)
	assert.Equal(t, true, decoded["jailbreak_attempt"])
	assert.Equal(t, true, decoded["sensitive_data_masked"])
	assert.Equal(t, false, decoded["prompt_injection"])

	// Check triggered rails
	inputRails, ok := decoded["triggered_input_rails"].([]any)
	require.True(t, ok, "triggered_input_rails should be []any")
	assert.Len(t, inputRails, 2)
	assert.Contains(t, inputRails, "check jailbreak")

	// Check violations
	violations, ok := decoded["guardrails_violations"].([]any)
	require.True(t, ok, "guardrails_violations should be []any")
	assert.Len(t, violations, 1)
}

func TestGuardrailsHeaderConstants(t *testing.T) {
	t.Parallel()

	// Verify header constants are properly defined
	assert.Equal(t, "X-Guardrails-Processed", audit.HeaderGuardrailsProcessed)
	assert.Equal(t, "X-Guardrails-Decision", audit.HeaderGuardrailsDecision)
	assert.Equal(t, "X-Guardrails-Violations", audit.HeaderGuardrailsViolations)
	assert.Equal(t, "X-Input-Rails-Triggered", audit.HeaderInputRailsTriggered)
	assert.Equal(t, "X-Output-Rails-Triggered", audit.HeaderOutputRailsTriggered)
	assert.Equal(t, "X-Sensitive-Data-Masked", audit.HeaderSensitiveDataMasked)
	assert.Equal(t, "X-Prompt-Injection-Detected", audit.HeaderPromptInjection)
	assert.Equal(t, "X-Jailbreak-Attempt-Detected", audit.HeaderJailbreakAttempt)
	assert.Equal(t, "X-Toxic-Content-Detected", audit.HeaderToxicContent)
	assert.Equal(t, "X-Off-Topic-Detected", audit.HeaderOffTopic)
	assert.Equal(t, "X-Hallucination-Risk", audit.HeaderHallucinationRisk)
	assert.Equal(t, "X-Guardrails-Latency-Ms", audit.HeaderGuardrailsLatencyMs)
	assert.Equal(t, "X-Guardrails-Error", audit.HeaderGuardrailsError)
}

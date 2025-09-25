// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// AuditEvent represents a complete audit log entry.
type AuditEvent struct {
	// Core identification
	TraceID   string    `json:"trace_id"`
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"event_type"`

	// Authentication & Authorization
	UserID          string   `json:"user_id,omitempty"`
	ClientID        string   `json:"client_id,omitempty"`
	AuthMethod      string   `json:"auth_method,omitempty"`
	Permissions     []string `json:"permissions,omitempty"`
	AttestationType string   `json:"attestation_type,omitempty"`
	AttestationOK   bool     `json:"attestation_verified"`

	// Request details
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Endpoint  string            `json:"endpoint"`
	UserAgent string            `json:"user_agent"`
	ClientIP  string            `json:"client_ip"`
	Headers   map[string]string `json:"headers,omitempty"`

	// Response details
	StatusCode       int           `json:"status_code"`
	ResponseSize     int64         `json:"response_size_bytes"`
	RequestSize      int64         `json:"request_size_bytes"`
	Duration         time.Duration `json:"duration_ms"`
	UpstreamDuration time.Duration `json:"upstream_duration_ms,omitempty"`

	// LLM specific
	Model        string  `json:"model,omitempty"`
	InputTokens  int     `json:"input_tokens,omitempty"`
	OutputTokens int     `json:"output_tokens,omitempty"`
	Temperature  float64 `json:"temperature,omitempty"`
	MaxTokens    int     `json:"max_tokens,omitempty"`

	// Security & Compliance
	TLSVersion      string   `json:"tls_version,omitempty"`
	CipherSuite     string   `json:"cipher_suite,omitempty"`
	ContentFiltered bool     `json:"content_filtered"`
	PIIDetected     bool     `json:"pii_detected"`
	ComplianceTags  []string `json:"compliance_tags,omitempty"`

	// Error handling
	Error     string `json:"error,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`

	// Additional metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AuditMiddleware provides structured audit logging.
type AuditMiddleware struct {
	logger    *slog.Logger
	config    AuditConfig
	extractor TokenExtractor
}

type AuditConfig struct {
	LogLevel         slog.Level
	EnablePIIMask    bool
	EnableTokens     bool
	SensitiveHeaders []string
	ComplianceMode   bool
}

type TokenExtractor interface {
	ExtractUserInfo(ctx context.Context, token string) (UserInfo, error)
}

type UserInfo struct {
	UserID      string
	ClientID    string
	Permissions []string
}

// responseCapture captures response data for audit logging.
type responseCapture struct {
	http.ResponseWriter

	statusCode int
	size       int64
	body       *bytes.Buffer
}

func (rc *responseCapture) WriteHeader(statusCode int) {
	rc.statusCode = statusCode
	rc.ResponseWriter.WriteHeader(statusCode)
}

func (rc *responseCapture) Write(data []byte) (int, error) {
	rc.size += int64(len(data))
	if rc.body != nil && rc.body.Len() < 1024*10 { // Capture first 10KB
		rc.body.Write(data)
	}

	return rc.ResponseWriter.Write(data)
}

// NewAuditMiddleware creates a new audit middleware instance.
func NewAuditMiddleware(logger *slog.Logger, config AuditConfig, extractor TokenExtractor) *AuditMiddleware {
	return &AuditMiddleware{
		logger:    logger,
		config:    config,
		extractor: extractor,
	}
}

// Middleware returns the HTTP middleware function.
func (am *AuditMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Generate trace and request IDs
		traceID := generateID()
		requestID := generateID()

		// Create context with trace information
		ctx := context.WithValue(r.Context(), "trace_id", traceID)
		ctx = context.WithValue(ctx, "request_id", requestID)
		r = r.WithContext(ctx)

		// Capture request body for audit (if needed)
		var requestBody []byte

		if am.shouldCaptureBody(r) {
			body, err := io.ReadAll(r.Body)
			if err == nil {
				requestBody = body
				r.Body = io.NopCloser(bytes.NewReader(body))
			}
		}

		// Create response capture wrapper
		capture := &responseCapture{
			ResponseWriter: w,
			statusCode:     200,
			body:           bytes.NewBuffer(nil),
		}

		// Extract user information
		userInfo := am.extractUserInfo(r)

		// Process request
		upstreamStart := time.Now()

		next.ServeHTTP(capture, r)

		upstreamDuration := time.Since(upstreamStart)

		// Create and log audit event
		event := am.createAuditEvent(r, capture, userInfo, traceID, requestID, start, upstreamDuration, requestBody)
		am.logAuditEvent(event)
	})
}

func (am *AuditMiddleware) createAuditEvent(
	r *http.Request,
	capture *responseCapture,
	userInfo UserInfo,
	traceID, requestID string,
	start time.Time,
	upstreamDuration time.Duration,
	requestBody []byte,
) AuditEvent {
	event := AuditEvent{
		TraceID:          traceID,
		RequestID:        requestID,
		Timestamp:        start,
		EventType:        "llm_request",
		UserID:           userInfo.UserID,
		ClientID:         userInfo.ClientID,
		Permissions:      userInfo.Permissions,
		Method:           r.Method,
		Path:             r.URL.Path,
		Endpoint:         fmt.Sprintf("%s %s", r.Method, r.URL.Path),
		UserAgent:        r.UserAgent(),
		ClientIP:         am.extractClientIP(r),
		StatusCode:       capture.statusCode,
		ResponseSize:     capture.size,
		RequestSize:      r.ContentLength,
		Duration:         time.Since(start),
		UpstreamDuration: upstreamDuration,
		Headers:          am.sanitizeHeaders(r.Header),
		TLSVersion:       am.getTLSVersion(r),
		ComplianceTags:   []string{"enterprise", "audit"},
	}

	// Extract LLM-specific information from request body
	if len(requestBody) > 0 {
		am.extractLLMMetadata(&event, requestBody)
	}

	// Extract LLM response information
	if capture.body.Len() > 0 {
		am.extractLLMResponse(&event, capture.body.Bytes())
	}

	// Perform content analysis
	event.PIIDetected = am.detectPII(requestBody, capture.body.Bytes())
	event.ContentFiltered = am.checkContentFilter(&event)

	return event
}

func (am *AuditMiddleware) extractLLMMetadata(event *AuditEvent, requestBody []byte) {
	var requestData map[string]interface{}
	if err := json.Unmarshal(requestBody, &requestData); err != nil {
		return
	}

	if model, ok := requestData["model"].(string); ok {
		event.Model = model
	}

	if temp, ok := requestData["temperature"].(float64); ok {
		event.Temperature = temp
	}

	if maxTokens, ok := requestData["max_tokens"].(float64); ok {
		event.MaxTokens = int(maxTokens)
	}

	// Estimate input tokens (rough approximation)
	if messages, ok := requestData["messages"].([]interface{}); ok {
		totalChars := 0

		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				if content, ok := msgMap["content"].(string); ok {
					totalChars += len(content)
				}
			}
		}

		event.InputTokens = totalChars / 4 // Rough token estimate
	}
}

func (am *AuditMiddleware) extractLLMResponse(event *AuditEvent, responseBody []byte) {
	var responseData map[string]interface{}
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		return
	}

	if usage, ok := responseData["usage"].(map[string]interface{}); ok {
		if promptTokens, ok := usage["prompt_tokens"].(float64); ok {
			event.InputTokens = int(promptTokens)
		}

		if completionTokens, ok := usage["completion_tokens"].(float64); ok {
			event.OutputTokens = int(completionTokens)
		}
	}
}

func (am *AuditMiddleware) logAuditEvent(event AuditEvent) {
	// Convert duration to milliseconds for logging
	durationMs := float64(event.Duration.Nanoseconds()) / 1e6
	upstreamMs := float64(event.UpstreamDuration.Nanoseconds()) / 1e6

	am.logger.Info("audit_event",
		slog.String("trace_id", event.TraceID),
		slog.String("request_id", event.RequestID),
		slog.String("event_type", event.EventType),
		slog.String("user_id", event.UserID),
		slog.String("client_id", event.ClientID),
		slog.String("method", event.Method),
		slog.String("endpoint", event.Endpoint),
		slog.String("client_ip", event.ClientIP),
		slog.Int("status_code", event.StatusCode),
		slog.Float64("duration_ms", durationMs),
		slog.Float64("upstream_duration_ms", upstreamMs),
		slog.String("model", event.Model),
		slog.Int("input_tokens", event.InputTokens),
		slog.Int("output_tokens", event.OutputTokens),
		slog.Bool("pii_detected", event.PIIDetected),
		slog.Bool("content_filtered", event.ContentFiltered),
		slog.Any("metadata", event),
	)
}

// Utility functions.
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)

	return hex.EncodeToString(bytes)
}

func (am *AuditMiddleware) shouldCaptureBody(r *http.Request) bool {
	return r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch
}

func (am *AuditMiddleware) extractUserInfo(r *http.Request) UserInfo {
	token := am.extractBearerToken(r)
	if token == "" || am.extractor == nil {
		return UserInfo{}
	}

	userInfo, err := am.extractor.ExtractUserInfo(r.Context(), token)
	if err != nil {
		return UserInfo{}
	}

	return userInfo
}

func (am *AuditMiddleware) extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}

func (am *AuditMiddleware) extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")

		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}

	return ip
}

func (am *AuditMiddleware) sanitizeHeaders(headers http.Header) map[string]string {
	sanitized := make(map[string]string)

	sensitiveHeaders := map[string]bool{
		"authorization": true,
		"cookie":        true,
		"x-api-key":     true,
	}

	for name, values := range headers {
		lowerName := strings.ToLower(name)
		if sensitiveHeaders[lowerName] {
			sanitized[name] = "[REDACTED]"
		} else if len(values) > 0 {
			sanitized[name] = values[0]
		}
	}

	return sanitized
}

func (am *AuditMiddleware) getTLSVersion(r *http.Request) string {
	if r.TLS == nil {
		return ""
	}

	switch r.TLS.Version {
	case 0x0304:
		return "TLS1.3"
	case 0x0303:
		return "TLS1.2"
	case 0x0302:
		return "TLS1.1"
	case 0x0301:
		return "TLS1.0"
	default:
		return "Unknown"
	}
}

func (am *AuditMiddleware) detectPII(requestBody, responseBody []byte) bool {
	// Simple PII detection patterns
	piiPatterns := []string{
		`\b\d{3}-\d{2}-\d{4}\b`,                               // SSN
		`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, // Email
		`\b\d{4}[- ]?\d{4}[- ]?\d{4}[- ]?\d{4}\b`,             // Credit card
	}

	content := string(append(requestBody, responseBody...))

	for _, pattern := range piiPatterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			return true
		}
	}

	return false
}

func (am *AuditMiddleware) checkContentFilter(event *AuditEvent) bool {
	// Implement content filtering logic
	// This could integrate with external content filtering services
	return false
}

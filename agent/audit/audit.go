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
	"strconv"
	"strings"
	"time"

	"github.com/absmach/supermq/pkg/authn"
)

type contextKey string

const (
	RequestIDCtxKey    contextKey = "request_id"
	TraceIDCtxKey      contextKey = "trace_id"
	ATLSExpectedCtxKey contextKey = "atls_expected"

	HeaderXEventType = "X-Event-Type"

	// HeaderGuardrailsProcessed and other guardrails detection header names for audit logging.
	HeaderGuardrailsProcessed  = "X-Guardrails-Processed"
	HeaderGuardrailsDecision   = "X-Guardrails-Decision"
	HeaderGuardrailsViolations = "X-Guardrails-Violations"
	HeaderInputRailsTriggered  = "X-Input-Rails-Triggered"
	HeaderOutputRailsTriggered = "X-Output-Rails-Triggered"
	HeaderSensitiveDataMasked  = "X-Sensitive-Data-Masked"
	HeaderPromptInjection      = "X-Prompt-Injection-Detected"
	HeaderJailbreakAttempt     = "X-Jailbreak-Attempt-Detected"
	HeaderToxicContent         = "X-Toxic-Content-Detected"
	HeaderOffTopic             = "X-Off-Topic-Detected"
	HeaderHallucinationRisk    = "X-Hallucination-Risk"
	HeaderGuardrailsLatencyMs  = "X-Guardrails-Latency-Ms"
	HeaderGuardrailsError      = "X-Guardrails-Error"
	strTrue                    = "true"
)

// GuardrailsResult represents a single guardrails detection/violation result.
type GuardrailsResult struct {
	Type        string  `json:"type"`                  // e.g., "pii", "prompt_injection"
	Category    string  `json:"category,omitempty"`    // e.g., "input_validation", "output_validation"
	Severity    string  `json:"severity,omitempty"`    // e.g., "low", "medium", "high", "critical"
	Description string  `json:"description,omitempty"` // Human-readable description
	Entity      string  `json:"entity,omitempty"`      // e.g., "EMAIL_ADDRESS", "CREDIT_CARD" for PII
	Confidence  float64 `json:"confidence,omitempty"`  // Detection confidence score (0.0-1.0)
	Action      string  `json:"action,omitempty"`      // e.g., "blocked", "masked", "allowed"
	Rail        string  `json:"rail,omitempty"`        // Name of the rail that triggered
}

// Event represents a complete audit log entry.
type Event struct {
	// Core identification
	TraceID   string    `json:"trace_id"`
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"event_type"`

	// Authentication & Authorization
	Session         authn.Session `json:"session,omitzero"`
	AuthMethod      string        `json:"auth_method,omitempty"`
	AttestationType string        `json:"attestation_type,omitempty"`
	AttestationOK   bool          `json:"attestation_ok,omitempty"`

	// Request details
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Endpoint  string            `json:"endpoint"`
	UserAgent string            `json:"user_agent"`
	ClientIP  string            `json:"client_ip"`
	Headers   map[string]string `json:"headers,omitempty"`

	// Response details
	StatusCode       int           `json:"status_code"`
	ResponseSize     int64         `json:"response_size"`
	RequestSize      int64         `json:"request_size"`
	Duration         time.Duration `json:"duration"`
	DurationMs       float64       `json:"duration_ms"`
	UpstreamDuration time.Duration `json:"upstream_duration,omitempty"`
	UpstreamMs       float64       `json:"upstream_ms,omitempty"`

	// LLM specific
	Model        string  `json:"model,omitempty"`
	InputTokens  int     `json:"input_tokens,omitempty"`
	OutputTokens int     `json:"output_tokens,omitempty"`
	Temperature  float64 `json:"temperature,omitempty"`
	MaxTokens    int     `json:"max_tokens,omitempty"`

	// Security & Compliance
	TLSVersion      string   `json:"tls_version,omitempty"`
	CipherSuite     string   `json:"cipher_suite,omitempty"`
	PeerCertIssuer  string   `json:"peer_cert_issuer,omitempty"`
	ContentFiltered bool     `json:"content_filtered"`
	PIIDetected     bool     `json:"pii_detected"`
	ComplianceTags  []string `json:"compliance_tags,omitempty"`

	// Guardrails Detection & Violations
	GuardrailsProcessed  bool               `json:"guardrails_processed"`
	GuardrailsDecision   string             `json:"guardrails_decision,omitempty"`   // ALLOW, BLOCK, MODIFY
	GuardrailsViolations []GuardrailsResult `json:"guardrails_violations,omitempty"` // List of detected violations
	TriggeredInputRails  []string           `json:"triggered_input_rails,omitempty"`
	TriggeredOutputRails []string           `json:"triggered_output_rails,omitempty"`
	SensitiveDataMasked  bool               `json:"sensitive_data_masked"`
	PromptInjection      bool               `json:"prompt_injection"`
	JailbreakAttempt     bool               `json:"jailbreak_attempt"`
	ToxicContent         bool               `json:"toxic_content"`
	OffTopicDetected     bool               `json:"off_topic_detected"`
	HallucinationRisk    bool               `json:"hallucination_risk"`
	GuardrailsLatencyMs  float64            `json:"guardrails_latency_ms,omitempty"`
	GuardrailsError      string             `json:"guardrails_error,omitempty"`

	// aTLS & Attestation (extends Auth section above)
	ATLSHandshake     bool           `json:"atls_handshake"`
	ATLSHandshakeMs   float64        `json:"atls_handshake_ms,omitempty"`
	AttestationError  string         `json:"attestation_error,omitempty"`
	AttestationNonce  string         `json:"attestation_nonce,omitempty"`
	AttestationReport map[string]any `json:"attestation_report,omitempty"`

	// Error handling
	Error     string `json:"error,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`

	// Additional metadata
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Middleware provides structured audit logging.
type Middleware struct {
	logger *slog.Logger
	config Config
}

type Service interface {
	// Middleware returns the HTTP middleware function for audit logging.
	Middleware(next http.Handler) http.Handler
}

type Config struct {
	EnablePIIMask    bool
	EnableTokens     bool
	SensitiveHeaders []string
	ComplianceMode   bool
	MaxBodyCapture   int
}

// responseCapture captures response data for audit logging.
type responseCapture struct {
	http.ResponseWriter

	statusCode      int
	size            int64
	body            *bytes.Buffer
	responseHeaders http.Header
}

func (rc *responseCapture) WriteHeader(statusCode int) {
	rc.statusCode = statusCode
	// Capture response headers before they're written
	rc.responseHeaders = rc.ResponseWriter.Header().Clone()
	rc.ResponseWriter.WriteHeader(statusCode)
}

func (rc *responseCapture) Write(data []byte) (int, error) {
	rc.size += int64(len(data))
	if rc.body != nil && rc.body.Len() < 1024*10 { // Capture first 10KB
		rc.body.Write(data)
	}

	return rc.ResponseWriter.Write(data)
}

// Header returns the header map to allow setting response headers.
func (rc *responseCapture) Header() http.Header {
	return rc.ResponseWriter.Header()
}

// NewMiddleware creates a new audit middleware instance.
func NewMiddleware(logger *slog.Logger, config Config) *Middleware {
	if config.MaxBodyCapture == 0 {
		config.MaxBodyCapture = 10240 // 10KB default
	}

	return &Middleware{
		logger: logger,
		config: config,
	}
}

// Middleware returns the HTTP middleware function.
func (am *Middleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Generate trace and request IDs
		traceID, err := generateID()
		if err != nil {
			am.logger.Error("failed to generate trace ID", slog.String("error", err.Error()))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)

			return
		}

		requestID, err := generateID()
		if err != nil {
			am.logger.Error("failed to generate request ID", slog.String("error", err.Error()))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)

			return
		}

		// Create context with trace information
		ctx := context.WithValue(r.Context(), TraceIDCtxKey, traceID)
		ctx = context.WithValue(ctx, RequestIDCtxKey, requestID)
		r = r.WithContext(ctx)

		// Capture request body for audit (if needed)
		var requestBody []byte

		if am.shouldCaptureBody(r) {
			body, err := io.ReadAll(r.Body)
			if err == nil {
				if len(body) > am.config.MaxBodyCapture {
					requestBody = body[:am.config.MaxBodyCapture]
				} else {
					requestBody = body
				}

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
		event := am.createAuditEvent(r, capture, &userInfo, traceID, requestID, start, upstreamDuration, requestBody)
		am.logAuditEvent(ctx, &event)
	})
}

// ExtractEventType extracts aTLS/attestation information from response headers.
func (am *Middleware) ExtractEventType(headers http.Header) string {
	if et := headers.Get(HeaderXEventType); et != "" {
		return et
	}

	return "llm_request"
}

func (am *Middleware) createAuditEvent(
	r *http.Request,
	capture *responseCapture,
	session *authn.Session,
	traceID, requestID string,
	start time.Time,
	upstreamDuration time.Duration,
	requestBody []byte,
) Event {
	duration := time.Since(start)
	durationMs := float64(duration.Nanoseconds()) / 1e6
	upstreamMs := float64(upstreamDuration.Nanoseconds()) / 1e6

	event := Event{
		TraceID:          traceID,
		RequestID:        requestID,
		Timestamp:        start,
		EventType:        am.ExtractEventType(capture.responseHeaders),
		Session:          *session,
		Method:           r.Method,
		Path:             r.URL.Path,
		Endpoint:         fmt.Sprintf("%s %s", r.Method, r.URL.Path),
		UserAgent:        r.UserAgent(),
		ClientIP:         am.extractClientIP(r),
		StatusCode:       capture.statusCode,
		ResponseSize:     capture.size,
		RequestSize:      r.ContentLength,
		Duration:         duration,
		DurationMs:       durationMs,
		UpstreamDuration: upstreamDuration,
		UpstreamMs:       upstreamMs,
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
	if am.config.EnablePIIMask {
		event.PIIDetected = am.detectPII(requestBody, capture.body.Bytes())
	}

	event.ContentFiltered = am.checkContentFilter(&event)

	// Extract attestation information from response headers (set by instrumented transport)
	am.extractAttestationInfo(&event, capture.responseHeaders)

	// Extract guardrails detection information from response headers and body
	am.extractGuardrailsInfo(&event, capture.responseHeaders, capture.body.Bytes())

	return event
}

func (am *Middleware) extractLLMMetadata(event *Event, requestBody []byte) {
	var requestData map[string]any
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
	if messages, ok := requestData["messages"].([]any); ok {
		totalChars := 0

		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]any); ok {
				if content, ok := msgMap["content"].(string); ok {
					totalChars += len(content)
				}
			}
		}

		event.InputTokens = totalChars / 4 // Rough token estimate
	}
}

func (am *Middleware) extractLLMResponse(event *Event, responseBody []byte) {
	var responseData map[string]any
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		return
	}

	if usage, ok := responseData["usage"].(map[string]any); ok {
		if promptTokens, ok := usage["prompt_tokens"].(float64); ok {
			event.InputTokens = int(promptTokens)
		}

		if completionTokens, ok := usage["completion_tokens"].(float64); ok {
			event.OutputTokens = int(completionTokens)
		}
	}
}

func (am *Middleware) extractAttestationInfo(event *Event, headers http.Header) {
	if headers == nil {
		return
	}

	// Extract TLS connection details (available for any TLS connection)
	// Using canonical header names (Go's http package canonicalizes headers)
	if tlsVersion := headers.Get("X-Tls-Version"); tlsVersion != "" {
		event.TLSVersion = tlsVersion
	}

	if cipherSuite := headers.Get("X-Tls-Cipher-Suite"); cipherSuite != "" {
		event.CipherSuite = cipherSuite
	}

	if peerCertIssuer := headers.Get("X-Tls-Peer-Cert-Issuer"); peerCertIssuer != "" {
		event.PeerCertIssuer = peerCertIssuer
	}

	// Check for attestation type header (indicates aTLS was configured)
	am.extractAttestationDetails(headers, event)
}

func (am *Middleware) extractAttestationDetails(headers http.Header, event *Event) {
	atlsType := headers.Get("X-Attestation-Type")
	if atlsType == "" {
		return
	}

	event.AttestationType = atlsType
	event.ATLSHandshake = headers.Get("X-Atls-Handshake") == strTrue
	event.AttestationOK = headers.Get("X-Attestation-Ok") == strTrue

	if atlsError := headers.Get("X-Attestation-Error"); atlsError != "" {
		event.AttestationError = atlsError
	}

	if atlsNonce := headers.Get("X-Attestation-Nonce"); atlsNonce != "" {
		event.AttestationNonce = atlsNonce
	}

	if handshakeMs := headers.Get("X-Atls-Handshake-Ms"); handshakeMs != "" {
		if ms, err := strconv.ParseFloat(handshakeMs, 64); err == nil {
			event.ATLSHandshakeMs = ms
		}
	}

	if reportJSON := headers.Get("X-Attestation-Report"); reportJSON != "" {
		var report map[string]any
		if err := json.Unmarshal([]byte(reportJSON), &report); err == nil {
			event.AttestationReport = report
		}
	}
}

// extractGuardrailsInfo extracts guardrails detection information from response headers and body.
func (am *Middleware) extractGuardrailsInfo(event *Event, headers http.Header, responseBody []byte) {
	// Extract from response headers (if guardrails service sets them)
	am.extractGuardrailsFromHeaders(event, headers)

	// Extract guardrails info from response body (JSON format)
	if len(responseBody) > 0 {
		am.extractGuardrailsFromResponseBody(event, responseBody)
	}

	// Derive detection flags from violations if not already set
	am.deriveDetectionFlags(event)
}

func (am *Middleware) extractGuardrailsFromHeaders(event *Event, headers http.Header) {
	if headers == nil {
		return
	}

	if processed := headers.Get(HeaderGuardrailsProcessed); processed == strTrue {
		event.GuardrailsProcessed = true
	}

	if decision := headers.Get(HeaderGuardrailsDecision); decision != "" {
		event.GuardrailsDecision = decision
	}

	if inputRails := headers.Get(HeaderInputRailsTriggered); inputRails != "" {
		event.TriggeredInputRails = strings.Split(inputRails, ",")
	}

	if outputRails := headers.Get(HeaderOutputRailsTriggered); outputRails != "" {
		event.TriggeredOutputRails = strings.Split(outputRails, ",")
	}

	event.SensitiveDataMasked = headers.Get(HeaderSensitiveDataMasked) == strTrue
	event.PromptInjection = headers.Get(HeaderPromptInjection) == strTrue
	event.JailbreakAttempt = headers.Get(HeaderJailbreakAttempt) == strTrue
	event.ToxicContent = headers.Get(HeaderToxicContent) == strTrue
	event.OffTopicDetected = headers.Get(HeaderOffTopic) == strTrue
	event.HallucinationRisk = headers.Get(HeaderHallucinationRisk) == strTrue

	if latencyStr := headers.Get(HeaderGuardrailsLatencyMs); latencyStr != "" {
		if latency, err := strconv.ParseFloat(latencyStr, 64); err == nil {
			event.GuardrailsLatencyMs = latency
		}
	}

	if grError := headers.Get(HeaderGuardrailsError); grError != "" {
		event.GuardrailsError = grError
	}

	// Parse violations from JSON header if present
	if violationsJSON := headers.Get(HeaderGuardrailsViolations); violationsJSON != "" {
		var violations []GuardrailsResult
		if err := json.Unmarshal([]byte(violationsJSON), &violations); err == nil {
			event.GuardrailsViolations = violations
		}
	}
}

// extractGuardrailsFromResponseBody parses guardrails response body for detection info.
func (am *Middleware) extractGuardrailsFromResponseBody(event *Event, responseBody []byte) {
	var responseData map[string]any
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		return
	}

	guardrails, ok := responseData["guardrails"].(map[string]any)
	if !ok {
		return
	}

	event.GuardrailsProcessed = true

	if decision, ok := guardrails["decision"].(string); ok {
		event.GuardrailsDecision = decision
	}

	am.extractTriggeredRails(event, guardrails)
	am.extractViolations(event, guardrails)

	// Extract latency if present
	if latency, ok := guardrails["latency_ms"].(float64); ok {
		event.GuardrailsLatencyMs = latency
	}

	// Extract error if present
	if grError, ok := guardrails["error"].(string); ok {
		event.GuardrailsError = grError
	}
}

func (am *Middleware) extractTriggeredRails(event *Event, guardrails map[string]any) {
	if inputRails, ok := guardrails["triggered_input_rails"].([]any); ok {
		for _, rail := range inputRails {
			if railStr, ok := rail.(string); ok {
				event.TriggeredInputRails = append(event.TriggeredInputRails, railStr)
			}
		}
	}

	if outputRails, ok := guardrails["triggered_output_rails"].([]any); ok {
		for _, rail := range outputRails {
			if railStr, ok := rail.(string); ok {
				event.TriggeredOutputRails = append(event.TriggeredOutputRails, railStr)
			}
		}
	}
}

func (am *Middleware) extractViolations(event *Event, guardrails map[string]any) {
	violations, ok := guardrails["violations"].([]any)
	if !ok {
		return
	}

	for _, v := range violations {
		vMap, ok := v.(map[string]any)
		if !ok {
			continue
		}

		violation := GuardrailsResult{}
		if t, ok := vMap["type"].(string); ok {
			violation.Type = t
		}

		if cat, ok := vMap["category"].(string); ok {
			violation.Category = cat
		}

		if sev, ok := vMap["severity"].(string); ok {
			violation.Severity = sev
		}

		if desc, ok := vMap["description"].(string); ok {
			violation.Description = desc
		}

		if entity, ok := vMap["entity"].(string); ok {
			violation.Entity = entity
		}

		if conf, ok := vMap["confidence"].(float64); ok {
			violation.Confidence = conf
		}

		if action, ok := vMap["action"].(string); ok {
			violation.Action = action
		}

		if rail, ok := vMap["rail"].(string); ok {
			violation.Rail = rail
		}

		event.GuardrailsViolations = append(event.GuardrailsViolations, violation)
	}
}

// deriveDetectionFlags sets boolean detection flags based on violations and triggered rails.
func (am *Middleware) deriveDetectionFlags(event *Event) {
	// Check for specific violation types
	for _, v := range event.GuardrailsViolations {
		switch strings.ToLower(v.Type) {
		case "pii", "sensitive_data", "pii_detected":
			event.PIIDetected = true
			event.SensitiveDataMasked = v.Action == "masked"
		case "prompt_injection", "injection":
			event.PromptInjection = true
		case "jailbreak", "jailbreak_attempt":
			event.JailbreakAttempt = true
		case "toxic", "toxic_content", "toxicity":
			event.ToxicContent = true
		case "off_topic", "off-topic":
			event.OffTopicDetected = true
		case "hallucination", "hallucination_risk":
			event.HallucinationRisk = true
		}
	}

	// Check triggered rails for detection patterns
	allRails := make([]string, 0, len(event.TriggeredInputRails)+len(event.TriggeredOutputRails))
	allRails = append(allRails, event.TriggeredInputRails...)
	allRails = append(allRails, event.TriggeredOutputRails...)

	for _, rail := range allRails {
		railLower := strings.ToLower(rail)
		if strings.Contains(railLower, "pii") || strings.Contains(railLower, "mask sensitive") {
			event.PIIDetected = true
			event.SensitiveDataMasked = true
		}

		if strings.Contains(railLower, "jailbreak") {
			event.JailbreakAttempt = true
		}

		if strings.Contains(railLower, "injection") || strings.Contains(railLower, "prompt injection") {
			event.PromptInjection = true
		}

		if strings.Contains(railLower, "toxic") {
			event.ToxicContent = true
		}

		if strings.Contains(railLower, "off_topic") || strings.Contains(railLower, "off topic") {
			event.OffTopicDetected = true
		}

		if strings.Contains(railLower, "hallucination") {
			event.HallucinationRisk = true
		}
	}

	// If any violation was detected, set content filtered flag
	if len(event.GuardrailsViolations) > 0 ||
		event.GuardrailsDecision == "BLOCK" ||
		event.GuardrailsDecision == "MODIFY" {
		event.ContentFiltered = true
	}
}

func (am *Middleware) logAuditEvent(ctx context.Context, event *Event) {
	logAttrs := []slog.Attr{
		slog.String("trace_id", event.TraceID),
		slog.String("request_id", event.RequestID),
		slog.Time("timestamp", event.Timestamp),
		slog.String("event_type", event.EventType),
		slog.String("user_id", event.Session.UserID),
		slog.String("method", event.Method),
		slog.String("path", event.Path),
		slog.String("endpoint", event.Endpoint),
		slog.String("client_ip", event.ClientIP),
		slog.Int("status_code", event.StatusCode),
		slog.Float64("duration_ms", event.DurationMs),
		slog.Float64("upstream_duration_ms", event.UpstreamMs),
		slog.Int64("response_size", event.ResponseSize),
		slog.Int64("request_size", event.RequestSize),
	}

	if event.Model != "" {
		logAttrs = append(logAttrs,
			slog.String("model", event.Model),
			slog.Int("input_tokens", event.InputTokens),
			slog.Int("output_tokens", event.OutputTokens),
		)
	}

	logAttrs = append(logAttrs,
		slog.Bool("pii_detected", event.PIIDetected),
		slog.Bool("content_filtered", event.ContentFiltered),
	)

	// Add TLS connection details
	if event.TLSVersion != "" {
		logAttrs = append(logAttrs,
			slog.String("tls_version", event.TLSVersion),
		)

		if event.CipherSuite != "" {
			logAttrs = append(logAttrs, slog.String("cipher_suite", event.CipherSuite))
		}

		if event.PeerCertIssuer != "" {
			logAttrs = append(logAttrs, slog.String("peer_cert_issuer", event.PeerCertIssuer))
		}
	}

	// Add attestation details if aTLS was configured or used
	if event.ATLSHandshake || event.AttestationType != "" {
		logAttrs = append(logAttrs,
			slog.Bool("atls_handshake", event.ATLSHandshake),
			slog.Float64("atls_handshake_ms", event.ATLSHandshakeMs),
			slog.Bool("attestation_ok", event.AttestationOK),
			slog.String("attestation_type", event.AttestationType),
		)

		if event.AttestationError != "" {
			logAttrs = append(logAttrs, slog.String("attestation_error", event.AttestationError))
		}

		if event.AttestationNonce != "" {
			logAttrs = append(logAttrs, slog.String("attestation_nonce", event.AttestationNonce))
		}
	}

	// Add guardrails detection details
	if event.GuardrailsProcessed {
		logAttrs = append(logAttrs,
			slog.Bool("guardrails_processed", event.GuardrailsProcessed),
			slog.String("guardrails_decision", event.GuardrailsDecision),
			slog.Float64("guardrails_latency_ms", event.GuardrailsLatencyMs),
		)

		if len(event.TriggeredInputRails) > 0 {
			logAttrs = append(logAttrs, slog.Any("triggered_input_rails", event.TriggeredInputRails))
		}

		if len(event.TriggeredOutputRails) > 0 {
			logAttrs = append(logAttrs, slog.Any("triggered_output_rails", event.TriggeredOutputRails))
		}

		if len(event.GuardrailsViolations) > 0 {
			logAttrs = append(logAttrs,
				slog.Int("guardrails_violations_count", len(event.GuardrailsViolations)),
				slog.Any("guardrails_violations", event.GuardrailsViolations),
			)
		}

		if event.GuardrailsError != "" {
			logAttrs = append(logAttrs, slog.String("guardrails_error", event.GuardrailsError))
		}
	}

	// Add specific guardrails detection flags
	if event.PromptInjection || event.JailbreakAttempt || event.ToxicContent ||
		event.OffTopicDetected || event.HallucinationRisk || event.SensitiveDataMasked {
		logAttrs = append(logAttrs,
			slog.Bool("prompt_injection", event.PromptInjection),
			slog.Bool("jailbreak_attempt", event.JailbreakAttempt),
			slog.Bool("toxic_content", event.ToxicContent),
			slog.Bool("off_topic_detected", event.OffTopicDetected),
			slog.Bool("hallucination_risk", event.HallucinationRisk),
			slog.Bool("sensitive_data_masked", event.SensitiveDataMasked),
		)
	}

	logAttrs = append(logAttrs, slog.Any("event", event))

	am.logger.LogAttrs(ctx, slog.LevelInfo, "audit_event", logAttrs...)
}

// Utility functions.
func generateID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate ID: %w", err)
	}

	return hex.EncodeToString(b), nil
}

func (am *Middleware) shouldCaptureBody(r *http.Request) bool {
	return r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch
}

func (am *Middleware) extractUserInfo(r *http.Request) authn.Session {
	session, ok := r.Context().Value(authn.SessionKey).(authn.Session)
	if !ok {
		return authn.Session{}
	}

	return session
}

func (am *Middleware) extractClientIP(r *http.Request) string {
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

func (am *Middleware) sanitizeHeaders(headers http.Header) map[string]string {
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

func (am *Middleware) getTLSVersion(r *http.Request) string {
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

func (am *Middleware) detectPII(requestBody, responseBody []byte) bool {
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

func (am *Middleware) checkContentFilter(_ *Event) bool {
	// Implement content filtering logic
	// This could integrate with external content filtering services
	return false
}

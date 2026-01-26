// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// AttestationContextKey is the context key for attestation results.
type attestationContextKey struct{}

// AttestationContextKey is exported for use in middleware.
var AttestationContextKey = attestationContextKey{}

// AttestationResult holds the results of an aTLS handshake and attestation verification.
type AttestationResult struct {
	// Handshake details
	ATLSHandshake   bool          `json:"atls_handshake"`
	HandshakeDuration time.Duration `json:"handshake_duration"`

	// Attestation verification
	AttestationOK    bool   `json:"attestation_ok"`
	AttestationError string `json:"attestation_error,omitempty"`

	// Platform details
	AttestationType string `json:"attestation_type,omitempty"` // SNP, TDX, Azure, NoCC
	AttestationNonce string `json:"attestation_nonce,omitempty"`

	// Report details (platform-specific)
	Report map[string]any `json:"report,omitempty"`

	// TLS details
	TLSVersion     string `json:"tls_version,omitempty"`
	CipherSuite    string `json:"cipher_suite,omitempty"`
	ServerName     string `json:"server_name,omitempty"`
	PeerCertIssuer string `json:"peer_cert_issuer,omitempty"`
	PeerCertSerial string `json:"peer_cert_serial,omitempty"`
}

// InstrumentedTransport wraps an http.RoundTripper to capture aTLS and attestation details.
type InstrumentedTransport struct {
	base            http.RoundTripper
	attestationType string
	mu              sync.RWMutex
	lastResult      *AttestationResult
}

// NewInstrumentedTransport creates a new instrumented transport wrapper.
func NewInstrumentedTransport(base http.RoundTripper, attestationType string) *InstrumentedTransport {
	return &InstrumentedTransport{
		base:            base,
		attestationType: attestationType,
	}
}

// RoundTrip implements http.RoundTripper and captures attestation details.
func (t *InstrumentedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	// Perform the actual request
	resp, err := t.base.RoundTrip(req)

	handshakeDuration := time.Since(start)

	// Check if aTLS is expected based on configuration
	atlsExpected := t.attestationType != "" && t.attestationType != "NoCC"

	// Capture attestation result
	result := &AttestationResult{
		ATLSHandshake:     false, // Will be set to true only if we have actual TLS state
		HandshakeDuration: handshakeDuration,
		AttestationType:   t.attestationType,
	}

	// Extract TLS connection state if available
	if resp != nil && resp.TLS != nil {
		t.extractTLSDetails(result, resp.TLS, req)
		// Only mark as aTLS handshake if we actually have TLS and aTLS was expected
		result.ATLSHandshake = atlsExpected
	}

	// Determine attestation status based on response and TLS state
	if err != nil {
		result.AttestationOK = false
		result.AttestationError = err.Error()
	} else if result.ATLSHandshake {
		// If we got a successful response with aTLS, attestation passed
		// (the TLS handshake would have failed if attestation failed)
		result.AttestationOK = true
	} else if atlsExpected && resp != nil && resp.TLS == nil {
		// aTLS was expected but no TLS connection - this is an error
		result.AttestationOK = false
		result.AttestationError = "aTLS expected but connection is not TLS"
	}

	// Store the result
	t.mu.Lock()
	t.lastResult = result
	t.mu.Unlock()

	// Add attestation result to response headers for audit middleware to pick up
	// (context doesn't flow back from RoundTrip, so we use headers)
	if resp != nil {
		// Always set attestation headers when aTLS is configured (even if not actually used)
		// This allows the audit log to show the expected vs actual state
		if atlsExpected || result.ATLSHandshake {
			resp.Header.Set("X-Attestation-Type", result.AttestationType)
			resp.Header.Set("X-Attestation-OK", boolToString(result.AttestationOK))
			resp.Header.Set("X-ATLS-Handshake", boolToString(result.ATLSHandshake))
			resp.Header.Set("X-ATLS-Handshake-Ms", floatToString(float64(result.HandshakeDuration.Nanoseconds())/1e6))
			if result.AttestationError != "" {
				resp.Header.Set("X-Attestation-Error", result.AttestationError)
			}
			if result.AttestationNonce != "" {
				resp.Header.Set("X-Attestation-Nonce", result.AttestationNonce)
			}
		}
	}

	return resp, err
}

// extractTLSDetails extracts TLS connection details from the connection state.
func (t *InstrumentedTransport) extractTLSDetails(result *AttestationResult, state *tls.ConnectionState, _ *http.Request) {
	result.TLSVersion = tlsVersionString(state.Version)
	result.CipherSuite = tls.CipherSuiteName(state.CipherSuite)
	result.ServerName = state.ServerName

	// Extract nonce from ServerName if it's an aTLS connection
	if len(state.ServerName) > 6 && state.ServerName[len(state.ServerName)-6:] == ".nonce" {
		nonceHex := state.ServerName[:len(state.ServerName)-6]
		result.AttestationNonce = nonceHex
	}

	// Extract peer certificate details
	if len(state.PeerCertificates) > 0 {
		cert := state.PeerCertificates[0]
		result.PeerCertIssuer = cert.Issuer.String()
		result.PeerCertSerial = hex.EncodeToString(cert.SerialNumber.Bytes())

		// Extract attestation report from certificate extensions
		result.Report = extractAttestationFromCert(cert, t.attestationType)
	}
}

// extractAttestationFromCert extracts attestation report data from certificate extensions.
func extractAttestationFromCert(cert *x509.Certificate, platformType string) map[string]any {
	// This is a simplified extraction - the actual attestation report
	// is verified during the TLS handshake by VerifyPeerCertificate
	report := make(map[string]any)

	report["platform"] = platformType
	report["verified_at"] = time.Now().UTC().Format(time.RFC3339)

	// Add certificate validity information
	report["cert_not_before"] = cert.NotBefore.UTC().Format(time.RFC3339)
	report["cert_not_after"] = cert.NotAfter.UTC().Format(time.RFC3339)
	report["cert_subject"] = cert.Subject.String()

	// Platform-specific report fields would be extracted here
	// For now, we indicate that verification happened
	switch platformType {
	case "SNP", "SNPvTPM":
		report["technology"] = "AMD SEV-SNP"
	case "TDX":
		report["technology"] = "Intel TDX"
	case "Azure":
		report["technology"] = "Azure Confidential Computing"
	default:
		report["technology"] = "Unknown"
	}

	return report
}

// GetLastResult returns the last attestation result (thread-safe).
func (t *InstrumentedTransport) GetLastResult() *AttestationResult {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.lastResult == nil {
		return nil
	}

	// Return a copy
	result := *t.lastResult
	return &result
}

// Helper functions

func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS13:
		return "TLS1.3"
	case tls.VersionTLS12:
		return "TLS1.2"
	case tls.VersionTLS11:
		return "TLS1.1"
	case tls.VersionTLS10:
		return "TLS1.0"
	default:
		return "Unknown"
	}
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func floatToString(f float64) string {
	return strconv.FormatFloat(f, 'f', 3, 64)
}

// ContextWithAttestation adds attestation result to context.
func ContextWithAttestation(ctx context.Context, result *AttestationResult) context.Context {
	return context.WithValue(ctx, AttestationContextKey, result)
}

// AttestationFromContext retrieves attestation result from context.
func AttestationFromContext(ctx context.Context) *AttestationResult {
	if result, ok := ctx.Value(AttestationContextKey).(*AttestationResult); ok {
		return result
	}
	return nil
}

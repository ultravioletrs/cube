// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	// Header names for attestation data.
	headerTLSVersion        = "X-Tls-Version"
	headerTLSCipherSuite    = "X-Tls-Cipher-Suite"
	headerTLSPeerCertIssuer = "X-Tls-Peer-Cert-Issuer"
	headerAttestationType   = "X-Attestation-Type"
	headerAttestationOK     = "X-Attestation-Ok"
	headerAttestationError  = "X-Attestation-Error"
	headerAttestationNonce  = "X-Attestation-Nonce"
	headerAttestationReport = "X-Attestation-Report"
	headerATLSHandshake     = "X-Atls-Handshake"
	headerATLSHandshakeMs   = "X-Atls-Handshake-Ms"

	// Common string constants.
	strTrue    = "true"
	strFalse   = "false"
	strUnknown = "Unknown"
)

// Platform-specific OIDs for attestation certificate extensions.
// These are defined by the cocos aTLS implementation.
//
//nolint:gochecknoglobals // OIDs are constants that need to be compared by value
var (
	snpvTPMOID = asn1.ObjectIdentifier{2, 99999, 1, 0}
	azureOID   = asn1.ObjectIdentifier{2, 99999, 1, 1}
	tdxOID     = asn1.ObjectIdentifier{2, 99999, 1, 2}
)

// AttestationContextKey is the context key for attestation results.
type attestationContextKey struct{}

// AttestationContextKey is exported for use in middleware.
//
//nolint:gochecknoglobals // Context keys must be package-level variables
var AttestationContextKey = attestationContextKey{}

// AttestationResult holds the results of an aTLS handshake and attestation verification.
type AttestationResult struct {
	// Handshake details
	ATLSHandshake     bool          `json:"atls_handshake"`
	HandshakeDuration time.Duration `json:"handshake_duration"`

	// Attestation verification
	AttestationOK    bool   `json:"attestation_ok"`
	AttestationError string `json:"attestation_error,omitempty"`

	// Platform details
	AttestationType  string `json:"attestation_type,omitempty"` // SNP, TDX, Azure, NoCC
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
		t.extractTLSDetails(result, resp.TLS)
		// Only mark as aTLS handshake if we actually have TLS and aTLS was expected
		result.ATLSHandshake = atlsExpected
	}

	// Determine attestation status based on response and TLS state
	switch {
	case err != nil:
		result.AttestationOK = false
		result.AttestationError = err.Error()
	case result.ATLSHandshake:
		// If we got a successful response with aTLS, attestation passed
		// (the TLS handshake would have failed if attestation failed)
		result.AttestationOK = true
	case atlsExpected && resp != nil && resp.TLS == nil:
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
		t.setResponseHeaders(resp, result, atlsExpected)
	}

	return resp, err
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

// setResponseHeaders adds attestation result to response headers for audit middleware.
func (t *InstrumentedTransport) setResponseHeaders(resp *http.Response, result *AttestationResult, atlsExpected bool) {
	// Always set TLS details when available (for audit logging)
	if result.TLSVersion != "" {
		resp.Header.Set(headerTLSVersion, result.TLSVersion)
	}

	if result.CipherSuite != "" {
		resp.Header.Set(headerTLSCipherSuite, result.CipherSuite)
	}

	if result.PeerCertIssuer != "" {
		resp.Header.Set(headerTLSPeerCertIssuer, result.PeerCertIssuer)
	}

	// Always set attestation headers when aTLS is configured (even if not actually used)
	// This allows the audit log to show the expected vs actual state
	if !atlsExpected && !result.ATLSHandshake {
		return
	}

	resp.Header.Set(headerAttestationType, result.AttestationType)
	resp.Header.Set(headerAttestationOK, boolToString(result.AttestationOK))
	resp.Header.Set(headerATLSHandshake, boolToString(result.ATLSHandshake))
	resp.Header.Set(headerATLSHandshakeMs, floatToString(float64(result.HandshakeDuration.Nanoseconds())/1e6))

	if result.AttestationError != "" {
		resp.Header.Set(headerAttestationError, result.AttestationError)
	}

	if result.AttestationNonce != "" {
		resp.Header.Set(headerAttestationNonce, result.AttestationNonce)
	}

	// Include attestation report if available
	if len(result.Report) > 0 {
		if reportJSON, jsonErr := json.Marshal(result.Report); jsonErr == nil {
			resp.Header.Set(headerAttestationReport, string(reportJSON))
		}
	}
}

// extractTLSDetails extracts TLS connection details from the connection state.
func (t *InstrumentedTransport) extractTLSDetails(result *AttestationResult, state *tls.ConnectionState) {
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
	report := make(map[string]any)

	report["platform"] = platformType
	report["verified_at"] = time.Now().UTC().Format(time.RFC3339)

	// Add certificate validity information
	report["cert_not_before"] = cert.NotBefore.UTC().Format(time.RFC3339)
	report["cert_not_after"] = cert.NotAfter.UTC().Format(time.RFC3339)
	report["cert_subject"] = cert.Subject.String()
	report["cert_issuer"] = cert.Issuer.String()
	report["cert_serial"] = hex.EncodeToString(cert.SerialNumber.Bytes())

	// Platform-specific technology name
	switch platformType {
	case "SNP", "SNPvTPM":
		report["technology"] = "AMD SEV-SNP"
	case "TDX":
		report["technology"] = "Intel TDX"
	case "Azure":
		report["technology"] = "Azure Confidential Computing"
	default:
		report["technology"] = strUnknown
	}

	// Extract attestation extension from certificate
	for _, ext := range cert.Extensions {
		var (
			extPlatform string
			extOIDStr   string
		)

		switch {
		case ext.Id.Equal(snpvTPMOID):
			extPlatform = "SNPvTPM"
			extOIDStr = "2.99999.1.0"
		case ext.Id.Equal(azureOID):
			extPlatform = "Azure"
			extOIDStr = "2.99999.1.1"
		case ext.Id.Equal(tdxOID):
			extPlatform = "TDX"
			extOIDStr = "2.99999.1.2"
		default:
			continue
		}

		// Found attestation extension
		report["attestation_extension_oid"] = extOIDStr
		report["attestation_extension_platform"] = extPlatform
		report["attestation_extension_critical"] = ext.Critical
		report["attestation_extension_size"] = len(ext.Value)

		// Include the raw attestation data as base64 for verification/debugging
		// The actual report is binary (protobuf for SNP/TDX, JWT for Azure)
		report["attestation_raw_base64"] = base64.StdEncoding.EncodeToString(ext.Value)

		// For SNP/TDX, we could parse the protobuf here if we import the cocos libraries
		// For now, we include the raw data and basic metadata
		break
	}

	return report
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
		return strUnknown
	}
}

func boolToString(b bool) string {
	if b {
		return strTrue
	}

	return strFalse
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

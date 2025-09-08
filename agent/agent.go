// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/ultravioletrs/cocos/pkg/attestation"
	"github.com/ultravioletrs/cocos/pkg/attestation/quoteprovider"
	"github.com/ultravioletrs/cocos/pkg/attestation/vtpm"
)

var (
	// ErrAttestationFailed indicates that attestation failed.
	ErrAttestationFailed = errors.New("attestation failed")
	// ErrAttestationVTpmFailed indicates that vTPM attestation failed.
	ErrAttestationVTpmFailed = errors.New("vTPM attestation failed")
	// ErrAttestationType indicates that the attestation type is invalid.
	ErrAttestationType = errors.New("invalid attestation type")
	// ErrUnauthorized indicates that authentication failed.
	ErrUnauthorized = errors.New("unauthorized")
)

type TLSConfig struct {
	Enabled            bool
	InsecureSkipVerify bool
	CertFile           string
	KeyFile            string
	CAFile             string
	MinVersion         uint16
	MaxVersion         uint16
}

type Config struct {
	OllamaURL string
	TLS       TLSConfig
}

type agentService struct {
	config    *Config
	provider  attestation.Provider
	transport *http.Transport
	auth      authn.Authentication
}

type Service interface {
	Proxy() *httputil.ReverseProxy
	Attestation(
		reportData [quoteprovider.Nonce]byte, nonce [vtpm.Nonce]byte, attType attestation.PlatformType,
	) ([]byte, error)
	Authenticate(req *http.Request) error
	AuthMiddleware(next http.Handler) http.Handler
}

func New(config *Config, auth authn.Authentication, provider attestation.Provider) (Service, error) {
	if config.OllamaURL == "" {
		return nil, errors.New("ollama URL is required")
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	if config.TLS.Enabled {
		tlsConfig, err := setTLSConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to set TLS config: %w", err)
		}

		transport.TLSClientConfig = tlsConfig
	}

	return &agentService{
		config:    config,
		transport: transport,
		provider:  provider,
		auth:      auth,
	}, nil
}

func (a *agentService) Authenticate(req *http.Request) error {
	token := util.ExtractBearerToken(req)

	if token == "" {
		return errors.Wrap(ErrUnauthorized, errors.New("missing or invalid token"))
	}

	_, err := a.auth.Authenticate(req.Context(), token)
	if err != nil {
		return errors.Wrap(ErrUnauthorized, err)
	}

	return nil
}

func (a *agentService) Proxy() *httputil.ReverseProxy {
	target, err := url.Parse(a.config.OllamaURL)
	if err != nil {
		log.Printf("Invalid Ollama URL: %v", err)

		return nil
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Transport = a.transport

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		a.modifyHeaders(req)
		log.Printf("Agent forwarding to Ollama: %s %s", req.Method, req.URL.Path)
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	return proxy
}

func (a *agentService) Attestation(
	reportData [quoteprovider.Nonce]byte, nonce [vtpm.Nonce]byte, attType attestation.PlatformType,
) ([]byte, error) {
	switch attType {
	case attestation.SNP, attestation.TDX:
		rawQuote, err := a.provider.TeeAttestation(reportData[:])
		if err != nil {
			return []byte{}, errors.Wrap(ErrAttestationFailed, err)
		}

		return rawQuote, nil
	case attestation.VTPM:
		vTPMQuote, err := a.provider.VTpmAttestation(nonce[:])
		if err != nil {
			return []byte{}, errors.Wrap(ErrAttestationVTpmFailed, err)
		}

		return vTPMQuote, nil
	case attestation.SNPvTPM:
		vTPMQuote, err := a.provider.Attestation(reportData[:], nonce[:])
		if err != nil {
			return []byte{}, errors.Wrap(ErrAttestationVTpmFailed, err)
		}

		return vTPMQuote, nil
	case attestation.Azure, attestation.NoCC, attestation.AzureToken:
		return []byte{}, ErrAttestationType
	default:
		return []byte{}, ErrAttestationType
	}
}

func (a *agentService) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := a.Authenticate(r)
		if err != nil {
			log.Printf("Authentication failed: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)

			return
		}

		next.ServeHTTP(w, r)
	})
}

func DefaultTLSConfig() TLSConfig {
	return TLSConfig{
		Enabled:            true,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
	}
}

func InsecureTLSConfig() TLSConfig {
	return TLSConfig{
		Enabled:            true,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
	}
}

func (a *agentService) modifyHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")

	req.Header.Del("Authorization")
}

func setTLSConfig(config *Config) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.TLS.InsecureSkipVerify,
	}

	if config.TLS.MinVersion != 0 {
		tlsConfig.MinVersion = config.TLS.MinVersion
	}

	if config.TLS.MaxVersion != 0 {
		tlsConfig.MaxVersion = config.TLS.MaxVersion
	}

	if config.TLS.CertFile != "" && config.TLS.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.TLS.CertFile, config.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

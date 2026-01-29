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

	"github.com/absmach/supermq/pkg/errors"
	"github.com/google/go-sev-guest/proto/sevsnp"
	tdxabi "github.com/google/go-tdx-guest/abi"
	"github.com/google/go-tdx-guest/proto/tdx"
	"github.com/google/go-tpm-tools/proto/attest"
	"github.com/ultravioletrs/cocos/pkg/attestation"
	"github.com/ultravioletrs/cocos/pkg/attestation/quoteprovider"
	"github.com/ultravioletrs/cocos/pkg/attestation/vtpm"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
	// ErrAttestationUnmarshal indicates that the attestation report could not be unmarshaled.
	ErrAttestationUnmarshal = errors.New("failed to unmarshal the attestation report")
)

type TLSConfig struct {
	Enabled            bool   `json:"enabled"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
	CertFile           string `json:"cert_file"`
	KeyFile            string `json:"key_file"`
	CAFile             string `json:"ca_file"`
	MinVersion         uint16 `json:"min_version"`
	MaxVersion         uint16 `json:"max_version"`
}

type Config struct {
	BackendURL string    `json:"backend_url"`
	TLS        TLSConfig `json:"tls"`
}

type agentService struct {
	config     *Config
	provider   attestation.Provider
	transport  *http.Transport
	ccPlatform attestation.PlatformType
}

type Service interface {
	Proxy() http.Handler
	Attestation(
		reportData [quoteprovider.Nonce]byte, nonce [vtpm.Nonce]byte, toJSON bool,
	) ([]byte, error)
}

func New(config *Config, provider attestation.Provider, ccPlatform attestation.PlatformType) (Service, error) {
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
		config:     config,
		transport:  transport,
		provider:   provider,
		ccPlatform: ccPlatform,
	}, nil
}

func (a *agentService) Proxy() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetURL := a.config.BackendURL

		log.Printf("Agent received: %s %s", r.Method, r.URL.Path)

		target, err := url.Parse(targetURL)
		if err != nil {
			log.Printf("Invalid target URL %s: %v", targetURL, err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)

			return
		}

		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.Transport = a.transport

		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			a.modifyHeaders(req)
			log.Printf("Agent forwarding to Backend: %s %s", req.Method, req.URL.Path)
		}

		proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
			log.Printf("Proxy error: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		}

		proxy.ServeHTTP(w, r)
	})
}

func (a *agentService) Attestation(
	reportData [quoteprovider.Nonce]byte, nonce [vtpm.Nonce]byte, toJSON bool,
) ([]byte, error) {
	switch a.ccPlatform {
	case attestation.SNP:
		rawQuote, err := a.provider.TeeAttestation(reportData[:])
		if err != nil {
			return []byte{}, errors.Wrap(ErrAttestationFailed, err)
		}

		if toJSON {
			return a.snpAttestationToJSON(rawQuote)
		}

		return rawQuote, nil
	case attestation.TDX:
		rawQuote, err := a.provider.TeeAttestation(reportData[:])
		if err != nil {
			return []byte{}, errors.Wrap(ErrAttestationFailed, err)
		}

		if toJSON {
			return a.tdxAttestationToJSON(rawQuote)
		}

		return rawQuote, nil
	case attestation.SNPvTPM:
		vTPMQuote, err := a.provider.Attestation(reportData[:], nonce[:])
		if err != nil {
			return []byte{}, errors.Wrap(ErrAttestationVTpmFailed, err)
		}

		if toJSON {
			return a.snpVtpmAttestationToJSON(vTPMQuote)
		}

		return vTPMQuote, nil
	default:
		return []byte{}, ErrAttestationType
	}
}

func (a *agentService) modifyHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Del("Authorization")
}

func (a *agentService) snpAttestationToJSON(report []byte) ([]byte, error) {
	attestationPB := &sevsnp.Attestation{}
	if err := proto.Unmarshal(report, attestationPB); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SNP attestation protobuf: %w", err)
	}

	return protojson.MarshalOptions{
		Multiline: true,
		Indent:    "\t",
	}.Marshal(attestationPB)
}

func (a *agentService) snpVtpmAttestationToJSON(report []byte) ([]byte, error) {
	attestationPB := &attest.Attestation{}
	if err := proto.Unmarshal(report, attestationPB); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SNPvTPM attestation protobuf: %w", err)
	}

	return protojson.MarshalOptions{
		Multiline: true,
		Indent:    "\t",
	}.Marshal(attestationPB)
}

func (a *agentService) tdxAttestationToJSON(vTPMQuote []byte) ([]byte, error) {
	res, err := tdxabi.QuoteToProto(vTPMQuote)
	if err != nil {
		return nil, errors.Wrap(ErrAttestationFailed, err)
	}

	quote, ok := res.(*tdx.QuoteV4)
	if !ok {
		return nil, errors.Wrap(ErrAttestationFailed, errors.New("failed to convert to tdx.QuoteV4"))
	}

	return protojson.MarshalOptions{
		Multiline: true,
		Indent:    "\t",
	}.Marshal(quote)
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

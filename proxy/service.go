// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package proxy

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/absmach/supermq/pkg/errors"
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
	AgentURL string
	TLS      TLSConfig
}

type service struct {
	config    *Config
	transport *http.Transport
}

type Service interface {
	Proxy() *httputil.ReverseProxy
}

func New(config *Config) (Service, error) {
	if config.AgentURL == "" {
		return nil, errors.New("agent URL must be provided")
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

	return &service{
		config:    config,
		transport: transport,
	}, nil
}

func (a *service) Proxy() *httputil.ReverseProxy {
	target, err := url.Parse(a.config.AgentURL)
	if err != nil {
		log.Printf("Invalid Agent URL: %v", err)

		return nil
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Transport = a.transport

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		a.modifyHeaders(req)
		log.Printf("Proxy forwarding to Agent: %s %s", req.Method, req.URL.Path)
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	return proxy
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

func (a *service) modifyHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
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

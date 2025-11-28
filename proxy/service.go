// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/absmach/supermq/pkg/errors"
	"github.com/go-chi/chi/v5"
	"github.com/ultravioletrs/cocos/pkg/clients"
	httpclient "github.com/ultravioletrs/cocos/pkg/clients/http"
)

type service struct {
	config    *clients.AttestedClientConfig
	transport *http.Transport
	secure    string
}

type Service interface {
	Proxy() *httputil.ReverseProxy
	Secure() string
}

func New(config *clients.AttestedClientConfig) (Service, error) {
	if config.URL == "" {
		return nil, errors.New("agent URL must be provided")
	}

	client, err := httpclient.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &service{
		config:    config,
		transport: client.Transport(),
		secure:    client.Secure(),
	}, nil
}

func (s *service) Proxy() *httputil.ReverseProxy {
	target, err := url.Parse(s.config.URL)
	if err != nil {
		log.Printf("Invalid Agent URL: %v", err)

		return nil
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = s.transport

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		s.modifyHeaders(req)

		if domainID := chi.URLParam(req, "domainID"); domainID != "" {
			prefix := fmt.Sprintf("/%s", domainID)
			req.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
			if req.URL.RawPath != "" {
				req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, prefix)
			}
			log.Printf("Proxy forwarding for domain %s to Agent (%s): %s %s", domainID, s.secure, req.Method, req.URL.Path)
		} else {
			log.Printf("Proxy forwarding to Agent (%s): %s %s", s.secure, req.Method, req.URL.Path)
		}
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		log.Printf("Proxy error for %s %s: %v", req.Method, req.URL.Path, err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	return proxy
}

func (s *service) Secure() string {
	return s.secure
}

func (s *service) modifyHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")

	// Add security-related headers if using aTLS
	if s.config.AttestedTLS {
		req.Header.Set("X-Attested-Tls", "true")

		if s.config.ProductName != "" {
			req.Header.Set("X-Product-Name", s.config.ProductName)
		}
	}
}

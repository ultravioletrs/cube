// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"
	"fmt"
	"net/http"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/ultravioletrs/cocos/pkg/clients"
	httpclient "github.com/ultravioletrs/cocos/pkg/clients/http"
)

type Service interface {
	// ProxyRequest checks if the request is allowed.
	ProxyRequest(ctx context.Context, session *authn.Session, path string) error
	// Secure returns the secure connection type.
	Secure() string
}

type service struct {
	config    *clients.AttestedClientConfig
	transport *http.Transport
	secure    string
}

func New(config *clients.AttestedClientConfig) (Service, error) {
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

func (s *service) ProxyRequest(_ context.Context, _ *authn.Session, _ string) error {
	return nil
}

func (s *service) Secure() string {
	return s.secure
}

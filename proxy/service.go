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

type service struct {
	config    *clients.AttestedClientConfig
	transport *http.Transport
	secure    string
	repo      Repository
}

func New(config *clients.AttestedClientConfig, repo Repository) (Service, error) {
	client, err := httpclient.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &service{
		config:    config,
		transport: client.Transport(),
		secure:    client.Secure(),
		repo:      repo,
	}, nil
}

func (s *service) ProxyRequest(_ context.Context, _ *authn.Session, _ string) error {
	return nil
}

func (s *service) Secure() string {
	return s.secure
}

// GetAttestationPolicy implements Service.
func (s *service) GetAttestationPolicy(ctx context.Context, _ *authn.Session) ([]byte, error) {
	return s.repo.GetAttestationPolicy(ctx)
}

// UpdateAttestationPolicy implements Service.
func (s *service) UpdateAttestationPolicy(ctx context.Context, _ *authn.Session, policy []byte) error {
	return s.repo.UpdateAttestationPolicy(ctx, policy)
}

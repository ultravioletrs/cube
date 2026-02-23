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
	"github.com/ultravioletrs/cube/proxy/router"
)

type service struct {
	config    *clients.AttestedClientConfig
	transport *http.Transport
	secure    string
	repo      Repository
	router    *router.Router
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

// NewWithRouter creates a new service with a router for dynamic route management.
func NewWithRouter(config *clients.AttestedClientConfig, repo Repository, rter *router.Router) (Service, error) {
	client, err := httpclient.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &service{
		config:    config,
		transport: client.Transport(),
		secure:    client.Secure(),
		repo:      repo,
		router:    rter,
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

// CreateRoute implements Service.
func (s *service) CreateRoute(
	ctx context.Context, _ *authn.Session, route *router.RouteRule,
) (*router.RouteRule, error) {
	if err := router.ValidateRoute(route); err != nil {
		return nil, err
	}

	existingRoutes, _, err := s.repo.ListRoutes(ctx, 0, MaxLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing routes: %w", err)
	}

	if err := router.DetectConflict(route, existingRoutes); err != nil {
		return nil, err
	}

	createdRoute, err := s.repo.CreateRoute(ctx, route)
	if err != nil {
		return nil, err
	}

	if err := s.refreshRoutes(ctx); err != nil {
		return nil, err
	}

	return createdRoute, nil
}

// GetRoute implements Service.
func (s *service) GetRoute(ctx context.Context, _ *authn.Session, name string) (*router.RouteRule, error) {
	return s.repo.GetRoute(ctx, name)
}

// UpdateRoute implements Service.
func (s *service) UpdateRoute(
	ctx context.Context, _ *authn.Session, name string, route *router.RouteRule,
) (*router.RouteRule, error) {
	if router.IsSystemRoute(name) || router.IsSystemRoute(route.Name) {
		return nil, router.ErrSystemRouteProtected
	}

	if err := router.ValidateRoute(route); err != nil {
		return nil, err
	}

	existingRoutes, _, err := s.repo.ListRoutes(ctx, 0, MaxLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing routes: %w", err)
	}

	var otherRoutes []router.RouteRule

	for _, r := range existingRoutes {
		if r.Name != name {
			otherRoutes = append(otherRoutes, r)
		}
	}

	if err := router.DetectConflict(route, otherRoutes); err != nil {
		return nil, err
	}

	updatedRoute, err := s.repo.UpdateRoute(ctx, name, route)
	if err != nil {
		return nil, err
	}

	if err := s.refreshRoutes(ctx); err != nil {
		return nil, err
	}

	return updatedRoute, nil
}

// DeleteRoute implements Service.
func (s *service) DeleteRoute(ctx context.Context, _ *authn.Session, name string) error {
	// Check if this is a system route
	if router.IsSystemRoute(name) {
		return router.ErrSystemRouteProtected
	}

	// Delete route from database
	if err := s.repo.DeleteRoute(ctx, name); err != nil {
		return err
	}

	// Update in-memory router with new routes from database
	return s.refreshRoutes(ctx)
}

// ListRoutes implements Service.
func (s *service) ListRoutes(
	ctx context.Context, _ *authn.Session, offset, limit uint64,
) ([]router.RouteRule, uint64, error) {
	return s.repo.ListRoutes(ctx, offset, limit)
}

// refreshRoutes updates the in-memory router with routes from database.
func (s *service) refreshRoutes(ctx context.Context) error {
	if s.router == nil {
		return nil // Router not set, skip refresh
	}

	routes, _, err := s.repo.ListRoutes(ctx, 0, MaxLimit)
	if err != nil {
		return err
	}

	s.router.UpdateRoutes(routes)

	return nil
}

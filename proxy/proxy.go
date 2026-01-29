// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/ultravioletrs/cube/proxy/router"
)

type ContextKey string

const MethodContextKey ContextKey = "method"

type Service interface {
	ProxyRequest(ctx context.Context, session *authn.Session, path string) error
	Secure() string
	UpdateAttestationPolicy(ctx context.Context, session *authn.Session, policy []byte) error
	GetAttestationPolicy(ctx context.Context, session *authn.Session) ([]byte, error)
	CreateRoute(ctx context.Context, session *authn.Session, route *router.RouteRule) (*router.RouteRule, error)
	UpdateRoute(ctx context.Context, session *authn.Session, route *router.RouteRule) (*router.RouteRule, error)
	DeleteRoute(ctx context.Context, session *authn.Session, name string) error
	GetRoute(ctx context.Context, session *authn.Session, name string) (*router.RouteRule, error)
	ListRoutes(ctx context.Context, session *authn.Session) ([]router.RouteRule, error)
}

type Repository interface {
	UpdateAttestationPolicy(ctx context.Context, policy []byte) error
	GetAttestationPolicy(ctx context.Context) ([]byte, error)
	CreateRoute(ctx context.Context, route *router.RouteRule) (*router.RouteRule, error)
	UpdateRoute(ctx context.Context, route *router.RouteRule) (*router.RouteRule, error)
	DeleteRoute(ctx context.Context, name string) error
	GetRoute(ctx context.Context, name string) (*router.RouteRule, error)
	ListRoutes(ctx context.Context) ([]router.RouteRule, error)
}

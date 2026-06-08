// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"
	"math"

	"github.com/ultravioletrs/cube/internal/cubeauth"
	"github.com/ultravioletrs/cube/proxy/router"
)

type ContextKey string

const (
	MethodContextKey ContextKey = "method"
	MaxLimit         uint64     = math.MaxInt64
)

type Service interface {
	ProxyRequest(ctx context.Context, session *cubeauth.Session, path string) error
	Secure() string
	UpdateAttestationPolicy(ctx context.Context, session *cubeauth.Session, policy []byte) error
	GetAttestationPolicy(ctx context.Context, session *cubeauth.Session) ([]byte, error)
	CreateRoute(ctx context.Context, session *cubeauth.Session, route *router.RouteRule) (*router.RouteRule, error)
	UpdateRoute(ctx context.Context, session *cubeauth.Session, name string,
		route *router.RouteRule) (*router.RouteRule, error)
	DeleteRoute(ctx context.Context, session *cubeauth.Session, name string) error
	GetRoute(ctx context.Context, session *cubeauth.Session, name string) (*router.RouteRule, error)
	ListRoutes(ctx context.Context, session *cubeauth.Session, offset,
		limit uint64) (routes []router.RouteRule, total uint64, err error)
}

type Repository interface {
	UpdateAttestationPolicy(ctx context.Context, policy []byte) error
	GetAttestationPolicy(ctx context.Context) ([]byte, error)
	CreateRoute(ctx context.Context, route *router.RouteRule) (*router.RouteRule, error)
	UpdateRoute(ctx context.Context, name string, route *router.RouteRule) (*router.RouteRule, error)
	DeleteRoute(ctx context.Context, name string) error
	GetRoute(ctx context.Context, name string) (*router.RouteRule, error)
	ListRoutes(ctx context.Context, offset, limit uint64) (routes []router.RouteRule, total uint64, err error)
}

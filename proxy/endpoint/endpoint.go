// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package endpoint

import (
	"context"
	"errors"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/go-kit/kit/endpoint"
	"github.com/ultravioletrs/cube/proxy"
	"github.com/ultravioletrs/cube/proxy/router"
)

var errInvalidRequestType = errors.New("invalid request type")

type Endpoints struct {
	ProxyRequest            endpoint.Endpoint
	GetAttestationPolicy    endpoint.Endpoint
	UpdateAttestationPolicy endpoint.Endpoint
	CreateRoute             endpoint.Endpoint
	GetRoute                endpoint.Endpoint
	UpdateRoute             endpoint.Endpoint
	DeleteRoute             endpoint.Endpoint
	ListRoutes              endpoint.Endpoint
}

func MakeEndpoints(s proxy.Service) Endpoints {
	return Endpoints{
		ProxyRequest:            MakeProxyRequestEndpoint(s),
		GetAttestationPolicy:    MakeGetAttestationPolicyEndpoint(s),
		UpdateAttestationPolicy: MakeUpdateAttestationPolicyEndpoint(s),
		CreateRoute:             MakeCreateRouteEndpoint(s),
		GetRoute:                MakeGetRouteEndpoint(s),
		UpdateRoute:             MakeUpdateRouteEndpoint(s),
		DeleteRoute:             MakeDeleteRouteEndpoint(s),
		ListRoutes:              MakeListRoutesEndpoint(s),
	}
}

type ProxyRequestRequest struct {
	Session  authn.Session
	DomainID string
	Path     string
}

type ProxyRequestResponse struct {
	Err error
}

func (r ProxyRequestResponse) Failed() error {
	return r.Err
}

func MakeProxyRequestEndpoint(s proxy.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(ProxyRequestRequest)
		if !ok {
			return ProxyRequestResponse{Err: errInvalidRequestType}, nil
		}

		err := s.ProxyRequest(ctx, &req.Session, req.Path)

		return ProxyRequestResponse{Err: err}, nil
	}
}

type GetAttestationPolicyRequest struct {
	Session *authn.Session
}

type GetAttestationPolicyResponse struct {
	Policy []byte
	Err    error
}

func (r GetAttestationPolicyResponse) Failed() error {
	return r.Err
}

func MakeGetAttestationPolicyEndpoint(s proxy.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(GetAttestationPolicyRequest)
		if !ok {
			return GetAttestationPolicyResponse{Err: errInvalidRequestType}, nil
		}

		policy, err := s.GetAttestationPolicy(ctx, req.Session)

		return GetAttestationPolicyResponse{Policy: policy, Err: err}, nil
	}
}

type UpdateAttestationPolicyRequest struct {
	Session *authn.Session
	Policy  []byte
}

type UpdateAttestationPolicyResponse struct {
	Err error
}

func (r UpdateAttestationPolicyResponse) Failed() error {
	return r.Err
}

func MakeUpdateAttestationPolicyEndpoint(s proxy.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(UpdateAttestationPolicyRequest)
		if !ok {
			return UpdateAttestationPolicyResponse{Err: errInvalidRequestType}, nil
		}

		err := s.UpdateAttestationPolicy(ctx, req.Session, req.Policy)

		return UpdateAttestationPolicyResponse{Err: err}, nil
	}
}

type CreateRouteRequest struct {
	Session *authn.Session
	Route   *router.RouteRule
}

type CreateRouteResponse struct {
	Route *router.RouteRule
	Err   error
}

func (r CreateRouteResponse) Failed() error {
	return r.Err
}

func MakeCreateRouteEndpoint(s proxy.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(CreateRouteRequest)
		if !ok {
			return CreateRouteResponse{Err: errInvalidRequestType}, nil
		}

		route, err := s.CreateRoute(ctx, req.Session, req.Route)

		return CreateRouteResponse{Route: route, Err: err}, nil
	}
}

type GetRouteRequest struct {
	Session *authn.Session
	Name    string
}

type GetRouteResponse struct {
	Route *router.RouteRule
	Err   error
}

func (r GetRouteResponse) Failed() error {
	return r.Err
}

func MakeGetRouteEndpoint(s proxy.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(GetRouteRequest)
		if !ok {
			return GetRouteResponse{Err: errInvalidRequestType}, nil
		}

		route, err := s.GetRoute(ctx, req.Session, req.Name)

		return GetRouteResponse{Route: route, Err: err}, nil
	}
}

type UpdateRouteRequest struct {
	Session *authn.Session
	Route   *router.RouteRule
}

type UpdateRouteResponse struct {
	Route *router.RouteRule
	Err   error
}

func (r UpdateRouteResponse) Failed() error {
	return r.Err
}

func MakeUpdateRouteEndpoint(s proxy.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(UpdateRouteRequest)
		if !ok {
			return UpdateRouteResponse{Err: errInvalidRequestType}, nil
		}

		route, err := s.UpdateRoute(ctx, req.Session, req.Route)

		return UpdateRouteResponse{Route: route, Err: err}, nil
	}
}

type DeleteRouteRequest struct {
	Session *authn.Session
	Name    string
}

type DeleteRouteResponse struct {
	Err error
}

func (r DeleteRouteResponse) Failed() error {
	return r.Err
}

func MakeDeleteRouteEndpoint(s proxy.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(DeleteRouteRequest)
		if !ok {
			return DeleteRouteResponse{Err: errInvalidRequestType}, nil
		}

		err := s.DeleteRoute(ctx, req.Session, req.Name)

		return DeleteRouteResponse{Err: err}, nil
	}
}

type ListRoutesRequest struct {
	Session *authn.Session
}

type ListRoutesResponse struct {
	Routes []router.RouteRule
	Err    error
}

func (r ListRoutesResponse) Failed() error {
	return r.Err
}

func MakeListRoutesEndpoint(s proxy.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(ListRoutesRequest)
		if !ok {
			return ListRoutesResponse{Err: errInvalidRequestType}, nil
		}

		routes, err := s.ListRoutes(ctx, req.Session)

		return ListRoutesResponse{Routes: routes, Err: err}, nil
	}
}

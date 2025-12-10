// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package endpoint

import (
	"context"
	"errors"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/go-kit/kit/endpoint"
	"github.com/ultravioletrs/cube/proxy"
)

var errInvalidRequestType = errors.New("invalid request type")

type Endpoints struct {
	ProxyRequest            endpoint.Endpoint
	GetAttestationPolicy    endpoint.Endpoint
	UpdateAttestationPolicy endpoint.Endpoint
}

func MakeEndpoints(s proxy.Service) Endpoints {
	return Endpoints{
		ProxyRequest:            MakeProxyRequestEndpoint(s),
		GetAttestationPolicy:    MakeGetAttestationPolicyEndpoint(s),
		UpdateAttestationPolicy: MakeUpdateAttestationPolicyEndpoint(s),
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

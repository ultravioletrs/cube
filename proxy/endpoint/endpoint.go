// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package endpoint

import (
	"context"
	"errors"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/go-kit/kit/endpoint"
	"github.com/ultraviolet/cube/proxy"
)

var errInvalidRequestType = errors.New("invalid request type")

type Endpoints struct {
	ProxyRequest endpoint.Endpoint
}

func MakeEndpoints(s proxy.Service) Endpoints {
	return Endpoints{
		ProxyRequest: MakeProxyRequestEndpoint(s),
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

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
	ProxyRequest    endpoint.Endpoint
	ListAuditLogs   endpoint.Endpoint
	ExportAuditLogs endpoint.Endpoint
}

func MakeEndpoints(s proxy.Service) Endpoints {
	return Endpoints{
		ProxyRequest:    MakeProxyRequestEndpoint(s),
		ListAuditLogs:   MakeListAuditLogsEndpoint(s),
		ExportAuditLogs: MakeExportAuditLogsEndpoint(s),
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

type ListAuditLogsRequest struct {
	Session  authn.Session
	DomainID string
	Query    proxy.AuditLogQuery
}

type ListAuditLogsResponse struct {
	Logs map[string]any
	Err  error
}

func (r ListAuditLogsResponse) Failed() error {
	return r.Err
}

func MakeListAuditLogsEndpoint(s proxy.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(ListAuditLogsRequest)
		if !ok {
			return ListAuditLogsResponse{Err: errInvalidRequestType}, nil
		}

		logs, err := s.ListAuditLogs(ctx, &req.Session, &req.Query)

		return ListAuditLogsResponse{Logs: logs, Err: err}, nil
	}
}

type ExportAuditLogsRequest struct {
	Session  authn.Session
	DomainID string
	Query    proxy.AuditLogQuery
}

type ExportAuditLogsResponse struct {
	Content     []byte
	ContentType string
	Err         error
}

func (r ExportAuditLogsResponse) Failed() error {
	return r.Err
}

func MakeExportAuditLogsEndpoint(s proxy.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(ExportAuditLogsRequest)
		if !ok {
			return ExportAuditLogsResponse{Err: errInvalidRequestType}, nil
		}

		content, contentType, err := s.ExportAuditLogs(ctx, &req.Session, &req.Query)

		return ExportAuditLogsResponse{Content: content, ContentType: contentType, Err: err}, nil
	}
}

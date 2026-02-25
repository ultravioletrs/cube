// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	mgauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/go-chi/chi/v5"
	"github.com/ultravioletrs/cube/proxy/endpoint"
	"github.com/ultravioletrs/cube/proxy/router"
)

var (
	errInvalidRequestType = errors.New("invalid request type")
	errRouteNameRequired  = errors.New("route name required")
)

func decodeGetAttestationPolicyRequest(ctx context.Context, r *http.Request) (any, error) {
	session, ok := ctx.Value(mgauthn.SessionKey).(mgauthn.Session)
	if !ok {
		return nil, errUnauthorized
	}

	if domainID := chi.URLParam(r, "domainID"); domainID != "" {
		session.DomainID = domainID
	}

	return endpoint.GetAttestationPolicyRequest{
		Session: &session,
	}, nil
}

func encodeGetAttestationPolicyResponse(ctx context.Context, w http.ResponseWriter, response any) error {
	resp, ok := response.(endpoint.GetAttestationPolicyResponse)
	if !ok {
		return errInvalidRequestType
	}

	if err := resp.Failed(); err != nil {
		encodeError(ctx, err, w)

		return nil
	}

	w.Header().Set("Content-Type", ContentType)
	_, err := w.Write(resp.Policy)

	return err
}

func decodeUpdateAttestationPolicyRequest(ctx context.Context, r *http.Request) (any, error) {
	session, ok := ctx.Value(mgauthn.SessionKey).(mgauthn.Session)
	if !ok {
		return nil, errUnauthorized
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	return endpoint.UpdateAttestationPolicyRequest{
		Session: &session,
		Policy:  body,
	}, nil
}

func encodeUpdateAttestationPolicyResponse(ctx context.Context, w http.ResponseWriter, response any) error {
	resp, ok := response.(endpoint.UpdateAttestationPolicyResponse)
	if !ok {
		return errInvalidRequestType
	}

	if err := resp.Failed(); err != nil {
		encodeError(ctx, err, w)

		return nil
	}

	w.WriteHeader(http.StatusCreated)

	return nil
}

func decodeCreateRouteRequest(ctx context.Context, r *http.Request) (any, error) {
	session, ok := ctx.Value(mgauthn.SessionKey).(mgauthn.Session)
	if !ok {
		return nil, errUnauthorized
	}

	var route router.RouteRule
	if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
		return nil, err
	}

	return endpoint.CreateRouteRequest{
		Session: &session,
		Route:   &route,
	}, nil
}

func encodeCreateRouteResponse(ctx context.Context, w http.ResponseWriter, response any) error {
	resp, ok := response.(endpoint.CreateRouteResponse)
	if !ok {
		return errInvalidRequestType
	}

	if err := resp.Failed(); err != nil {
		encodeError(ctx, err, w)

		return nil
	}

	w.Header().Set("Content-Type", ContentType)
	w.WriteHeader(http.StatusCreated)

	return json.NewEncoder(w).Encode(resp.Route)
}

func decodeGetRouteRequest(ctx context.Context, r *http.Request) (any, error) {
	session, ok := ctx.Value(mgauthn.SessionKey).(mgauthn.Session)
	if !ok {
		return nil, errUnauthorized
	}

	name := r.PathValue("name")
	if name == "" {
		return nil, errRouteNameRequired
	}

	return endpoint.GetRouteRequest{
		Session: &session,
		Name:    name,
	}, nil
}

func encodeGetRouteResponse(ctx context.Context, w http.ResponseWriter, response any) error {
	resp, ok := response.(endpoint.GetRouteResponse)
	if !ok {
		return errInvalidRequestType
	}

	if err := resp.Failed(); err != nil {
		encodeError(ctx, err, w)

		return nil
	}

	if resp.Route == nil {
		w.Header().Set("Content-Type", ContentType)
		w.WriteHeader(http.StatusNotFound)

		return json.NewEncoder(w).Encode(map[string]string{
			"error": "route not found",
		})
	}

	w.Header().Set("Content-Type", ContentType)

	return json.NewEncoder(w).Encode(resp.Route)
}

func decodeUpdateRouteRequest(ctx context.Context, r *http.Request) (any, error) {
	session, ok := ctx.Value(mgauthn.SessionKey).(mgauthn.Session)
	if !ok {
		return nil, errUnauthorized
	}

	name := r.PathValue("name")
	if name == "" {
		return nil, errRouteNameRequired
	}

	var route router.RouteRule
	if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
		return nil, err
	}

	return endpoint.UpdateRouteRequest{
		Session: &session,
		Name:    name,
		Route:   &route,
	}, nil
}

func encodeUpdateRouteResponse(ctx context.Context, w http.ResponseWriter, response any) error {
	resp, ok := response.(endpoint.UpdateRouteResponse)
	if !ok {
		return errInvalidRequestType
	}

	if err := resp.Failed(); err != nil {
		encodeError(ctx, err, w)

		return nil
	}

	w.Header().Set("Content-Type", ContentType)
	w.WriteHeader(http.StatusOK)

	return json.NewEncoder(w).Encode(resp.Route)
}

func decodeDeleteRouteRequest(ctx context.Context, r *http.Request) (any, error) {
	session, ok := ctx.Value(mgauthn.SessionKey).(mgauthn.Session)
	if !ok {
		return nil, errUnauthorized
	}

	name := r.PathValue("name")
	if name == "" {
		return nil, errRouteNameRequired
	}

	return endpoint.DeleteRouteRequest{
		Session: &session,
		Name:    name,
	}, nil
}

func encodeDeleteRouteResponse(ctx context.Context, w http.ResponseWriter, response any) error {
	resp, ok := response.(endpoint.DeleteRouteResponse)
	if !ok {
		return errInvalidRequestType
	}

	if err := resp.Failed(); err != nil {
		encodeError(ctx, err, w)

		return nil
	}

	w.Header().Set("Content-Type", ContentType)
	w.WriteHeader(http.StatusOK)

	return json.NewEncoder(w).Encode(map[string]string{
		"message": "route deleted successfully",
	})
}

func decodeListRoutesRequest(ctx context.Context, r *http.Request) (any, error) {
	session, ok := ctx.Value(mgauthn.SessionKey).(mgauthn.Session)
	if !ok {
		return nil, errUnauthorized
	}

	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")

	offset := uint64(0)
	limit := uint64(10)

	if offsetStr != "" {
		var err error

		offset, err = strconv.ParseUint(offsetStr, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	if limitStr != "" {
		var err error

		limit, err = strconv.ParseUint(limitStr, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	return endpoint.ListRoutesRequest{
		Session: &session,
		Offset:  offset,
		Limit:   limit,
	}, nil
}

func encodeListRoutesResponse(ctx context.Context, w http.ResponseWriter, response any) error {
	resp, ok := response.(endpoint.ListRoutesResponse)
	if !ok {
		return errInvalidRequestType
	}

	if err := resp.Failed(); err != nil {
		encodeError(ctx, err, w)

		return nil
	}

	w.Header().Set("Content-Type", ContentType)

	routes := resp.Routes
	if routes == nil {
		routes = []router.RouteRule{}
	}

	return json.NewEncoder(w).Encode(map[string]any{
		"total":  resp.Total,
		"offset": resp.Offset,
		"limit":  resp.Limit,
		"routes": routes,
	})
}

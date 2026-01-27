// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

// Package api provides HTTP transport layer for the agent service.
package api //nolint:revive // api is a standard package name for HTTP handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/absmach/supermq"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ultravioletrs/cube/agent"
	"github.com/ultravioletrs/cube/agent/endpoint"
)

const ContentType = "application/json"

func MakeHandler(svc agent.Service, instanceID string) http.Handler {
	endpoints := endpoint.MakeEndpoints(svc)
	mux := chi.NewRouter()

	mux.Get("/health", supermq.Health("cube-agent", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	mux.Post("/attestation", kithttp.NewServer(
		endpoints.Attestation,
		decodeAttestationRequest,
		encodeAttestationResponse,
		kithttp.ServerErrorEncoder(encodeError),
	).ServeHTTP)

	mux.Handle("/*", svc.Proxy())

	return mux
}

func decodeAttestationRequest(_ context.Context, r *http.Request) (any, error) {
	var req endpoint.AttestationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

func encodeAttestationResponse(ctx context.Context, w http.ResponseWriter, response any) error {
	resp, ok := response.(endpoint.AttestationResponse)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)

		return json.NewEncoder(w).Encode(map[string]string{"error": "invalid response type"})
	}

	if resp.Err != nil {
		encodeError(ctx, resp.Err, w)

		return resp.Err
	}

	// Check if the report is JSON (starts with '{' or '[') or binary
	if len(resp.Report) > 0 && (resp.Report[0] == '{' || resp.Report[0] == '[') {
		w.Header().Set("Content-Type", ContentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	w.WriteHeader(http.StatusOK)
	_, err := w.Write(resp.Report)

	return err
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", ContentType)
	w.WriteHeader(http.StatusBadRequest)

	err = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

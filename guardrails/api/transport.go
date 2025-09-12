// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/errors"

	"github.com/ultraviolet/cube/guardrails"
)

const ContentType = "application/json"

func MakeHandler(svc guardrails.Service, instanceID string) http.Handler {
	mux := chi.NewRouter()

	mux.Get("/health", supermq.Health("cube-guardrails", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	opts := []kithttp.ServerOption{
		kithttp.ServerBefore(kithttp.PopulateRequestContext),
		kithttp.ServerErrorEncoder(encodeError()),
	}

	mux.Route("/v1/chat", func(r chi.Router) {
		r.Post("/completions", kithttp.NewServer(
			chatCompletionEndpoint(svc),
			decodeChatCompletionRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)
	})

	mux.Route("/nemo", func(r chi.Router) {
		r.Get("/config", kithttp.NewServer(
			getNeMoConfigEndpoint(svc),
			decodeEmptyRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Get("/config/yaml", kithttp.NewServer(
			getNeMoConfigYAMLEndpoint(svc),
			decodeEmptyRequest,
			encodeYAMLResponse,
			opts...,
		).ServeHTTP)

	})

	mux.Route("/flows", func(r chi.Router) {
		r.Post("/", kithttp.NewServer(
			createFlowEndpoint(svc),
			decodeCreateFlowRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Get("/{id}", kithttp.NewServer(
			getFlowEndpoint(svc),
			decodeGetFlowRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Get("/", kithttp.NewServer(
			getFlowsEndpoint(svc),
			decodeEmptyRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Put("/{id}", kithttp.NewServer(
			updateFlowEndpoint(svc),
			decodeUpdateFlowRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Delete("/{id}", kithttp.NewServer(
			deleteFlowEndpoint(svc),
			decodeDeleteFlowRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)
	})

	mux.Route("/kb", func(r chi.Router) {
		r.Post("/files", kithttp.NewServer(
			createKBFileEndpoint(svc),
			decodeCreateKBFileRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Get("/files/{id}", kithttp.NewServer(
			getKBFileEndpoint(svc),
			decodeGetKBFileRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Get("/files", kithttp.NewServer(
			getKBFilesEndpoint(svc),
			decodeEmptyRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Get("/files/list", kithttp.NewServer(
			listKBFilesEndpoint(svc),
			decodeListKBFilesRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Put("/files/{id}", kithttp.NewServer(
			updateKBFileEndpoint(svc),
			decodeUpdateKBFileRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Delete("/files/{id}", kithttp.NewServer(
			deleteKBFileEndpoint(svc),
			decodeDeleteKBFileRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Post("/search", kithttp.NewServer(
			searchKBFilesEndpoint(svc),
			decodeSearchKBFilesRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)
	})

	mux.Handle("/*", svc.Proxy())

	return mux
}

func decodeEmptyRequest(_ context.Context, _ *http.Request) (interface{}, error) {
	return struct{}{}, nil
}

func decodeImportConfigRequest(_ context.Context, r *http.Request) (interface{}, error) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return importConfigRequest{Data: data}, nil
}

func decodeChatCompletionRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req chatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}

// Flow decoder functions.
func decodeCreateFlowRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req createFlowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}

func decodeGetFlowRequest(_ context.Context, r *http.Request) (interface{}, error) {
	id := chi.URLParam(r, "id")
	return getFlowRequest{ID: id}, nil
}

func decodeUpdateFlowRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req updateFlowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	req.ID = chi.URLParam(r, "id")
	return req, nil
}

func decodeDeleteFlowRequest(_ context.Context, r *http.Request) (interface{}, error) {
	id := chi.URLParam(r, "id")
	return deleteFlowRequest{ID: id}, nil
}

// KB File decoder functions.
func decodeCreateKBFileRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req createKBFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}

func decodeGetKBFileRequest(_ context.Context, r *http.Request) (interface{}, error) {
	id := chi.URLParam(r, "id")
	return getKBFileRequest{ID: id}, nil
}

func decodeListKBFilesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	category := r.URL.Query().Get("category")
	tags := r.URL.Query()["tags"] // Get array of tags.

	var req listKBFilesRequest
	req.Limit = 10
	req.Offset = 0
	req.Category = category
	req.Tags = tags

	if limitStr != "" {
		limit, err := strconv.ParseUint(limitStr, 10, 64)
		if err != nil {
			return nil, errors.Wrap(errors.ErrMalformedEntity, err)
		}
		req.Limit = limit
	}

	if offsetStr != "" {
		offset, err := strconv.ParseUint(offsetStr, 10, 64)
		if err != nil {
			return nil, errors.Wrap(errors.ErrMalformedEntity, err)
		}
		req.Offset = offset
	}

	return req, nil
}

func decodeUpdateKBFileRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req updateKBFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	req.ID = chi.URLParam(r, "id")
	return req, nil
}

func decodeDeleteKBFileRequest(_ context.Context, r *http.Request) (interface{}, error) {
	id := chi.URLParam(r, "id")
	return deleteKBFileRequest{ID: id}, nil
}

func decodeSearchKBFilesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req searchKBFilesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}

func encodeJSONResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", ContentType)
	return json.NewEncoder(w).Encode(response)
}

func encodeYAMLResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=guardrails-config.yaml")
	data, ok := response.([]byte)
	if !ok {
		return errors.New("invalid response type for YAML export")
	}
	_, err := w.Write(data)
	return err
}

func encodeError() kithttp.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter) {
		w.Header().Set("Content-Type", ContentType)

		var statusCode int
		var message string

		switch {
		case errors.Contains(err, errors.ErrMalformedEntity):
			statusCode = http.StatusBadRequest
			message = "Bad Request: Invalid request format"
		case errors.Contains(err, guardrails.ErrNotFound):
			statusCode = http.StatusNotFound
			message = "Not Found"
		default:
			statusCode = http.StatusInternalServerError
			message = "Internal Server Error"
		}

		w.WriteHeader(statusCode)

		errorResponse := map[string]interface{}{
			"error": map[string]interface{}{
				"message": message,
				"type":    "error",
				"code":    statusCode,
			},
		}

		_ = json.NewEncoder(w).Encode(errorResponse)
	}
}

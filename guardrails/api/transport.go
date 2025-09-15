// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	kithttp "github.com/go-kit/kit/transport/http"
	"io"
	"net/http"
	"strconv"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ultraviolet/cube/guardrails"
)

const ContentType = "application/json"

func MakeHandler(svc guardrails.Service, instanceID string) http.Handler {
	mux := chi.NewRouter()

	mux.Get("/health", supermq.Health("cube-guardrails", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	opts := []kithttp.ServerOption{
		kithttp.ServerBefore(kithttp.PopulateRequestContext),
		kithttp.ServerErrorEncoder(encodeError),
	}

	// Policy Management
	mux.Route("/api/v1/policies", func(r chi.Router) {
		r.Post("/", kithttp.NewServer(
			createPolicyEndpoint(svc),
			decodeCreatePolicyRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Get("/{id}", kithttp.NewServer(
			getPolicyEndpoint(svc),
			decodeGetPolicyRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Get("/", kithttp.NewServer(
			listPoliciesEndpoint(svc),
			decodeListPoliciesRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Put("/{id}", kithttp.NewServer(
			updatePolicyEndpoint(svc),
			decodeUpdatePolicyRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Delete("/{id}", kithttp.NewServer(
			deletePolicyEndpoint(svc),
			decodeDeletePolicyRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)
	})

	// Configuration Management
	mux.Route("/api/v1/config", func(r chi.Router) {
		// Topics management
		r.Get("/topics", kithttp.NewServer(
			getRestrictedTopicsEndpoint(svc),
			decodeEmptyRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Put("/topics", kithttp.NewServer(
			updateRestrictedTopicsEndpoint(svc),
			decodeUpdateTopicsRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Post("/topics", kithttp.NewServer(
			addRestrictedTopicEndpoint(svc),
			decodeAddTopicRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Delete("/topics/{topic}", kithttp.NewServer(
			removeRestrictedTopicEndpoint(svc),
			decodeRemoveTopicRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		// Bias patterns management
		r.Get("/bias-patterns", kithttp.NewServer(
			getBiasPatternsEndpoint(svc),
			decodeEmptyRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Put("/bias-patterns", kithttp.NewServer(
			updateBiasPatternsEndpoint(svc),
			decodeUpdateBiasPatternsRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		// Factuality configuration
		r.Get("/factuality", kithttp.NewServer(
			getFactualityConfigEndpoint(svc),
			decodeEmptyRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		r.Put("/factuality", kithttp.NewServer(
			updateFactualityConfigEndpoint(svc),
			decodeUpdateFactualityConfigRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		// Export/Import
		r.Get("/export", kithttp.NewServer(
			exportConfigEndpoint(svc),
			decodeEmptyRequest,
			encodeYAMLResponse,
			opts...,
		).ServeHTTP)

		r.Post("/import", kithttp.NewServer(
			importConfigEndpoint(svc),
			decodeImportConfigRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)

		// Audit log endpoint
		r.Get("/audit", kithttp.NewServer(
			getAuditLogEndpoint(svc),
			decodeGetAuditLogRequest,
			encodeJSONResponse,
			opts...,
		).ServeHTTP)
	})

	// Default proxy handler for all other requests
	mux.Handle("/*", svc.Proxy())

	return mux
}

func decodeEmptyRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func decodeCreatePolicyRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req createPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}

func decodeGetPolicyRequest(_ context.Context, r *http.Request) (interface{}, error) {
	id := chi.URLParam(r, "id")
	return getPolicyRequest{ID: id}, nil
}

func decodeListPoliciesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 10
	offset := 0

	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return nil, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}

	if offsetStr != "" {
		var err error
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			return nil, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}

	return listPoliciesRequest{Limit: limit, Offset: offset}, nil
}

func decodeUpdatePolicyRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req updatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	req.ID = chi.URLParam(r, "id")
	return req, nil
}

func decodeDeletePolicyRequest(_ context.Context, r *http.Request) (interface{}, error) {
	id := chi.URLParam(r, "id")
	return deletePolicyRequest{ID: id}, nil
}

func decodeUpdateTopicsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req updateTopicsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}

func decodeAddTopicRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req addTopicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}

func decodeRemoveTopicRequest(_ context.Context, r *http.Request) (interface{}, error) {
	topic := chi.URLParam(r, "topic")
	return removeTopicRequest{Topic: topic}, nil
}

func decodeUpdateBiasPatternsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req updateBiasPatternsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}

func decodeUpdateFactualityConfigRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req updateFactualityConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return req, nil
}

func decodeGetAuditLogRequest(_ context.Context, r *http.Request) (interface{}, error) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return nil, errors.Wrap(errors.ErrMalformedEntity, err)
		}
	}
	return getAuditLogRequest{Limit: limit}, nil
}

func decodeImportConfigRequest(_ context.Context, r *http.Request) (interface{}, error) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	return importConfigRequest{Data: data}, nil
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

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", ContentType)
	switch errors.Contains(err, errors.ErrMalformedEntity) {
	case true:
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(errorResponse{Error: err.Error()})
}

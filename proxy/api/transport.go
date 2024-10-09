// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ultraviolet/cube/proxy"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const ContentType = "application/json"

func MakeHandler(svc proxy.Service, logger *slog.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	mux := chi.NewRouter()

	mux.HandleFunc("/", otelhttp.NewHandler(kithttp.NewServer(
		identifyEndpoint(svc),
		decodeIdentifyReq,
		encodeResponse,
		opts...,
	), "identify").ServeHTTP)

	mux.Get("/health", magistrala.Health("cube-proxy", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeIdentifyReq(_ context.Context, r *http.Request) (interface{}, error) {
	return identifyRequest{
		Token: apiutil.ExtractBearerToken(r),
	}, nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	var wrapper error
	if errors.Contains(err, apiutil.ErrValidation) {
		wrapper, err = errors.Unwrap(err)
	}

	w.Header().Set("Content-Type", ContentType)
	switch {
	case errors.Contains(err, apiutil.ErrBearerToken),
		errors.Contains(err, svcerr.ErrAuthentication):
		err = unwrap(err)
		w.WriteHeader(http.StatusUnauthorized)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	if wrapper != nil {
		err = errors.Wrap(wrapper, err)
	}

	if errorVal, ok := err.(errors.Error); ok {
		if err := json.NewEncoder(w).Encode(errorVal); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func unwrap(err error) error {
	wrapper, err := errors.Unwrap(err)
	if wrapper != nil {
		return wrapper
	}

	return err
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	if ar, ok := response.(magistrala.Response); ok {
		for k, v := range ar.Headers() {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", ContentType)
		w.WriteHeader(ar.Code())

		if ar.Empty() {
			return nil
		}
	}

	return json.NewEncoder(w).Encode(response)
}

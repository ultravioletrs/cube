// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// GuardrailsController is satisfied by *guardrails.GuardedClient.
// Defined here to avoid importing the guardrails package from the transport layer.
type GuardrailsController interface {
	IsEnabled() bool
	SetEnabled(bool)
}

// MountGuardrails registers the guardrails status endpoints.
// If ctrl is nil (guardrails not configured), the endpoints still respond but
// always report configured=false and ignore enable/disable requests.
func MountGuardrails(r chi.Router, ctrl GuardrailsController) {
	r.Get("/api/v1/guardrails", guardrailsStatusHandler(ctrl))
	r.Put("/api/v1/guardrails", guardrailsSetHandler(ctrl))
}

type guardrailsStatusResponse struct {
	Enabled    bool `json:"enabled"`
	Configured bool `json:"configured"`
}

func guardrailsStatusHandler(ctrl GuardrailsController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ctrl == nil {
			writeJSON(w, http.StatusOK, guardrailsStatusResponse{Enabled: false, Configured: false})
			return
		}
		writeJSON(w, http.StatusOK, guardrailsStatusResponse{Enabled: ctrl.IsEnabled(), Configured: true})
	}
}

func guardrailsSetHandler(ctrl GuardrailsController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}
		if ctrl == nil {
			writeJSON(w, http.StatusOK, guardrailsStatusResponse{Enabled: false, Configured: false})
			return
		}
		ctrl.SetEnabled(body.Enabled)
		writeJSON(w, http.StatusOK, guardrailsStatusResponse{Enabled: ctrl.IsEnabled(), Configured: true})
	}
}

// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"net/http"
	"strings"
)

const (
	modelAuthHeader     = "X-Model-Authorization"
	guardrailsReqHeader = "X-Guardrails-Request"
	authorizationHeader = "Authorization"
	bearerPrefix        = "Bearer "
)

func ModelAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(guardrailsReqHeader) != "true" {
				next.ServeHTTP(w, r)

				return
			}

			if r.Header.Get(authorizationHeader) != "" {
				next.ServeHTTP(w, r)

				return
			}

			modelAuth := r.Header.Get(modelAuthHeader)
			if modelAuth == "" {
				next.ServeHTTP(w, r)

				return
			}

			if !strings.HasPrefix(modelAuth, bearerPrefix) {
				modelAuth = bearerPrefix + modelAuth
			}

			r.Header.Set(authorizationHeader, modelAuth)
			next.ServeHTTP(w, r)
		})
	}
}

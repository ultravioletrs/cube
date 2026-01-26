// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"net/http"
	"strings"

	mgauthn "github.com/absmach/supermq/pkg/authn"
)

const (
	modelAuthHeader     = "X-Model-Authorization"
	guardrailsReqHeader = "X-Guardrails-Request"
	authorizationHeader = "Authorization"
	bearerPrefix        = "Bearer "
)

// ModelAuthMiddleware creates an HTTP middleware that checks for X-Model-Authorization
// header on guardrails requests and copies it to the Authorization header if present.
// This allows the guardrails service to authenticate requests using the original
// user's token when making callbacks to the proxy/agent.
//
// The middleware only activates when X-Guardrails-Request header is set to "true".
func ModelAuthMiddleware(authn mgauthn.AuthNMiddleware) func(http.Handler) http.Handler {
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

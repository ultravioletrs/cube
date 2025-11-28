// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/authz"
	"github.com/go-chi/chi/v5"
)

const (
	userType       = "user"
	usersKind      = "users"
	domainType     = "domain"
	membershipPerm = "membership"
)

func AuthorizationMiddleware(az authz.Authorization) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			session, ok := ctx.Value(authn.SessionKey).(authn.Session)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			domainID := chi.URLParam(r, "domainID")
			if domainID == "" {
				http.Error(w, "Bad Request: missing domainID", http.StatusBadRequest)
				return
			}

			// Check if user has membership permission on the domain
			req := authz.PolicyReq{
				Domain:      domainID,
				SubjectType: userType,
				SubjectKind: usersKind,
				Subject:     session.DomainUserID,
				Permission:  membershipPerm,
				ObjectType:  domainType,
				Object:      domainID,
			}

			if err := az.Authorize(ctx, req); err != nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

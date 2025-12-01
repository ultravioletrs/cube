// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/authz"
	"github.com/go-chi/chi/v5"
)

const (
	userType   = "user"
	usersKind  = "users"
	domainType = "domain"

	membershipPerm         = "membership"
	llmChatCompletionsPerm = "llm_chat_completions_permission"
	llmCompletionsPerm     = "llm_completions_permission"
	auditLogReadPerm       = "audit_log_read_permission"
	auditLogExportPerm     = "audit_log_export_permission"
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

			// Determine the type of authorization check based on the path
			path := r.URL.Path

			// Check for LLM endpoints
			if strings.Contains(path, "/v1/chat/completions") {
				if err := checkLLMPermission(ctx, az, r, session, domainID, llmChatCompletionsPerm); err != nil {
					http.Error(w, "Forbidden: insufficient permissions for chat completions", http.StatusForbidden)
					return
				}
			} else if strings.Contains(path, "/v1/completions") {
				if err := checkLLMPermission(ctx, az, r, session, domainID, llmCompletionsPerm); err != nil {
					http.Error(w, "Forbidden: insufficient permissions for completions", http.StatusForbidden)
					return
				}
			} else if strings.HasPrefix(path, "/"+domainID+"/audit") {
				// Check for audit log endpoints
				permission := auditLogReadPerm
				if strings.Contains(path, "/export") {
					permission = auditLogExportPerm
				}

				req := authz.PolicyReq{
					Domain:      domainID,
					SubjectType: userType,
					SubjectKind: usersKind,
					Subject:     session.DomainUserID,
					Permission:  permission,
					ObjectType:  domainType,
					Object:      domainID,
				}

				if err := az.Authorize(ctx, req); err != nil {
					http.Error(w, "Forbidden: insufficient permissions for audit logs", http.StatusForbidden)
					return
				}
			} else {
				// Default: check domain membership permission
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
			}

			next.ServeHTTP(w, r)
		})
	}
}

// checkLLMPermission checks if the user has permission to access LLM endpoints
// It extracts the model from the request body and checks permissions on the domain level
func checkLLMPermission(ctx context.Context, az authz.Authorization, r *http.Request, session authn.Session, domainID, permission string) error {
	// For LLM requests, we check permission at the domain level
	// The model field in the request is used for LLM selection, not authorization
	req := authz.PolicyReq{
		Domain:      domainID,
		SubjectType: userType,
		SubjectKind: usersKind,
		Subject:     session.DomainUserID,
		Permission:  permission,
		ObjectType:  domainType,
		Object:      domainID,
	}

	return az.Authorize(ctx, req)
}

// extractModelFromRequest extracts the model field from the request body (if needed for future use)
func extractModelFromRequest(r *http.Request) (string, error) {
	if r.Body == nil {
		return "", nil
	}

	// Read the body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	// Restore the body for downstream handlers
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Parse JSON to extract model
	var requestData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &requestData); err != nil {
		return "", err
	}

	if model, ok := requestData["model"].(string); ok {
		return model, nil
	}

	return "", nil
}

// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

// Package auth provides request authentication by delegating token validation
// to the ATOM auth gRPC service.
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	atomv1 "github.com/ultravioletrs/cube/proto/atom/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	userIDKey   contextKey = "user_id"
	domainIDKey contextKey = "domain_id"
)

type Identity struct {
	EntityID string
	TenantID string
}

// Authenticator wraps the ATOM AuthService gRPC client.
type Authenticator struct {
	client  atomv1.AuthServiceClient
	timeout time.Duration
}

// NewAuthenticator dials ATOM auth and returns an Authenticator.
// addr should be host:port, e.g. "atom:8081".
func NewAuthenticator(addr string) (*Authenticator, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("dial auth gRPC %s: %w", addr, err)
	}
	timeout := 15 * time.Second
	return &Authenticator{
		client:  atomv1.NewAuthServiceClient(conn),
		timeout: timeout,
	}, conn, nil
}

// Identify authenticates the token via ATOM and returns the entity identity.
func (a *Authenticator) Identify(ctx context.Context, token string) (Identity, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	res, err := a.client.Authenticate(ctx, &atomv1.AuthenticateRequest{Token: token})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.Unauthenticated, codes.NotFound, codes.PermissionDenied:
				return Identity{}, fmt.Errorf("unauthenticated: %w", err)
			}
		}
		return Identity{}, fmt.Errorf("auth authenticate: %w", err)
	}

	identity := Identity{
		EntityID: res.GetEntityId(),
		TenantID: res.GetTenantId(),
	}
	if identity.EntityID == "" {
		return Identity{}, fmt.Errorf("auth authenticate returned empty entity id")
	}
	return identity, nil
}

// Middleware returns an HTTP middleware that extracts the Bearer token,
// calls the auth gRPC Authenticate RPC, and stores the user ID in the context.
func Middleware(auth *Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if raw == "" {
				writeError(w, http.StatusUnauthorized, "missing authorization token")
				return
			}

			identity, err := auth.Identify(r.Context(), raw)
			if err != nil {
				if strings.Contains(err.Error(), "unauthenticated:") {
					writeError(w, http.StatusUnauthorized, "invalid or expired token")
					return
				}
				writeError(w, http.StatusServiceUnavailable, "authentication service unavailable")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, identity.EntityID)
			if domainID := firstNonEmpty(r.Header.Get("X-Domain-Id"), identity.TenantID); domainID != "" {
				ctx = context.WithValue(ctx, domainIDKey, domainID)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserID extracts the authenticated user ID from a request context.
func UserID(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

// DomainID extracts the domain ID from a request context (set from X-Domain-ID header).
func DomainID(ctx context.Context) string {
	v, _ := ctx.Value(domainIDKey).(string)
	return v
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"error":%q}`, msg)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

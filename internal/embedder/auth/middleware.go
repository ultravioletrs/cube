// Package auth provides request authentication by delegating token validation
// to the SuperMQ auth gRPC service (Authenticate RPC at auth:8181).
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	grpcAuthV1 "github.com/absmach/supermq/api/grpc/auth/v1"
	authgrpc "github.com/absmach/supermq/auth/api/grpc/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type contextKey string

const userIDKey contextKey = "user_id"

// Authenticator wraps the SuperMQ AuthService gRPC client.
type Authenticator struct {
	client  grpcAuthV1.AuthServiceClient
	timeout time.Duration
}

// NewAuthenticator dials the SuperMQ auth service and returns an Authenticator.
// addr should be host:port, e.g. "auth:8181".
func NewAuthenticator(addr string) (*Authenticator, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("dial auth gRPC %s: %w", addr, err)
	}
	timeout := 15 * time.Second
	return &Authenticator{
		client:  authgrpc.NewAuthClient(conn, timeout),
		timeout: timeout,
	}, conn, nil
}

// Identify authenticates the token via the SuperMQ Auth gRPC API and
// returns the user ID on success.
func (a *Authenticator) Identify(ctx context.Context, token string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	res, err := a.client.Authenticate(ctx, &grpcAuthV1.AuthNReq{Token: token})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.Unauthenticated, codes.NotFound, codes.PermissionDenied:
				return "", fmt.Errorf("unauthenticated: %w", err)
			}
		}
		return "", fmt.Errorf("auth authenticate: %w", err)
	}

	userID := res.GetUserId()
	if userID == "" {
		userID = res.GetId()
	}
	if userID == "" {
		return "", fmt.Errorf("auth authenticate returned empty user id")
	}
	return userID, nil
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

			userID, err := auth.Identify(r.Context(), raw)
			if err != nil {
				if strings.Contains(err.Error(), "unauthenticated:") {
					writeError(w, http.StatusUnauthorized, "invalid or expired token")
					return
				}
				writeError(w, http.StatusServiceUnavailable, "authentication service unavailable")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserID extracts the authenticated user ID from a request context.
func UserID(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"error":%q}`, msg)
}

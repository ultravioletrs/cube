// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package cubeauth

import "context"

type contextKey string

const SessionKey contextKey = "cube_session"

type Session struct {
	EntityID  string `json:"entityId"`
	TenantID  string `json:"tenantId,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	Token     string `json:"-"`
	ExpiresAt string `json:"expiresAt,omitempty"`
}

func WithSession(ctx context.Context, session Session) context.Context {
	return context.WithValue(ctx, SessionKey, session)
}

func SessionFromContext(ctx context.Context) (Session, bool) {
	session, ok := ctx.Value(SessionKey).(Session)
	return session, ok
}

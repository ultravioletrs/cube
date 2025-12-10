// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"

	"github.com/absmach/supermq/pkg/authn"
)

type Service interface {
	ProxyRequest(ctx context.Context, session *authn.Session, path string) error
	Secure() string
	UpdateAttestationPolicy(ctx context.Context, session *authn.Session, policy []byte) error
	GetAttestationPolicy(ctx context.Context, session *authn.Session) ([]byte, error)
}

type Repository interface {
	UpdateAttestationPolicy(ctx context.Context, policy []byte) error
	GetAttestationPolicy(ctx context.Context) ([]byte, error)
}

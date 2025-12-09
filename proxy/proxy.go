// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"
	"net/http/httputil"
)

type Service interface {
	Proxy() *httputil.ReverseProxy
	Secure() string
	UpdateAttestationPolicy(ctx context.Context, policy []byte) error
	GetAttestationPolicy(ctx context.Context) ([]byte, error)
}

type Repository interface {
	UpdateAttestationPolicy(ctx context.Context, policy []byte) error
	GetAttestationPolicy(ctx context.Context) ([]byte, error)
}

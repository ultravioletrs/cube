// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package proxy

import (
	"context"

	"github.com/absmach/supermq/pkg/authn"
)

type Service interface {
	Identify(ctx context.Context, token string) error
}

type service struct {
	auth authn.Authentication
}

func NewService(auth authn.Authentication) Service {
	return &service{
		auth: auth,
	}
}

func (s *service) Identify(ctx context.Context, token string) error {
	_, err := s.auth.Authenticate(ctx, token)

	return err
}

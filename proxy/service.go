package proxy

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth/api/grpc"
)

type Service interface {
	Identify(ctx context.Context, token string) error
}

type service struct {
	auth grpc.AuthServiceClient
}

func NewService(authClient grpc.AuthServiceClient) Service {
	return &service{
		auth: authClient,
	}
}

func (s *service) Identify(ctx context.Context, token string) error {
	_, err := s.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})

	return err
}

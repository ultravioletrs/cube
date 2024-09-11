package main

import (
	"context"
	"strings"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/mproxy/pkg/session"
)

var _ session.Handler = (*handler)(nil)

var errClientNotInitialized = errors.New("client is not initialized")

type handler struct {
	auth magistrala.AuthnServiceClient
}

func NewHandler(authClient magistrala.AuthnServiceClient) session.Handler {
	return &handler{
		auth: authClient,
	}
}

func (h *handler) AuthConnect(ctx context.Context) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return errClientNotInitialized
	}

	token, err := extractBearerToken(string(s.Password))
	if err != nil {
		return errors.Wrap(apiutil.ErrValidation, err)
	}

	if _, err := h.auth.Identify(ctx, &magistrala.IdentityReq{Token: token}); err != nil {
		return err
	}

	return nil
}

func (h *handler) AuthPublish(ctx context.Context, topic *string, payload *[]byte) error {
	return nil
}

func (h *handler) AuthSubscribe(ctx context.Context, topics *[]string) error {
	return nil
}

func (h *handler) Connect(ctx context.Context) error {
	return nil
}

func (h *handler) Publish(ctx context.Context, topic *string, payload *[]byte) error {
	return nil
}

func (h *handler) Subscribe(ctx context.Context, topics *[]string) error {
	return nil
}

func (h *handler) Unsubscribe(ctx context.Context, topics *[]string) error {
	return nil
}

func (h *handler) Disconnect(ctx context.Context) error {
	return nil
}

func extractBearerToken(topic string) (string, error) {
	if topic == "" {
		return "", apiutil.ErrBearerKey
	}

	if !strings.HasPrefix(topic, "Bearer ") {
		return "", apiutil.ErrBearerKey
	}

	token := strings.TrimPrefix(topic, "Bearer ")
	if token == "" {
		return "", apiutil.ErrBearerKey
	}

	return token, nil
}

package api

import (
	"context"
	"io"
	"net/http"

	mgauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/ultraviolet/cube/proxy/endpoint"
)

func decodeGetAttestationPolicyRequest(ctx context.Context, _ *http.Request) (interface{}, error) {
	session, ok := ctx.Value(mgauthn.SessionKey).(mgauthn.Session)
	if !ok {
		return nil, errUnauthorized
	}

	return endpoint.GetAttestationPolicyRequest{
		Session: &session,
	}, nil
}

func encodeGetAttestationPolicyResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(endpoint.GetAttestationPolicyResponse)
	w.Header().Set("Content-Type", ContentType)
	_, err := w.Write(resp.Policy)
	return err
}

func decodeUpdateAttestationPolicyRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	session, ok := ctx.Value(mgauthn.SessionKey).(mgauthn.Session)
	if !ok {
		return nil, errUnauthorized
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return endpoint.UpdateAttestationPolicyRequest{
		Session: &session,
		Policy:  body,
	}, nil
}

func encodeUpdateAttestationPolicyResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.WriteHeader(http.StatusCreated)
	return nil
}

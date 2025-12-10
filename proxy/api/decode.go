// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"errors"
	"io"
	"net/http"

	mgauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/ultravioletrs/cube/proxy/endpoint"
)

var errInvalidRequestType = errors.New("invalid request type")

func decodeGetAttestationPolicyRequest(ctx context.Context, _ *http.Request) (any, error) {
	session, ok := ctx.Value(mgauthn.SessionKey).(mgauthn.Session)
	if !ok {
		return nil, errUnauthorized
	}

	return endpoint.GetAttestationPolicyRequest{
		Session: &session,
	}, nil
}

func encodeGetAttestationPolicyResponse(_ context.Context, w http.ResponseWriter, response any) error {
	resp, ok := response.(endpoint.GetAttestationPolicyResponse)
	if !ok {
		return errInvalidRequestType
	}

	w.Header().Set("Content-Type", ContentType)
	_, err := w.Write(resp.Policy)

	return err
}

func decodeUpdateAttestationPolicyRequest(ctx context.Context, r *http.Request) (any, error) {
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

func encodeUpdateAttestationPolicyResponse(_ context.Context, w http.ResponseWriter, _ any) error {
	w.WriteHeader(http.StatusCreated)

	return nil
}

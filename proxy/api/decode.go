package api

import (
	"context"
	"io"
	"net/http"
)

type getAttestationPolicyRequest struct{}

type getAttestationPolicyResponse struct {
	Policy []byte `json:"policy"`
}

type updateAttestationPolicyRequest struct {
	Policy []byte `json:"policy"`
}

type updateAttestationPolicyResponse struct{}

func decodeGetAttestationPolicyRequest(_ context.Context, _ *http.Request) (interface{}, error) {
	return getAttestationPolicyRequest{}, nil
}

func encodeGetAttestationPolicyResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(getAttestationPolicyResponse)
	w.Header().Set("Content-Type", ContentType)
	_, err := w.Write(resp.Policy)
	return err
}

func decodeUpdateAttestationPolicyRequest(_ context.Context, r *http.Request) (interface{}, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return updateAttestationPolicyRequest{Policy: body}, nil
}

func encodeUpdateAttestationPolicyResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.WriteHeader(http.StatusCreated)
	return nil
}

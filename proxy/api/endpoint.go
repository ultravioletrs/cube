package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/ultraviolet/cube/proxy"
)

func makeGetAttestationPolicyEndpoint(svc proxy.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		policy, err := svc.GetAttestationPolicy(ctx)
		if err != nil {
			return nil, err
		}
		return getAttestationPolicyResponse{Policy: policy}, nil
	}
}

func makeUpdateAttestationPolicyEndpoint(svc proxy.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateAttestationPolicyRequest)
		err := svc.UpdateAttestationPolicy(ctx, req.Policy)
		if err != nil {
			return nil, err
		}
		return updateAttestationPolicyResponse{}, nil
	}
}

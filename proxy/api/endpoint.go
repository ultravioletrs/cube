package api

import (
	"context"

	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/go-kit/kit/endpoint"
	proxy "github.com/ultraviolet/vault-proxy"
)

func identifyEndpoint(svc proxy.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(identifyRequest)
		if err := req.Validate(); err != nil {
			return identifyResponse{identified: false}, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.Identify(ctx, req.Token); err != nil {
			return identifyResponse{identified: false}, err
		}

		return identifyResponse{identified: true}, nil
	}
}

// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package api

import (
	"context"

	"github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/go-kit/kit/endpoint"
	"github.com/ultraviolet/cube/proxy"
)

func identifyEndpoint(svc proxy.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req, ok := request.(identifyRequest)
		if !ok {
			return identifyResponse{identified: false}, errors.New("invalid request type")
		}

		if err := req.Validate(); err != nil {
			return identifyResponse{identified: false}, errors.Wrap(util.ErrValidation, err)
		}

		if err := svc.Identify(ctx, req.Token); err != nil {
			return identifyResponse{identified: false}, err
		}

		return identifyResponse{identified: true}, nil
	}
}

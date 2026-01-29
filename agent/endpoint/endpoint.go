// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package endpoint

import (
	"context"
	"encoding/base64"
	"errors"

	"github.com/go-kit/kit/endpoint"
	"github.com/ultravioletrs/cocos/pkg/attestation/quoteprovider"
	"github.com/ultravioletrs/cocos/pkg/attestation/vtpm"
	"github.com/ultravioletrs/cube/agent"
)

var (
	errInvalidRequestType = errors.New("invalid request type")
	errInvalidReportData  = errors.New("invalid report_data: must be base64-encoded 64 bytes")
	errInvalidNonce       = errors.New("invalid nonce: must be base64-encoded 32 bytes")
)

type Endpoints struct {
	Attestation endpoint.Endpoint
}

func MakeEndpoints(s agent.Service) Endpoints {
	return Endpoints{
		Attestation: MakeAttestationEndpoint(s),
	}
}

type AttestationRequest struct {
	ReportData string `json:"report_data"`
	Nonce      string `json:"nonce"`
	ToJSON     bool   `json:"to_json"`
}

type AttestationResponse struct {
	Report []byte `json:"report,omitempty"`
	Err    error  `json:"err,omitempty"`
}

func (r AttestationResponse) Failed() error {
	return r.Err
}

func MakeAttestationEndpoint(s agent.Service) endpoint.Endpoint {
	return func(_ context.Context, request any) (any, error) {
		req, ok := request.(AttestationRequest)
		if !ok {
			return nil, errInvalidRequestType
		}

		// Parse and validate report data
		reportDataBytes, err := base64.StdEncoding.DecodeString(req.ReportData)
		if err != nil || len(reportDataBytes) > quoteprovider.Nonce {
			return nil, errInvalidReportData
		}

		var reportData [quoteprovider.Nonce]byte
		copy(reportData[:], reportDataBytes)

		// Parse and validate nonce
		nonceBytes, err := base64.StdEncoding.DecodeString(req.Nonce)
		if err != nil || len(nonceBytes) > vtpm.Nonce {
			return nil, errInvalidNonce
		}

		var nonce [vtpm.Nonce]byte
		copy(nonce[:], nonceBytes)

		// Call service method
		report, err := s.Attestation(reportData, nonce, req.ToJSON)
		if err != nil {
			return nil, err
		}

		return AttestationResponse{Report: report}, nil
	}
}

// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package endpoint

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/go-kit/kit/endpoint"
	"github.com/ultraviolet/cube/agent"
	"github.com/ultravioletrs/cocos/pkg/attestation"
	"github.com/ultravioletrs/cocos/pkg/attestation/quoteprovider"
	"github.com/ultravioletrs/cocos/pkg/attestation/vtpm"
)

var (
	errInvalidRequestType     = errors.New("invalid request type")
	errInvalidReportData      = errors.New("invalid report_data: must be base64-encoded 64 bytes")
	errInvalidNonce           = errors.New("invalid nonce: must be base64-encoded 32 bytes")
	errInvalidAttestationType = errors.New("invalid attestation_type: must be one of snp, tdx, vtpm, snpvtpm")
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
	ReportData      string `json:"report_data"`
	Nonce           string `json:"nonce"`
	AttestationType string `json:"attestation_type"`
	ToJSON          bool   `json:"to_json"`
}

type AttestationResponse struct {
	Report []byte `json:"report,omitempty"`
	Err    error  `json:"error,omitempty"`
}

func (r AttestationResponse) Failed() error {
	return r.Err
}

func MakeAttestationEndpoint(s agent.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(AttestationRequest)
		if !ok {
			return AttestationResponse{Err: errInvalidRequestType}, nil
		}

		// Parse and validate report data
		reportDataBytes, err := base64.StdEncoding.DecodeString(req.ReportData)
		if err != nil || len(reportDataBytes) > quoteprovider.Nonce {
			return AttestationResponse{Err: errInvalidReportData}, nil
		}
		var reportData [quoteprovider.Nonce]byte
		copy(reportData[:], reportDataBytes)

		// Parse and validate nonce
		nonceBytes, err := base64.StdEncoding.DecodeString(req.Nonce)
		if err != nil || len(nonceBytes) > vtpm.Nonce {
			return AttestationResponse{Err: errInvalidNonce}, nil
		}
		var nonce [vtpm.Nonce]byte
		copy(nonce[:], nonceBytes)

		// Parse attestation type
		attType, err := parseAttestationType(req.AttestationType)
		if err != nil {
			return AttestationResponse{Err: err}, nil
		}

		// Call service method
		report, err := s.Attestation(reportData, nonce, attType, req.ToJSON)
		if err != nil {
			return AttestationResponse{Err: err}, nil
		}

		return AttestationResponse{Report: report}, nil
	}
}

func parseAttestationType(attType string) (attestation.PlatformType, error) {
	switch attType {
	case "snp":
		return attestation.SNP, nil
	case "tdx":
		return attestation.TDX, nil
	case "vtpm":
		return attestation.VTPM, nil
	case "snpvtpm":
		return attestation.SNPvTPM, nil
	default:
		return attestation.NoCC, fmt.Errorf("%w: %s", errInvalidAttestationType, attType)
	}
}

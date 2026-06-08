// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ultravioletrs/cube/internal/cubeauth"
	atomv1 "github.com/ultravioletrs/cube/proto/atom/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const DefaultTimeout = 15 * time.Second

var (
	errAuthorizationDenied = errors.New("authorization denied")
	errEmptyCertificate    = errors.New("atom issued empty certificate")
	errGraphQLFailed       = errors.New("atom graphql failed")
)

type Client struct {
	graphQLURL string
	http       *http.Client
	timeout    time.Duration
	conn       *grpc.ClientConn
	auth       atomv1.AuthServiceClient
	authz      atomv1.AuthzServiceClient
	certs      atomv1.CertificateServiceClient
}

type CheckRequest struct {
	SubjectID  string
	Action     string
	ObjectKind string
	ObjectID   string
	ResourceID string
	Context    map[string]string
}

type IssuedCertificate struct {
	Certificate struct {
		CredentialID    string `json:"credentialId"`
		SerialNumber    string `json:"serialNumber"`
		CertificatePEM  string `json:"certificatePem"`
		ExpiresAt       string `json:"expiresAt"`
		FingerprintSHA2 string `json:"fingerprintSha256"`
	} `json:"certificate"`
}

func NewClient(grpcAddr, graphQLURL string, timeout time.Duration) (*Client, error) {
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	if grpcAddr == "" {
		grpcAddr = "atom:8081"
	}

	if graphQLURL == "" {
		graphQLURL = "http://atom:8080/graphql"
	}

	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial atom gRPC %s: %w", grpcAddr, err)
	}

	return &Client{
		graphQLURL: graphQLURL,
		http:       &http.Client{Timeout: timeout},
		timeout:    timeout,
		conn:       conn,
		auth:       atomv1.NewAuthServiceClient(conn),
		authz:      atomv1.NewAuthzServiceClient(conn),
		certs:      atomv1.NewCertificateServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}

	return c.conn.Close()
}

func (c *Client) Authenticate(ctx context.Context, token string) (cubeauth.Session, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.auth.Authenticate(ctx, &atomv1.AuthenticateRequest{Token: token})
	if err != nil {
		return cubeauth.Session{}, err
	}

	return cubeauth.Session{
		EntityID:  resp.GetEntityId(),
		TenantID:  resp.GetTenantId(),
		SessionID: resp.GetSessionId(),
		Token:     token,
		ExpiresAt: jwtExpiresAt(token),
	}, nil
}

func (c *Client) Check(ctx context.Context, callerToken string, req *CheckRequest) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+callerToken)

	resp, err := c.authz.Check(ctx, &atomv1.CheckRequest{
		SubjectId:  req.SubjectID,
		Action:     req.Action,
		ResourceId: req.ResourceID,
		Context:    req.Context,
		ObjectKind: req.ObjectKind,
		ObjectId:   req.ObjectID,
	})
	if err != nil {
		return err
	}

	if !resp.GetAllowed() {
		if reason := resp.GetReason(); reason != "" {
			return fmt.Errorf("%w: %s", errAuthorizationDenied, reason)
		}

		return errAuthorizationDenied
	}

	return nil
}

func (c *Client) ResolveCertificate(
	ctx context.Context, callerToken, serialNumber, fingerprintSHA256 string,
) (*atomv1.ResolveCertificateResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+callerToken)

	return c.certs.ResolveCertificate(ctx, &atomv1.ResolveCertificateRequest{
		SerialNumber:      serialNumber,
		FingerprintSha256: fingerprintSHA256,
	})
}

func (c *Client) IssueCertificateFromCSR(
	ctx context.Context, token, entityID, csrPEM string, ttlSeconds uint64,
) (IssuedCertificate, error) {
	const query = `mutation IssueCertificateFromCsr($input: IssueCertificateFromCsrInput!) {
  issueCertificateFromCsr(input: $input) {
    certificate {
      credentialId
      serialNumber
      certificatePem
      expiresAt
      fingerprintSha256
    }
  }
}`

	input := map[string]any{
		"entityId": entityID,
		"csrPem":   csrPEM,
	}
	if ttlSeconds > 0 {
		input["ttlSecs"] = ttlSeconds
	}

	var payload struct {
		Data struct {
			IssueCertificateFromCSR IssuedCertificate `json:"issueCertificateFromCsr"`
		} `json:"data"`
		Errors []graphQLError `json:"errors"`
	}
	if err := c.doGraphQL(ctx, token, query, map[string]any{"input": input}, &payload); err != nil {
		return IssuedCertificate{}, err
	}

	if len(payload.Errors) > 0 {
		return IssuedCertificate{}, payload.Errors[0]
	}

	if payload.Data.IssueCertificateFromCSR.Certificate.CertificatePEM == "" {
		return IssuedCertificate{}, errEmptyCertificate
	}

	return payload.Data.IssueCertificateFromCSR, nil
}

func (c *Client) doGraphQL(
	ctx context.Context, token string, query string, variables map[string]any, out any,
) error {
	body, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.graphQLURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%w: status=%d body=%s", errGraphQLFailed, resp.StatusCode, string(data))
	}

	return json.Unmarshal(data, out)
}

type graphQLError struct {
	Message string `json:"message"`
}

func (e graphQLError) Error() string {
	if e.Message == "" {
		return "atom graphql error"
	}

	return e.Message
}

func jwtExpiresAt(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return ""
	}

	return time.Unix(claims.Exp, 0).UTC().Format(time.RFC3339)
}

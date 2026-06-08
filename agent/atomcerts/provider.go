// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package atomcerts

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/ultravioletrs/cocos/pkg/atls"
	"github.com/ultravioletrs/cocos/pkg/attestation"
	"github.com/ultravioletrs/cube/internal/atom"
)

const (
	nonceLength = 64
	nonceSuffix = ".nonce"
)

type Provider struct {
	attestationProvider atls.AttestationProvider
	atomClient          *atom.Client
	token               string
	entityID            string
	ttl                 time.Duration
	subject             atls.CertificateSubject
}

func NewProvider(
	provider attestation.Provider,
	platformType attestation.PlatformType,
	atomClient *atom.Client,
	token string,
	entityID string,
	ttl time.Duration,
) (atls.CertificateProvider, error) {
	attestationProvider, err := atls.NewAttestationProvider(provider, platformType)
	if err != nil {
		return nil, fmt.Errorf("create attestation provider: %w", err)
	}
	if atomClient == nil {
		return nil, fmt.Errorf("atom client is required")
	}
	if token == "" {
		return nil, fmt.Errorf("atom token is required")
	}
	if entityID == "" {
		return nil, fmt.Errorf("atom entity id is required")
	}
	if ttl <= 0 {
		ttl = 365 * 24 * time.Hour
	}

	return &Provider{
		attestationProvider: attestationProvider,
		atomClient:          atomClient,
		token:               token,
		entityID:            entityID,
		ttl:                 ttl,
		subject:             atls.DefaultCertificateSubject(),
	}, nil
}

func (p *Provider) GetCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}

	pubKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}

	nonce, err := extractNonceFromSNI(clientHello.ServerName)
	if err != nil {
		return nil, fmt.Errorf("extract nonce: %w", err)
	}

	attestationData, err := p.attestationProvider.Attest(pubKeyDER, nonce)
	if err != nil {
		return nil, fmt.Errorf("attest public key: %w", err)
	}

	csrPEM, err := p.createCSR(privateKey, pkix.Extension{
		Id:    p.attestationProvider.OID(),
		Value: attestationData,
	})
	if err != nil {
		return nil, err
	}

	ttlSeconds := uint64(p.ttl.Seconds())
	issued, err := p.atomClient.IssueCertificateFromCSR(
		clientHello.Context(), p.token, p.entityID, string(csrPEM), ttlSeconds,
	)
	if err != nil {
		return nil, fmt.Errorf("issue atom certificate from csr: %w", err)
	}

	certDERs, err := parseCertificateChain(issued.Certificate.CertificatePEM)
	if err != nil {
		return nil, err
	}

	return &tls.Certificate{
		Certificate: certDERs,
		PrivateKey:  privateKey,
	}, nil
}

func (p *Provider) createCSR(privateKey *ecdsa.PrivateKey, extension pkix.Extension) ([]byte, error) {
	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			Organization:  []string{p.subject.Organization},
			CommonName:    p.subject.CommonName,
			Country:       []string{p.subject.Country},
			Province:      []string{p.subject.Province},
			Locality:      []string{p.subject.Locality},
			StreetAddress: []string{p.subject.StreetAddress},
			PostalCode:    []string{p.subject.PostalCode},
		},
		ExtraExtensions: []pkix.Extension{extension},
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, privateKey)
	if err != nil {
		return nil, fmt.Errorf("create csr: %w", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}), nil
}

func extractNonceFromSNI(serverName string) ([]byte, error) {
	if len(serverName) < len(nonceSuffix) || serverName[len(serverName)-len(nonceSuffix):] != nonceSuffix {
		return nil, fmt.Errorf("invalid server name: %s", serverName)
	}

	nonceStr := serverName[:len(serverName)-len(nonceSuffix)]
	nonce, err := hex.DecodeString(nonceStr)
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}
	if len(nonce) != nonceLength {
		return nil, fmt.Errorf("invalid nonce length: expected %d bytes, got %d bytes", nonceLength, len(nonce))
	}

	return nonce, nil
}

func parseCertificateChain(certificatePEM string) ([][]byte, error) {
	var certs [][]byte
	rest := []byte(certificatePEM)
	for {
		block, remaining := pem.Decode(rest)
		if block == nil {
			break
		}
		rest = remaining
		if block.Type == "CERTIFICATE" {
			certs = append(certs, block.Bytes)
		}
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("atom returned no certificate PEM block")
	}

	return certs, nil
}

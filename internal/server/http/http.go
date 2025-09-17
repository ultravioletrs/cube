// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/absmach/supermq/pkg/server"
	cubeServer "github.com/ultraviolet/cube/internal/server"
	"github.com/ultravioletrs/cocos/pkg/atls"
)

const (
	httpProtocol  = "http"
	httpsProtocol = "https"
)

var (
	errAppendServerCA = errors.New("failed to append server ca to tls.Config")
	errAppendClientCA = errors.New("failed to append client ca to tls.Config")
)

type tlsSetupResult struct {
	config *tls.Config
	mtls   bool
}

type httpServer struct {
	cubeServer.BaseServer

	server *http.Server
	caURL  string
}

var _ server.Server = (*httpServer)(nil)

func NewServer(
	ctx context.Context, cancel context.CancelFunc, name string, config *cubeServer.Config,
	handler http.Handler, logger *slog.Logger, caURL string,
) server.Server {
	baseServer := cubeServer.NewBaseServer(ctx, cancel, name, config, logger)
	hserver := &http.Server{Addr: baseServer.Address, Handler: handler}

	return &httpServer{
		BaseServer: baseServer,
		server:     hserver,
		caURL:      caURL,
	}
}

func (s *httpServer) Start() error {
	s.Protocol = httpProtocol

	if s.shouldUseAttestedTLS() {
		return s.startWithAttestedTLS()
	}

	if s.shouldUseRegularTLS() {
		return s.startWithRegularTLS()
	}

	return s.startWithoutTLS()
}

func (s *httpServer) Stop() error {
	defer s.Cancel()

	ctx, cancel := context.WithTimeout(context.Background(), server.StopWaitTime)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		s.Logger.Error(fmt.Sprintf(
			"%s service %s server error occurred during shutdown at %s: %s", s.Name, s.Protocol, s.Address, err))

		return fmt.Errorf("%s service %s server error occurred during shutdown at %s: %w", s.Name, s.Protocol, s.Address, err)
	}

	s.Logger.Info(fmt.Sprintf("%s %s service shutdown of http at %s", s.Name, s.Protocol, s.Address))

	return nil
}

func (s *httpServer) shouldUseAttestedTLS() bool {
	return s.Config.AttestedTLS && s.caURL != ""
}

func (s *httpServer) shouldUseRegularTLS() bool {
	return s.Config.CertFile != "" || s.Config.KeyFile != ""
}

func (s *httpServer) startWithAttestedTLS() error {
	tlsConfig, err := s.setupAttestedTLS()
	if err != nil {
		return err
	}

	s.server.TLSConfig = tlsConfig.config
	s.Protocol = httpsProtocol

	s.logAttestedTLSStart(tlsConfig.mtls)

	return s.listenAndServe(true)
}

func (s *httpServer) startWithRegularTLS() error {
	tlsConfig, err := s.setupRegularTLS()
	if err != nil {
		return err
	}

	s.server.TLSConfig = tlsConfig.config
	s.Protocol = httpsProtocol

	s.logRegularTLSStart(tlsConfig.mtls)

	return s.listenAndServe(true)
}

func (s *httpServer) startWithoutTLS() error {
	s.Logger.Info(fmt.Sprintf("%s service %s server listening at %s without TLS", s.Name, s.Protocol, s.Address))

	return s.listenAndServe(false)
}

func (s *httpServer) setupAttestedTLS() (*tlsSetupResult, error) {
	tlsConfig := &tls.Config{
		ClientAuth:     tls.NoClientCert,
		GetCertificate: atls.GetCertificate(s.caURL, ""),
	}

	mtls, err := s.configureCertificateAuthorities(tlsConfig)
	if err != nil {
		return nil, err
	}

	return &tlsSetupResult{config: tlsConfig, mtls: mtls}, nil
}

func (s *httpServer) setupRegularTLS() (*tlsSetupResult, error) {
	certificate, err := loadX509KeyPair(s.Config.CertFile, s.Config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load auth certificates: %w", err)
	}

	tlsConfig := &tls.Config{
		ClientAuth:   tls.NoClientCert,
		Certificates: []tls.Certificate{certificate},
	}

	mtls, err := s.configureCertificateAuthorities(tlsConfig)
	if err != nil {
		return nil, err
	}

	if mtls {
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return &tlsSetupResult{config: tlsConfig, mtls: mtls}, nil
}

func (s *httpServer) configureCertificateAuthorities(tlsConfig *tls.Config) (bool, error) {
	var mtls bool

	// Configure root CA
	if err := s.configureRootCA(tlsConfig); err != nil {
		return false, err
	}

	// Configure client CA
	hasClientCA, err := s.configureClientCA(tlsConfig)
	if err != nil {
		return false, err
	}

	if hasClientCA {
		mtls = true
	}

	return mtls, nil
}

func (s *httpServer) configureRootCA(tlsConfig *tls.Config) error {
	rootCA, err := loadCertFile(s.Config.ServerCAFile)
	if err != nil {
		return fmt.Errorf("failed to load server ca file: %w", err)
	}

	if len(rootCA) == 0 {
		return nil
	}

	if tlsConfig.RootCAs == nil {
		tlsConfig.RootCAs = x509.NewCertPool()
	}

	if !tlsConfig.RootCAs.AppendCertsFromPEM(rootCA) {
		return errAppendServerCA
	}

	return nil
}

func (s *httpServer) configureClientCA(tlsConfig *tls.Config) (bool, error) {
	clientCA, err := loadCertFile(s.Config.ClientCAFile)
	if err != nil {
		return false, fmt.Errorf("failed to load client ca file: %w", err)
	}

	if len(clientCA) == 0 {
		return false, nil
	}

	if tlsConfig.ClientCAs == nil {
		tlsConfig.ClientCAs = x509.NewCertPool()
	}

	if !tlsConfig.ClientCAs.AppendCertsFromPEM(clientCA) {
		return false, errAppendClientCA
	}

	return true, nil
}

func (s *httpServer) logAttestedTLSStart(mtls bool) {
	if mtls {
		s.Logger.Info(fmt.Sprintf("%s service %s server listening at %s with Attested mTLS", s.Name, s.Protocol, s.Address))
	} else {
		s.Logger.Info(fmt.Sprintf("%s service %s server listening at %s with Attested TLS", s.Name, s.Protocol, s.Address))
	}
}

func (s *httpServer) logRegularTLSStart(mtls bool) {
	if mtls {
		s.Logger.Info(fmt.Sprintf(
			"%s service %s server listening at %s with TLS/mTLS cert %s , key %s and CAs %s, %s",
			s.Name, s.Protocol, s.Address, s.Config.CertFile, s.Config.KeyFile,
			s.Config.ServerCAFile, s.Config.ClientCAFile))
	} else {
		s.Logger.Info(
			fmt.Sprintf("%s service %s server listening at %s with TLS cert %s and key %s",
				s.Name, s.Protocol, s.Address, s.Config.CertFile, s.Config.KeyFile))
	}
}

func (s *httpServer) listenAndServe(useTLS bool) error {
	errCh := make(chan error, 1)

	go func() {
		if useTLS {
			errCh <- s.server.ListenAndServeTLS("", "")
		} else {
			errCh <- s.server.ListenAndServe()
		}
	}()

	select {
	case <-s.Ctx.Done():
		return s.Stop()
	case err := <-errCh:
		return err
	}
}

func loadCertFile(certFile string) ([]byte, error) {
	if certFile != "" {
		return readFileOrData(certFile)
	}

	return []byte{}, nil
}

func readFileOrData(input string) ([]byte, error) {
	if len(input) < 1000 && !strings.Contains(input, "\n") {
		data, err := os.ReadFile(input)
		if err == nil {
			return data, nil
		}

		return nil, err
	}

	return []byte(input), nil
}

func loadX509KeyPair(certfile, keyfile string) (tls.Certificate, error) {
	cert, err := readFileOrData(certfile)
	if err != nil {
		return tls.Certificate{}, err
	}

	key, err := readFileOrData(keyfile)
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.X509KeyPair(cert, key)
}

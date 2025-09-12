// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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

type httpServer struct {
	cubeServer.BaseServer
	server *http.Server
	caUrl  string
}

var _ server.Server = (*httpServer)(nil)

func NewServer(ctx context.Context, cancel context.CancelFunc, name string, config cubeServer.Config, handler http.Handler, logger *slog.Logger, caUrl string) server.Server {
	baseServer := cubeServer.NewBaseServer(ctx, cancel, name, config, logger)
	hserver := &http.Server{Addr: baseServer.Address, Handler: handler}

	return &httpServer{
		BaseServer: baseServer,
		server:     hserver,
		caUrl:      caUrl,
	}
}

func (s *httpServer) Start() error {
	errCh := make(chan error)
	s.Protocol = httpProtocol

	// Check if this is an Agent config with AttestedTLS enabled
	if s.Config.AttestedTLS && s.caUrl != "" {
		tlsConfig := &tls.Config{
			ClientAuth:     tls.NoClientCert,
			GetCertificate: atls.GetCertificate(s.caUrl, ""),
		}

		var mtls bool
		mtls = false

		// Loading Server CA file
		rootCA, err := loadCertFile(s.Config.ServerCAFile)
		if err != nil {
			return fmt.Errorf("failed to load server ca file: %w", err)
		}
		if len(rootCA) > 0 {
			if tlsConfig.RootCAs == nil {
				tlsConfig.RootCAs = x509.NewCertPool()
			}
			if !tlsConfig.RootCAs.AppendCertsFromPEM(rootCA) {
				return fmt.Errorf("failed to append server ca to tls.Config")
			}
			mtls = true
		}

		// Loading Client CA File
		clientCA, err := loadCertFile(s.Config.ClientCAFile)
		if err != nil {
			return fmt.Errorf("failed to load client ca file: %w", err)
		}
		if len(clientCA) > 0 {
			if tlsConfig.ClientCAs == nil {
				tlsConfig.ClientCAs = x509.NewCertPool()
			}
			if !tlsConfig.ClientCAs.AppendCertsFromPEM(clientCA) {
				return fmt.Errorf("failed to append client ca to tls.Config")
			}
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
			mtls = true
		}

		s.server.TLSConfig = tlsConfig
		s.Protocol = httpsProtocol

		if mtls {
			s.Logger.Info(fmt.Sprintf("%s service %s server listening at %s with Attested mTLS", s.Name, s.Protocol, s.Address))
		} else {
			s.Logger.Info(fmt.Sprintf("%s service %s server listening at %s with Attested TLS", s.Name, s.Protocol, s.Address))
		}

		go func() {
			errCh <- s.server.ListenAndServeTLS("", "")
		}()
	} else {
		// Handle regular TLS and non-TLS cases
		switch {
		case s.Config.CertFile != "" || s.Config.KeyFile != "":
			certificate, err := loadX509KeyPair(s.Config.CertFile, s.Config.KeyFile)
			if err != nil {
				return fmt.Errorf("failed to load auth certificates: %w", err)
			}

			tlsConfig := &tls.Config{
				ClientAuth:   tls.NoClientCert,
				Certificates: []tls.Certificate{certificate},
			}

			var mtlsCA string
			// Loading Server CA file
			rootCA, err := loadCertFile(s.Config.ServerCAFile)
			if err != nil {
				return fmt.Errorf("failed to load root ca file: %w", err)
			}
			if len(rootCA) > 0 {
				if tlsConfig.RootCAs == nil {
					tlsConfig.RootCAs = x509.NewCertPool()
				}
				if !tlsConfig.RootCAs.AppendCertsFromPEM(rootCA) {
					return fmt.Errorf("failed to append root ca to tls.Config")
				}
				mtlsCA = fmt.Sprintf("root ca %s", s.Config.ServerCAFile)
			}

			// Loading Client CA File
			clientCA, err := loadCertFile(s.Config.ClientCAFile)
			if err != nil {
				return fmt.Errorf("failed to load client ca file: %w", err)
			}
			if len(clientCA) > 0 {
				if tlsConfig.ClientCAs == nil {
					tlsConfig.ClientCAs = x509.NewCertPool()
				}
				if !tlsConfig.ClientCAs.AppendCertsFromPEM(clientCA) {
					return fmt.Errorf("failed to append client ca to tls.Config")
				}
				mtlsCA = fmt.Sprintf("%s client ca %s", mtlsCA, s.Config.ClientCAFile)
			}

			s.server.TLSConfig = tlsConfig
			s.Protocol = httpsProtocol

			switch {
			case mtlsCA != "":
				tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
				s.Logger.Info(fmt.Sprintf("%s service %s server listening at %s with TLS/mTLS cert %s , key %s and %s", s.Name, s.Protocol, s.Address, s.Config.CertFile, s.Config.KeyFile, mtlsCA))
			default:
				s.Logger.Info(fmt.Sprintf("%s service %s server listening at %s with TLS cert %s and key %s", s.Name, s.Protocol, s.Address, s.Config.CertFile, s.Config.KeyFile))
			}

			go func() {
				errCh <- s.server.ListenAndServeTLS("", "")
			}()
		default:
			s.Logger.Info(fmt.Sprintf("%s service %s server listening at %s without TLS", s.Name, s.Protocol, s.Address))
			go func() {
				errCh <- s.server.ListenAndServe()
			}()
		}
	}

	select {
	case <-s.Ctx.Done():
		return s.Stop()
	case err := <-errCh:
		return err
	}
}

func (s *httpServer) Stop() error {
	defer s.Cancel()
	ctx, cancel := context.WithTimeout(context.Background(), server.StopWaitTime)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		s.Logger.Error(fmt.Sprintf("%s service %s server error occurred during shutdown at %s: %s", s.Name, s.Protocol, s.Address, err))
		return fmt.Errorf("%s service %s server error occurred during shutdown at %s: %w", s.Name, s.Protocol, s.Address, err)
	}
	s.Logger.Info(fmt.Sprintf("%s %s service shutdown of http at %s", s.Name, s.Protocol, s.Address))
	return nil
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
		} else {
			return nil, err
		}
	}
	return []byte(input), nil
}

func loadX509KeyPair(certfile, keyfile string) (tls.Certificate, error) {
	cert, err := readFileOrData(certfile)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to read cert: %v", err)
	}

	key, err := readFileOrData(keyfile)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to read key: %v", err)
	}

	return tls.X509KeyPair(cert, key)
}

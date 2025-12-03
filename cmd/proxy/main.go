// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"

	"github.com/absmach/callhome/pkg/client"
	"github.com/absmach/supermq"
	mglog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/authn/authsvc"
	authzsvc "github.com/absmach/supermq/pkg/authz/authsvc"
	domainsgrpc "github.com/absmach/supermq/pkg/domains/grpcclient"
	"github.com/absmach/supermq/pkg/grpcclient"
	"github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/prometheus"
	"github.com/absmach/supermq/pkg/server"
	"github.com/absmach/supermq/pkg/server/http"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"github.com/ultraviolet/cube/agent/audit"
	"github.com/ultraviolet/cube/proxy"
	"github.com/ultraviolet/cube/proxy/api"
	"github.com/ultraviolet/cube/proxy/middleware"
	"github.com/ultravioletrs/cocos/pkg/clients"
	httpclient "github.com/ultravioletrs/cocos/pkg/clients/http"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "cube_proxy"
	envPrefixHTTP  = "UV_CUBE_PROXY_"
	defSvcHTTPPort = "8900"
	envPrefixAgent = "UV_CUBE_AGENT_"
	envPrefixAuth  = "SMQ_AUTH_GRPC_"
)

type config struct {
	LogLevel      string  `env:"UV_CUBE_PROXY_LOG_LEVEL"   envDefault:"info"`
	TargetURL     string  `env:"UV_CUBE_PROXY_TARGET_URL"  envDefault:"http://ollama:11434"`
	SendTelemetry bool    `env:"SMQ_SEND_TELEMETRY"        envDefault:"true"`
	InstanceID    string  `env:"UV_CUBE_PROXY_INSTANCE_ID" envDefault:""`
	JaegerURL     url.URL `env:"SMQ_JAEGER_URL"            envDefault:"http://localhost:4318/v1/traces"`
	TraceRatio    float64 `env:"SMQ_JAEGER_TRACE_RATIO"    envDefault:"1.0"`
	OpenSearchURL string  `env:"UV_CUBE_OPENSEARCH_URL"    envDefault:"http://opensearch:9200"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger, err := mglog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err.Error())
	}

	var exitCode int
	defer mglog.ExitWithError(&exitCode)

	if cfg.InstanceID == "" {
		if cfg.InstanceID, err = uuid.New().ID(); err != nil {
			logger.Error(fmt.Sprintf("failed to generate instanceID: %s", err))

			exitCode = 1

			return
		}
	}

	// Initialize auth gRPC client
	grpcCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&grpcCfg, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load auth gRPC client configuration : %s", err))

		exitCode = 1

		return
	}

	auth, authnClient, err := authsvc.NewAuthentication(ctx, grpcCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to init auth gRPC client: %s", err))

		exitCode = 1

		return
	}
	defer authnClient.Close()

	logger.Info("AuthN successfully connected to auth gRPC server " + authnClient.Secure())

	domainsAuthz, _, domainsClient, err := domainsgrpc.NewAuthorization(ctx, grpcCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to init domains gRPC client: %s", err))

		exitCode = 1

		return
	}
	defer domainsClient.Close()

	authz, authzClient, err := authzsvc.NewAuthorization(ctx, grpcCfg, domainsAuthz)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to init authz gRPC client: %s", err))

		exitCode = 1

		return
	}
	defer authzClient.Close()

	logger.Info("AuthZ successfully connected to auth gRPC server " + authzClient.Secure())

	tp, err := jaeger.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to init Jaeger: %s", err))

		exitCode = 1

		return
	}

	defer func() {
		err := tp.Shutdown(ctx)
		if err != nil {
			logger.Error(fmt.Sprintf("error shutting down tracer provider: %v", err))
		}
	}()

	tracer := tp.Tracer(svcName)

	agentConfig := clients.AttestedClientConfig{}

	if err := env.ParseWithOptions(&agentConfig, env.Options{Prefix: envPrefixAgent}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s agent client configuration : %s", svcName, err))

		exitCode = 1

		return
	}

	agentClient, err := httpclient.NewClient(&agentConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create agent HTTP client: %s", err))

		exitCode = 1

		return
	}

	svc, err := newService(logger, tracer, &agentConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create service: %s", err))

		exitCode = 1

		return
	}

	svc = middleware.AuthMiddleware(authz)(svc)

	logger.Info(fmt.Sprintf(
		"%s service %s client configured to connect to agent at %s with %s",
		svcName, svc.Secure(), agentConfig.URL, svc.Secure()))

	auditSvc := audit.NewAuditMiddleware(logger, audit.Config{
		ComplianceMode:   true,
		EnablePIIMask:    true,
		EnableTokens:     true,
		SensitiveHeaders: []string{},
	})

	idp := uuid.New()

	authmMiddleware := smqauthn.NewAuthNMiddleware(
		auth, smqauthn.WithAllowUnverifiedUser(true), smqauthn.WithDomainCheck(false),
	)

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))

		exitCode = 1

		return
	}

	httpSvr := http.NewServer(
		ctx, cancel, svcName, httpServerConfig, api.MakeHandler(
			svc, cfg.InstanceID, auditSvc, authmMiddleware, idp, agentClient.Transport(), agentConfig.URL, cfg.OpenSearchURL,
		),
		logger)

	if cfg.SendTelemetry {
		chc := client.New(svcName, supermq.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return httpSvr.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, httpSvr)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Proxy service terminated: %s", err))
	}
}

func newService(
	logger *slog.Logger, tracer trace.Tracer, agentConfig *clients.AttestedClientConfig,
) (proxy.Service, error) {
	svc, err := proxy.New(agentConfig)
	if err != nil {
		return nil, err
	}

	svc = middleware.NewLoggingMiddleware(logger, svc)
	svc = middleware.NewTracingMiddleware(tracer, svc)
	counter, latency := prometheus.MakeMetrics(svcName, "api")
	svc = middleware.NewMetricsMiddleware(counter, latency, svc)

	return svc, nil
}

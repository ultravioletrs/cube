// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/absmach/callhome/pkg/client"
	"github.com/absmach/supermq"
	mglog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/authn/authsvc"
	"github.com/absmach/supermq/pkg/authz"
	authzsvc "github.com/absmach/supermq/pkg/authz/authsvc"
	domainsgrpc "github.com/absmach/supermq/pkg/domains/grpcclient"
	"github.com/absmach/supermq/pkg/grpcclient"
	"github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/postgres"
	"github.com/absmach/supermq/pkg/prometheus"
	"github.com/absmach/supermq/pkg/server"
	"github.com/absmach/supermq/pkg/server/http"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"github.com/ultravioletrs/cocos/pkg/clients"
	httpclient "github.com/ultravioletrs/cocos/pkg/clients/http"
	"github.com/ultravioletrs/cube/agent/audit"
	"github.com/ultravioletrs/cube/proxy"
	"github.com/ultravioletrs/cube/proxy/api"
	"github.com/ultravioletrs/cube/proxy/middleware"
	ppostgres "github.com/ultravioletrs/cube/proxy/postgres"
	"github.com/ultravioletrs/cube/proxy/router"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "cube_proxy"
	envPrefixHTTP  = "UV_CUBE_PROXY_"
	defSvcHTTPPort = "8900"
	envPrefixAgent = "UV_CUBE_AGENT_"
	envPrefixAuth  = "SMQ_AUTH_GRPC_"
	envPrefixDB    = "UV_CUBE_PROXY_DB_"
	defDB          = "postgres"
)

type config struct {
	LogLevel          string  `env:"UV_CUBE_PROXY_LOG_LEVEL"     envDefault:"info"`
	TargetURL         string  `env:"UV_CUBE_PROXY_TARGET_URL"    envDefault:"http://ollama:11434"`
	SendTelemetry     bool    `env:"SMQ_SEND_TELEMETRY"          envDefault:"true"`
	InstanceID        string  `env:"UV_CUBE_PROXY_INSTANCE_ID"   envDefault:""`
	JaegerURL         url.URL `env:"SMQ_JAEGER_URL"              envDefault:"http://localhost:4318/v1/traces"`
	TraceRatio        float64 `env:"SMQ_JAEGER_TRACE_RATIO"      envDefault:"1.0"`
	OpenSearchURL     string  `env:"UV_CUBE_OPENSEARCH_URL"      envDefault:"http://opensearch:9200"`
	RouterConfig      string  `env:"UV_CUBE_PROXY_ROUTER_CONFIG" envDefault:"docker/config.json"`
	GuardrailsEnabled bool    `env:"UV_CUBE_GUARDRAILS_ENABLED"  envDefault:"true"`
	GuardrailsURL     string  `env:"UV_CUBE_GUARDRAILS_URL"      envDefault:"http://guardrails:8001"`
	AgentURL          string  `env:"UV_CUBE_AGENT_URL"           envDefault:"http://cube-agent:8901"`
}

type fileConfig struct {
	Router router.Config `json:"router"`
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

	auth, authzz, closeAuth, err := initAuthClients(ctx, logger)
	if err != nil {
		logger.Error(err.Error())

		exitCode = 1

		return
	}
	defer closeAuth()

	tp, err := jaeger.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to init Jaeger: %s", err))
	}

	dbConfig := postgres.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())

		exitCode = 1

		return
	}

	db, err := postgres.Setup(dbConfig, *ppostgres.Migration())
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to %s database: %s", svcName, err))

		exitCode = 1

		return
	}
	defer db.Close()

	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("Error shutting down tracer provider: %v", err))
		}
	}()

	tracer := tp.Tracer(svcName)

	database := postgres.NewDatabase(db, dbConfig, tracer)

	repo := ppostgres.NewRepository(database)

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

	rter, err := initRouter(cfg.RouterConfig)
	if err != nil {
		logger.Error(err.Error())

		exitCode = 1

		return
	}

	// Load routes from database and update in-memory router
	if err := loadDatabaseRoutes(ctx, repo, rter); err != nil {
		logger.Error(fmt.Sprintf("failed to load routes from database: %s", err))
	}

	svc, err := newService(logger, tracer, &agentConfig, repo, rter)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create service: %s", err))

		exitCode = 1

		return
	}

	svc = middleware.AuthMiddleware(authzz)(svc)

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

	guardrailsCfg := api.GuardrailsConfig{
		Enabled:  cfg.GuardrailsEnabled,
		URL:      cfg.GuardrailsURL,
		AgentURL: cfg.AgentURL,
	}

	// Wrap agent transport with instrumented transport for aTLS audit logging
	agentTransport := agentClient.Transport()
	attestationType := deriveAttestationType(agentClient.Secure())
	instrumentedTransport := audit.NewInstrumentedTransport(agentTransport, attestationType)

	httpSvr := http.NewServer(
		ctx, cancel, svcName, httpServerConfig, api.MakeHandler(
			svc, cfg.InstanceID, auditSvc, authmMiddleware, idp, instrumentedTransport, rter, guardrailsCfg,
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

// newServiceWithRouter creates a service with router integration for dynamic route management.
func newService(
	logger *slog.Logger, tracer trace.Tracer, agentConfig *clients.AttestedClientConfig,
	repo proxy.Repository, rter *router.Router,
) (proxy.Service, error) {
	svc, err := proxy.NewWithRouter(agentConfig, repo, rter)
	if err != nil {
		return nil, err
	}

	svc = middleware.NewLoggingMiddleware(logger, svc)
	svc = middleware.NewTracingMiddleware(tracer, svc)
	counter, latency := prometheus.MakeMetrics(svcName, "api")
	svc = middleware.NewMetricsMiddleware(counter, latency, svc)

	return svc, nil
}

// loadDatabaseRoutes loads routes from the database and updates the in-memory router.
func loadDatabaseRoutes(ctx context.Context, repo proxy.Repository, rter *router.Router) error {
	routes, err := repo.ListRoutes(ctx)
	if err != nil {
		return err
	}

	if len(routes) > 0 {
		rter.UpdateRoutes(routes)
	}

	return nil
}

// initAuthClients initializes authentication and authorization gRPC clients.
func initAuthClients(
	ctx context.Context,
	logger *slog.Logger,
) (smqauthn.Authentication, authz.Authorization, func(), error) {
	grpcCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&grpcCfg, env.Options{Prefix: envPrefixAuth}); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load auth gRPC client configuration: %w", err)
	}

	auth, authnClient, err := authsvc.NewAuthentication(ctx, grpcCfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to init auth gRPC client: %w", err)
	}

	logger.Info("AuthN successfully connected to auth gRPC server " + authnClient.Secure())

	domainsAuthz, _, domainsClient, err := domainsgrpc.NewAuthorization(ctx, grpcCfg)
	if err != nil {
		authnClient.Close()

		return nil, nil, nil, fmt.Errorf("failed to init domains gRPC client: %w", err)
	}

	authorization, authzClient, err := authzsvc.NewAuthorization(ctx, grpcCfg, domainsAuthz)
	if err != nil {
		authnClient.Close()
		domainsClient.Close()

		return nil, nil, nil, fmt.Errorf("failed to init authz gRPC client: %w", err)
	}

	logger.Info("AuthZ successfully connected to auth gRPC server " + authzClient.Secure())

	closeFunc := func() {
		authnClient.Close()
		domainsClient.Close()
		authzClient.Close()
	}

	return auth, authorization, closeFunc, nil
}

// initRouter initializes the router from a config file.
func initRouter(configPath string) (*router.Router, error) {
	routerFile, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read router config file: %w", err)
	}

	var fileCfg fileConfig
	if err := json.Unmarshal(routerFile, &fileCfg); err != nil {
		return nil, fmt.Errorf("failed to parse router config file: %w", err)
	}

	return router.New(fileCfg.Router), nil
}

// deriveAttestationType determines the attestation type from the security string.
// The actual platform type (SNP, TDX, Azure) is extracted from certificate extensions
// during the TLS handshake; this function identifies if aTLS is enabled.
func deriveAttestationType(securityStr string) string {
	lower := strings.ToLower(securityStr)
	if strings.Contains(lower, "atls") {
		// aTLS or maTLS is enabled - actual platform type comes from cert extensions
		return "aTLS"
	}
	return "NoCC"
}

// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	stdhttp "net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/google/uuid"
	"github.com/ultravioletrs/cocos/pkg/clients"
	httpclient "github.com/ultravioletrs/cocos/pkg/clients/http"
	"github.com/ultravioletrs/cube/agent/audit"
	"github.com/ultravioletrs/cube/internal/atom"
	"github.com/ultravioletrs/cube/proxy"
	"github.com/ultravioletrs/cube/proxy/api"
	"github.com/ultravioletrs/cube/proxy/middleware"
	ppostgres "github.com/ultravioletrs/cube/proxy/postgres"
	"github.com/ultravioletrs/cube/proxy/router"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "cube_proxy"
	envPrefixHTTP  = "UV_CUBE_PROXY_"
	defSvcHTTPPort = "8900"
	envPrefixAgent = "UV_CUBE_AGENT_"
	envPrefixDB    = "UV_CUBE_PROXY_DB_"
	defDB          = "postgres"
)

type config struct {
	LogLevel       string        `env:"UV_CUBE_PROXY_LOG_LEVEL"     envDefault:"info"`
	TargetURL      string        `env:"UV_CUBE_PROXY_TARGET_URL"    envDefault:"http://ollama:11434"`
	InstanceID     string        `env:"UV_CUBE_PROXY_INSTANCE_ID"   envDefault:""`
	OpenSearchURL  string        `env:"UV_CUBE_OPENSEARCH_URL"      envDefault:"http://opensearch:9200"`
	RouterConfig   string        `env:"UV_CUBE_PROXY_ROUTER_CONFIG" envDefault:"docker/config.json"`
	AgentURL       string        `env:"UV_CUBE_AGENT_URL"           envDefault:"http://cube-agent:8901"`
	AtomGRPCURL    string        `env:"ATOM_GRPC_URL"               envDefault:"atom:8081"`
	AtomGraphQLURL string        `env:"ATOM_GRAPHQL_URL"            envDefault:"http://atom:8080/graphql"`
	AtomTimeout    time.Duration `env:"ATOM_TIMEOUT"                envDefault:"15s"`
}

type httpServerConfig struct {
	Host         string        `env:"HOST"                 envDefault:"0.0.0.0"`
	Port         string        `env:"PORT"                 envDefault:"8900"`
	ServerCert   string        `env:"SERVER_CERT"          envDefault:""`
	ServerKey    string        `env:"SERVER_KEY"           envDefault:""`
	ReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT"  envDefault:"10s"`
	WriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT" envDefault:"10s"`
	IdleTimeout  time.Duration `env:"SERVER_IDLE_TIMEOUT"  envDefault:"60s"`
}

type fileConfig struct {
	Router router.Config `json:"router"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger := newLogger(cfg.LogLevel)

	if cfg.InstanceID == "" {
		cfg.InstanceID = uuid.NewString()
	}

	atomClient, err := atom.NewClient(cfg.AtomGRPCURL, cfg.AtomGraphQLURL, cfg.AtomTimeout)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)

		return
	}
	defer atomClient.Close()

	dbConfig := ppostgres.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
		os.Exit(1)

		return
	}

	db, err := ppostgres.Setup(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to %s database: %s", svcName, err))
		os.Exit(1)

		return
	}
	defer db.Close()

	tracer := noop.NewTracerProvider().Tracer(svcName)
	repo := ppostgres.NewRepository(db)

	agentConfig := clients.AttestedClientConfig{}

	if err := env.ParseWithOptions(&agentConfig, env.Options{Prefix: envPrefixAgent}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s agent client configuration : %s", svcName, err))
		os.Exit(1)

		return
	}

	agentClient, err := httpclient.NewClient(&agentConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create agent HTTP client: %s", err))
		os.Exit(1)

		return
	}

	fileCfg, rter, err := initRouter(cfg.RouterConfig)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)

		return
	}

	// Load routes from database and update in-memory router
	if err := loadDatabaseRoutes(ctx, repo, rter, fileCfg.Router.Routes); err != nil {
		logger.Error(fmt.Sprintf("failed to load routes from database: %s", err))
	}

	svc, err := newService(logger, tracer, &agentConfig, repo, rter)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create service: %s", err))
		os.Exit(1)

		return
	}

	svc = middleware.AuthMiddleware(atomClient)(svc)

	logger.Info(fmt.Sprintf(
		"%s service %s client configured to connect to agent at %s with %s",
		svcName, svc.Secure(), agentConfig.URL, svc.Secure()))

	auditSvc := audit.NewMiddleware(logger, audit.Config{
		ComplianceMode:   true,
		EnablePIIMask:    true,
		EnableTokens:     true,
		SensitiveHeaders: []string{},
	})

	httpServerConfig := httpServerConfig{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		os.Exit(1)

		return
	}

	// Wrap agent transport with instrumented transport for aTLS audit logging
	agentTransport := agentClient.Transport()
	attestationType := deriveAttestationType(agentClient.Secure())
	instrumentedTransport := audit.NewInstrumentedTransport(agentTransport, attestationType)

	handler := api.MakeHandler(svc, cfg.InstanceID, auditSvc, atomClient, instrumentedTransport, rter)
	httpSvr := newHTTPServer(httpServerConfig, handler)

	g.Go(func() error {
		return serveHTTP(httpSvr, httpServerConfig, logger)
	})

	g.Go(func() error {
		<-ctx.Done()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer shutdownCancel()

		return httpSvr.Shutdown(shutdownCtx)
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

	return svc, nil
}

// loadDatabaseRoutes loads routes from the database and updates the in-memory router.
func loadDatabaseRoutes(
	ctx context.Context, repo proxy.Repository, rter *router.Router, defaultRoutes []router.RouteRule,
) error {
	routes, _, err := repo.ListRoutes(ctx, 0, proxy.MaxLimit)
	if err != nil {
		return err
	}

	existing := make(map[string]struct{}, len(routes))
	for i := range routes {
		existing[routes[i].Name] = struct{}{}
	}

	seeded := 0

	for i := range defaultRoutes {
		route := &defaultRoutes[i]
		if route.Enabled != nil && !*route.Enabled {
			continue
		}

		if _, ok := existing[route.Name]; ok {
			continue
		}

		if err := router.ValidateRoute(route); err != nil {
			log.Printf("Skipping invalid default route %s: %s", route.Name, err)

			continue
		}

		created, err := repo.CreateRoute(ctx, route)
		if err != nil {
			log.Printf("Failed to seed default route %s: %s", route.Name, err)

			continue
		}

		routes = append(routes, *created)
		existing[created.Name] = struct{}{}
		seeded++
	}

	if seeded > 0 {
		log.Printf("Seeded %d missing default routes into database", seeded)
	}

	if len(routes) > 0 {
		rter.UpdateRoutes(routes)
	}

	return nil
}

func newLogger(level string) *slog.Logger {
	var slogLevel slog.Level
	if err := slogLevel.UnmarshalText([]byte(strings.ToLower(level))); err != nil {
		slogLevel = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slogLevel}))
}

func newHTTPServer(cfg httpServerConfig, handler stdhttp.Handler) *stdhttp.Server {
	return &stdhttp.Server{
		Addr:         net.JoinHostPort(cfg.Host, cfg.Port),
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
}

func serveHTTP(srv *stdhttp.Server, cfg httpServerConfig, logger *slog.Logger) error {
	logger.Info("HTTP server starting", "addr", srv.Addr)

	var err error
	if cfg.ServerCert != "" && cfg.ServerKey != "" {
		err = srv.ListenAndServeTLS(cfg.ServerCert, cfg.ServerKey)
	} else {
		err = srv.ListenAndServe()
	}

	if errors.Is(err, stdhttp.ErrServerClosed) {
		return nil
	}

	return err
}

// initRouter initializes the router from a config file.
func initRouter(configPath string) (fileConfig, *router.Router, error) {
	routerFile, err := os.ReadFile(configPath)
	if err != nil {
		return fileConfig{}, nil, fmt.Errorf("failed to read router config file: %w", err)
	}

	var fileCfg fileConfig
	if err := json.Unmarshal(routerFile, &fileCfg); err != nil {
		return fileConfig{}, nil, fmt.Errorf("failed to parse router config file: %w", err)
	}

	return fileCfg, router.New(fileCfg.Router), nil
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

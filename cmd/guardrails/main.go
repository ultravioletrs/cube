// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"github.com/absmach/supermq/pkg/jaeger"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/trace"
	"log"
	"log/slog"
	"net/url"
	"os"

	"github.com/absmach/supermq/pkg/postgres"
	pgClient "github.com/absmach/supermq/pkg/postgres"
	"github.com/absmach/supermq/pkg/prometheus"
	"github.com/absmach/supermq/pkg/server"
	httpserver "github.com/absmach/supermq/pkg/server/http"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"github.com/ultraviolet/cube/guardrails"
	"github.com/ultraviolet/cube/guardrails/api"
	"github.com/ultraviolet/cube/guardrails/middleware"
	guardrailspg "github.com/ultraviolet/cube/guardrails/postgres"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "cube_guardrails"
	envPrefix      = "UV_CUBE_GUARDRAILS_"
	envPrefixHTTP  = "UV_CUBE_GUARDRAILS_HTTP_"
	envPrefixDB    = "UV_CUBE_GUARDRAILS_DB_"
	defDB          = "guardrails"
	defSvcHTTPPort = "8002"
)

type config struct {
	LogLevel         string  `env:"UV_CUBE_GUARDRAILS_LOG_LEVEL"     envDefault:"info"`
	InstanceID       string  `env:"UV_CUBE_GUARDRAILS_INSTANCE_ID"   envDefault:""`
	GuardrailsURL    string  `env:"UV_CUBE_GUARDRAILS_NEMO_URL"      envDefault:"http://nemo-guardrails:8001"`
	TargetURL        string  `env:"UV_CUBE_GUARDRAILS_TARGET_URL"    envDefault:"http://cube-agent:8901"`
	PolicyConfigPath string  `env:"UV_CUBE_GUARDRAILS_POLICY_CONFIG" envDefault:"/config/guardrails_config.yaml"`
	Timeout          int     `env:"UV_CUBE_GUARDRAILS_TIMEOUT"       envDefault:"30"`
	JaegerURL        url.URL `env:"SMQ_JAEGER_URL"               envDefault:"http://localhost:4318/v1/traces"`
	TraceRatio       float64 `env:"SMQ_JAEGER_TRACE_RATIO"       envDefault:"1.0"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger := initLogger(cfg.LogLevel)

	var exitCode int
	defer func() {
		os.Exit(exitCode)
	}()

	if cfg.InstanceID == "" {
		var err error
		if cfg.InstanceID, err = uuid.New().ID(); err != nil {
			logger.Error(fmt.Sprintf("failed to generate instanceID: %s", err))
			exitCode = 1
			return
		}
	}

	migration := *guardrailspg.Migration()

	dbConfig := postgres.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}

	db, err := pgClient.Setup(dbConfig, migration)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to %s database: %s", svcName, err))
		exitCode = 1
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error(fmt.Sprintf("Error closing database: %s", err))
		}
	}()

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

	svc, err := newService(db, logger, cfg, dbConfig, tracer)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	hs := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, cfg.InstanceID), logger)

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("%s service terminated: %s", svcName, err))
	}
}

func newService(db *sqlx.DB, logger *slog.Logger, cfg config, dbConfig postgres.Config, tracer trace.Tracer) (guardrails.Service, error) {
	database := postgres.NewDatabase(db, dbConfig, nil)

	repo := guardrailspg.NewRepository(database)

	serviceConfig := &guardrails.ServiceConfig{
		GuardrailsURL:    cfg.GuardrailsURL,
		TargetURL:        cfg.TargetURL,
		TLS:              guardrails.InsecureTLSConfig(),
		PolicyConfigPath: cfg.PolicyConfigPath,
		Timeout:          cfg.Timeout,
	}

	svc, err := guardrails.New(serviceConfig, repo)
	if err != nil {
		return nil, err
	}

	counter, latency := prometheus.MakeMetrics(svcName, "api")
	svc = middleware.NewLoggingMiddleware(logger, svc)
	svc = middleware.NewTracingMiddleware(tracer, svc)
	svc = middleware.NewMetricsMiddleware(counter, latency, svc)

	return svc, nil
}

func initLogger(levelText string) *slog.Logger {
	var level slog.Level
	if err := level.UnmarshalText([]byte(levelText)); err != nil {
		level = slog.LevelInfo
	}

	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(logHandler)
}

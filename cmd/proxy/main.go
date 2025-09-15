// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/absmach/callhome/pkg/client"
	"github.com/absmach/supermq"
	mglog "github.com/absmach/supermq/logger"
	"github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/prometheus"
	"github.com/absmach/supermq/pkg/server"
	smqHttp "github.com/absmach/supermq/pkg/server/http"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"github.com/ultraviolet/cube/proxy"
	"github.com/ultraviolet/cube/proxy/api"
	"github.com/ultraviolet/cube/proxy/middleware"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "cube_proxy"
	envPrefixHTTP  = "UV_CUBE_PROXY_"
	defSvcHTTPPort = "8900"
)

type config struct {
	LogLevel          string  `env:"UV_CUBE_PROXY_LOG_LEVEL"      envDefault:"info"`
	TargetURL         string  `env:"UV_CUBE_PROXY_TARGET_URL"     envDefault:"http://cube-agent:8901"`
	GuardrailsURL     string  `env:"UV_CUBE_GUARDRAILS_URL"       envDefault:"http://cube-guardrails:8002"`
	GuardrailsEnabled bool    `env:"UV_CUBE_GUARDRAILS_ENABLED"   envDefault:"true"`
	SendTelemetry     bool    `env:"SMQ_SEND_TELEMETRY"           envDefault:"true"`
	InstanceID        string  `env:"UV_CUBE_PROXY_INSTANCE_ID"    envDefault:""`
	JaegerURL         url.URL `env:"SMQ_JAEGER_URL"               envDefault:"http://localhost:4318/v1/traces"`
	TraceRatio        float64 `env:"SMQ_JAEGER_TRACE_RATIO"       envDefault:"1.0"`
}

type guardrailsHTTPClient struct {
	guardrailsURL string
	targetURL     string
	httpClient    *http.Client
	logger        *slog.Logger
}

type guardrailsProxyService struct {
	proxy            proxy.Service
	guardrailsClient *guardrailsHTTPClient
}

type directProxyService struct {
	proxy proxy.Service
}

func newGuardrailsHTTPClient(guardrailsURL, targetURL string, logger *slog.Logger) *guardrailsHTTPClient {
	return &guardrailsHTTPClient{
		guardrailsURL: guardrailsURL,
		targetURL:     targetURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		logger: logger,
	}
}

func (g *guardrailsProxyService) Proxy() *httputil.ReverseProxy {
	target, _ := url.Parse(g.guardrailsClient.guardrailsURL)
	reverseProxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := reverseProxy.Director
	reverseProxy.Director = func(req *http.Request) {
		log.Printf("[MAIN-PROXY] Request routing through GUARDRAILS HTTP service: %s %s", req.Method, req.URL.Path)
		originalDirector(req)
	}

	reverseProxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return reverseProxy
}

func (d *directProxyService) Proxy() *httputil.ReverseProxy {
	reverseProxy := d.proxy.Proxy()

	originalDirector := reverseProxy.Director
	reverseProxy.Director = func(req *http.Request) {
		log.Printf("[MAIN-PROXY] Request routing DIRECTLY (no guardrails): %s %s", req.Method, req.URL.Path)
		originalDirector(req)
	}

	return reverseProxy
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

	svc, err := newService(logger, tracer, cfg)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create service: %s", err))

		exitCode = 1

		return
	}

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))

		exitCode = 1

		return
	}

	httpSvr := smqHttp.NewServer(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, cfg.InstanceID), logger)

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

func newService(logger *slog.Logger, tracer trace.Tracer, cfg config) (proxy.Service, error) {
	var svc proxy.Service

	if cfg.GuardrailsEnabled {
		logger.Info("guardrails enabled", "guardrails_url", cfg.GuardrailsURL, "target_url", cfg.TargetURL)

		guardrailsClient := newGuardrailsHTTPClient(cfg.GuardrailsURL, cfg.TargetURL, logger)

		proxySvc, err := proxy.New(&proxy.Config{
			AgentURL: cfg.TargetURL,
			TLS:      proxy.InsecureTLSConfig(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy service: %w", err)
		}

		svc = &guardrailsProxyService{
			proxy:            proxySvc,
			guardrailsClient: guardrailsClient,
		}
	} else {
		logger.Info("guardrails disabled, using direct proxy", "target_url", cfg.TargetURL)
		proxySvc, err := proxy.New(&proxy.Config{AgentURL: cfg.TargetURL, TLS: proxy.InsecureTLSConfig()})
		if err != nil {
			return nil, err
		}

		svc = &directProxyService{proxy: proxySvc}
	}

	svc = middleware.NewLoggingMiddleware(logger, svc)
	svc = middleware.NewTracingMiddleware(tracer, svc)
	counter, latency := prometheus.MakeMetrics(svcName, "api")
	svc = middleware.NewMetricsMiddleware(counter, latency, svc)

	return svc, nil
}

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/absmach/callhome/pkg/client"
	"github.com/absmach/magistrala"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/grpcclient"
	"github.com/absmach/magistrala/pkg/server"
	"github.com/absmach/mproxy"
	mproxyhttp "github.com/absmach/mproxy/pkg/http"
	"github.com/absmach/mproxy/pkg/session"
	"github.com/caarlos0/env/v11"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "vault_proxy"
	envPrefix      = "UV_VAULT_PROXY_"
	envPrefixAuth  = "MG_AUTH_GRPC_"
	defSvcHTTPPort = "8900"
)

type config struct {
	LogLevel      string `env:"UV_VAULT_PROXY_LOG_LEVEL"  envDefault:"info"`
	SendTelemetry bool   `env:"MG_SEND_TELEMETRY"         envDefault:"true"`
	TargetURL     string `env:"UV_VAULT_PROXY_TARGET_URL" envDefault:"http://ollama:11434"`
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

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefix}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	authCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&authCfg, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	authClient, authHandler, err := grpcclient.SetupAuthClient(ctx, authCfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authHandler.Close()

	logger.Info("Auth service gRPC client successfully connected to auth gRPC server " + authHandler.Secure())

	svc := NewHandler(authClient)

	if cfg.SendTelemetry {
		chc := client.New(svcName, magistrala.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return proxyHTTP(ctx, cfg.TargetURL, httpServerConfig, logger, svc)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("HTTP adapter service terminated: %s", err))
	}
}

func proxyHTTP(ctx context.Context, url string, cfg server.Config, logger *slog.Logger, sessionHandler session.Handler) error {
	config := mproxy.Config{
		Address:    fmt.Sprintf("%s:%s", "", cfg.Port),
		Target:     url,
		PathPrefix: "/",
	}

	if cfg.CertFile != "" || cfg.KeyFile != "" {
		tlsCert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return err
		}
		config.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
		}
	}
	mp, err := mproxyhttp.NewProxy(config, sessionHandler, logger)
	if err != nil {
		return err
	}
	http.HandleFunc("/", mp.ServeHTTP)

	errCh := make(chan error)
	switch {
	case cfg.CertFile != "" || cfg.KeyFile != "":
		go func() {
			errCh <- mp.Listen(ctx)
		}()
		logger.Info(fmt.Sprintf("%s service https server listening at %s:%s with TLS cert %s and key %s", svcName, cfg.Host, cfg.Port, cfg.CertFile, cfg.KeyFile))
	default:
		go func() {
			errCh <- mp.Listen(ctx)
		}()
		logger.Info(fmt.Sprintf("%s service http server listening at %s:%s without TLS", svcName, cfg.Host, cfg.Port))
	}

	select {
	case <-ctx.Done():
		logger.Info(fmt.Sprintf("proxy HTTP shutdown at %s", config.Target))
		return nil
	case err := <-errCh:
		return err
	}
}

package agent

import (
	"context"
	"fmt"
	"log"
	"os"

	mglog "github.com/absmach/supermq/logger"
	"github.com/absmach/supermq/pkg/authn/authsvc"
	"github.com/absmach/supermq/pkg/grpcclient"
	"github.com/absmach/supermq/pkg/server"
	"github.com/absmach/supermq/pkg/server/http"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"github.com/ultraviolet/cube/agent"
	"github.com/ultraviolet/cube/agent/api"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "agent"
	defSvcHTTPPort = "8901"
	envPrefixHTTP  = "UV_CUBE_AGENT_"
	envPrefixAuth  = "SMQ_AUTH_GRPC_"
)

type Config struct {
	LogLevel   string `env:"UV_CUBE_AGENT_LOG_LEVEL"   envDefault:"info"`
	TargetURL  string `env:"UV_CUBE_AGENT_TARGET_URL"  envDefault:"http://ollama:11434"`
	InstanceID string `env:"UV_CUBE_AGENT_INSTANCE_ID" envDefault:""`
}

func main() {
	cfg := Config{}
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

	grpcCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&grpcCfg, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load auth gRPC client configuration : %s", err))
		exitCode = 1

		return
	}

	ctx := context.Background()

	auth, authnClient, err := authsvc.NewAuthentication(ctx, grpcCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to init auth gRPC client: %s", err))
		exitCode = 1

		return
	}
	defer authnClient.Close()
	logger.Info("AuthN  successfully connected to auth gRPC server " + authnClient.Secure())

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1

		return
	}

	svc, err := agent.NewAgentService(agent.Config{
		OllamaURL: cfg.TargetURL,
		TLS:       agent.InsecureTLSConfig()}, auth)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create agent service: %s", err))
		exitCode = 1

		return
	}

	ctx, cancel := context.WithCancel(ctx)
	g, ctx := errgroup.WithContext(ctx)

	httpSvr := http.NewServer(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, logger, cfg.InstanceID), logger)

	g.Go(func() error {
		return httpSvr.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, httpSvr)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Agent service terminated: %s", err))
	}
}

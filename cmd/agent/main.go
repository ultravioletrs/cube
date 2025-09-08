// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package main

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
	"github.com/ultravioletrs/cocos/pkg/attestation"
	"github.com/ultravioletrs/cocos/pkg/attestation/azure"
	"github.com/ultravioletrs/cocos/pkg/attestation/tdx"
	"github.com/ultravioletrs/cocos/pkg/attestation/vtpm"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "agent"
	defSvcHTTPPort = "8901"
	envPrefixHTTP  = "UV_CUBE_AGENT_"
	envPrefixAuth  = "SMQ_AUTH_GRPC_"
)

type Config struct {
	LogLevel      string `env:"UV_CUBE_AGENT_LOG_LEVEL"   envDefault:"info"`
	TargetURL     string `env:"UV_CUBE_AGENT_TARGET_URL"  envDefault:"http://ollama:11434"`
	InstanceID    string `env:"UV_CUBE_AGENT_INSTANCE_ID" envDefault:""`
	AgentMaaURL   string `env:"AGENT_MAA_URL"             envDefault:"https://sharedeus2.eus2.attest.azure.net"`
	AgentOSBuild  string `env:"AGENT_OS_BUILD"            envDefault:"UVC"`
	AgentOSDistro string `env:"AGENT_OS_DISTRO"           envDefault:"UVC"`
	AgentOSType   string `env:"AGENT_OS_TYPE"             envDefault:"UVC"`
	Vmpl          int    `env:"AGENT_VMPL"                envDefault:"2"`
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

	var provider attestation.Provider

	ccPlatform := attestation.CCPlatform()

	azureConfig := azure.NewEnvConfigFromAgent(
		cfg.AgentOSBuild,
		cfg.AgentOSType,
		cfg.AgentOSDistro,
		cfg.AgentMaaURL,
	)
	azure.InitializeDefaultMAAVars(azureConfig)

	switch ccPlatform {
	case attestation.SNP:
		provider = vtpm.NewProvider(false, uint(cfg.Vmpl))
	case attestation.SNPvTPM:
		provider = vtpm.NewProvider(true, uint(cfg.Vmpl))
	case attestation.Azure:
		provider = azure.NewProvider()
	case attestation.TDX:
		provider = tdx.NewProvider()
	case attestation.NoCC:
		logger.Info("TEE device not found")

		provider = &attestation.EmptyProvider{}
	case attestation.VTPM, attestation.AzureToken:
		logger.Info("vTPM attestation is not supported")

		exitCode = 1

		return
	}

	svc, err := agent.New(&agent.Config{
		OllamaURL: cfg.TargetURL,
		TLS:       agent.InsecureTLSConfig(),
	}, auth, provider)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create agent service: %s", err))

		exitCode = 1

		return
	}

	ctx, cancel := context.WithCancel(ctx)
	g, ctx := errgroup.WithContext(ctx)

	httpSvr := http.NewServer(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, cfg.InstanceID), logger)

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

// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"log/slog"
)

// Config is a struct that contains the configuration for the server.
type Config struct {
	Host         string `env:"HOST"            envDefault:"localhost"`
	Port         string `env:"PORT"            envDefault:""`
	CertFile     string `env:"SERVER_CERT"     envDefault:""`
	KeyFile      string `env:"SERVER_KEY"      envDefault:""`
	ServerCAFile string `env:"SERVER_CA_CERTS" envDefault:""`
	ClientCAFile string `env:"CLIENT_CA_CERTS" envDefault:""`
	AttestedTLS  bool   `env:"ATTESTED_TLS"    envDefault:"false"`
}

type BaseServer struct {
	Ctx      context.Context
	Cancel   context.CancelFunc
	Name     string
	Address  string
	Config   Config
	Logger   *slog.Logger
	Protocol string
}

func NewBaseServer(
	ctx context.Context, cancel context.CancelFunc, name string, config *Config, logger *slog.Logger,
) BaseServer {
	address := fmt.Sprintf("%s:%s", config.Host, config.Port)

	return BaseServer{
		Ctx:     ctx,
		Cancel:  cancel,
		Name:    name,
		Address: address,
		Config:  *config,
		Logger:  logger,
	}
}

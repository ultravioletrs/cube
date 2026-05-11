// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
)

const (
	ProviderLocal = "local"
	ProviderS3    = "s3"
)

// Store persists and retrieves raw uploaded objects.
type Store interface {
	Put(ctx context.Context, key, contentType string, size int64, body io.Reader) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

// Config describes how uploaded files are persisted.
type Config struct {
	Provider string
	LocalDir string

	S3Endpoint        string
	S3Region          string
	S3Bucket          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3UseSSL          bool
	S3PathStyle       bool
	S3EnsureBucket    bool
}

func NewStore(cfg Config) (Store, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		provider = ProviderLocal
	}

	switch provider {
	case ProviderLocal:
		return newLocalStore(cfg.LocalDir)
	case ProviderS3:
		return newS3Store(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage provider %q", cfg.Provider)
	}
}

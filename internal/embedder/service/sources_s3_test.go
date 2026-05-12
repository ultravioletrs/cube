// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"encoding/json"
	"testing"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

func TestSanitizeS3Config(t *testing.T) {
	t.Run("normalizes valid config", func(t *testing.T) {
		raw, err := sanitizeS3Config(json.RawMessage(`{
			"endpoint":"https://s3.example.com",
			"bucket":" docs ",
			"region":" ",
			"root_path":" /team/docs/ ",
			"scope_paths":["/team/docs/specs","team/docs/specs","team/docs/guides/../guides"],
			"selected_paths":["team/docs/specs/file.txt","team/docs/specs/./file.txt"],
			"access_key_id":" key ",
			"secret_access_key":" secret ",
			"config_ref":" secret/s3 "
		}`))
		if err != nil {
			t.Fatalf("sanitizeS3Config returned error: %v", err)
		}

		var cfg domain.S3Config
		if err := json.Unmarshal(raw, &cfg); err != nil {
			t.Fatalf("decode sanitized config: %v", err)
		}

		if cfg.Endpoint != "s3.example.com" {
			t.Fatalf("expected normalized endpoint, got %q", cfg.Endpoint)
		}
		if cfg.Bucket != "docs" {
			t.Fatalf("expected trimmed bucket, got %q", cfg.Bucket)
		}
		if cfg.Region != "us-east-1" {
			t.Fatalf("expected default region, got %q", cfg.Region)
		}
		if cfg.RootPath != "team/docs" {
			t.Fatalf("expected normalized root_path, got %q", cfg.RootPath)
		}
		if len(cfg.ScopePaths) != 2 {
			t.Fatalf("expected 2 deduplicated scopes, got %d: %#v", len(cfg.ScopePaths), cfg.ScopePaths)
		}
		if len(cfg.SelectedPaths) != 1 || cfg.SelectedPaths[0] != "team/docs/specs/file.txt" {
			t.Fatalf("unexpected selected_paths: %#v", cfg.SelectedPaths)
		}
		if cfg.AccessKeyID != "key" || cfg.SecretAccessKey != "secret" {
			t.Fatalf("expected trimmed credentials, got key=%q secret=%q", cfg.AccessKeyID, cfg.SecretAccessKey)
		}
		if cfg.ConfigRef != "secret/s3" {
			t.Fatalf("expected trimmed config_ref, got %q", cfg.ConfigRef)
		}
		if cfg.UseSSL == nil || !*cfg.UseSSL {
			t.Fatalf("expected use_ssl=true, got %#v", cfg.UseSSL)
		}
		if cfg.PathStyle == nil || !*cfg.PathStyle {
			t.Fatalf("expected path_style=true, got %#v", cfg.PathStyle)
		}
	})

	t.Run("requires bucket", func(t *testing.T) {
		_, err := sanitizeS3Config(json.RawMessage(`{"root_path":"team/docs"}`))
		if err == nil {
			t.Fatal("expected error when bucket is missing")
		}
	})

	t.Run("requires scope information", func(t *testing.T) {
		_, err := sanitizeS3Config(json.RawMessage(`{"bucket":"docs"}`))
		if err == nil {
			t.Fatal("expected error when root_path, scope_paths and selected_paths are missing")
		}
	})

	t.Run("requires full credentials pair", func(t *testing.T) {
		_, err := sanitizeS3Config(json.RawMessage(`{
			"bucket":"docs",
			"root_path":"team/docs",
			"access_key_id":"key"
		}`))
		if err == nil {
			t.Fatal("expected error when secret_access_key is missing")
		}
	})
}

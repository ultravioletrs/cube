// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"encoding/json"
	"testing"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

func TestSanitizeRcloneConfig(t *testing.T) {
	t.Run("normalizes valid config", func(t *testing.T) {
		raw, err := sanitizeRcloneConfig(json.RawMessage(`{
			"remote":" corp-files ",
			"root_path":" /team/docs/ ",
			"scope_paths":["/team/docs/specs","team/docs/specs","team/docs/guides/../guides"],
			"config_ref":" secret/rclone "
		}`))
		if err != nil {
			t.Fatalf("sanitizeRcloneConfig returned error: %v", err)
		}

		var cfg domain.RcloneConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			t.Fatalf("decode sanitized config: %v", err)
		}

		if cfg.Remote != "corp-files" {
			t.Fatalf("expected remote to be trimmed, got %q", cfg.Remote)
		}
		if cfg.RootPath != "team/docs" {
			t.Fatalf("expected normalized root_path, got %q", cfg.RootPath)
		}
		if len(cfg.ScopePaths) != 2 {
			t.Fatalf("expected 2 deduplicated scopes, got %d: %#v", len(cfg.ScopePaths), cfg.ScopePaths)
		}
		if cfg.ScopePaths[0] != "team/docs/guides" || cfg.ScopePaths[1] != "team/docs/specs" {
			t.Fatalf("unexpected normalized scopes: %#v", cfg.ScopePaths)
		}
		if cfg.ConfigRef != "secret/rclone" {
			t.Fatalf("expected config_ref to be trimmed, got %q", cfg.ConfigRef)
		}
	})

	t.Run("requires remote", func(t *testing.T) {
		_, err := sanitizeRcloneConfig(json.RawMessage(`{"root_path":"team/docs"}`))
		if err == nil {
			t.Fatal("expected error when remote is missing")
		}
	})

	t.Run("requires root or scopes", func(t *testing.T) {
		_, err := sanitizeRcloneConfig(json.RawMessage(`{"remote":"corp-files"}`))
		if err == nil {
			t.Fatal("expected error when root_path and scope_paths are missing")
		}
	})

	t.Run("rejects scope outside root", func(t *testing.T) {
		_, err := sanitizeRcloneConfig(json.RawMessage(`{
			"remote":"corp-files",
			"root_path":"team/docs",
			"scope_paths":["team/other"]
		}`))
		if err == nil {
			t.Fatal("expected error for scope outside root")
		}
	})
}

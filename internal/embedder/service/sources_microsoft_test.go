// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"encoding/json"
	"testing"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

func TestSanitizeMicrosoftConfig(t *testing.T) {
	t.Run("normalizes valid config with access token", func(t *testing.T) {
		raw, err := sanitizeMicrosoftConfig(json.RawMessage(`{
			"access_token":" token ,",
			"drive_id":" drive-1 ",
			"root_path":" /team/docs/ ",
			"scope_paths":["/team/docs/specs","team/docs/specs","team/docs/guides/../guides"],
			"selected_paths":["team/docs/specs/file.docx","team/docs/specs/./file.docx"],
			"config_ref":" secret/ms "
		}`))
		if err != nil {
			t.Fatalf("sanitizeMicrosoftConfig returned error: %v", err)
		}

		var cfg domain.MicrosoftConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			t.Fatalf("decode sanitized config: %v", err)
		}

		if cfg.AccessToken != "token" {
			t.Fatalf("expected trimmed access token, got %q", cfg.AccessToken)
		}
		if cfg.DriveID != "drive-1" {
			t.Fatalf("expected trimmed drive_id, got %q", cfg.DriveID)
		}
		if cfg.RootPath != "team/docs" {
			t.Fatalf("expected normalized root_path, got %q", cfg.RootPath)
		}
		if len(cfg.ScopePaths) != 2 {
			t.Fatalf("expected 2 deduplicated scope paths, got %d: %#v", len(cfg.ScopePaths), cfg.ScopePaths)
		}
		if len(cfg.SelectedPaths) != 1 || cfg.SelectedPaths[0] != "team/docs/specs/file.docx" {
			t.Fatalf("unexpected selected_paths: %#v", cfg.SelectedPaths)
		}
		if cfg.ConfigRef != "secret/ms" {
			t.Fatalf("expected trimmed config_ref, got %q", cfg.ConfigRef)
		}
	})

	t.Run("accepts client credentials without access token", func(t *testing.T) {
		_, err := sanitizeMicrosoftConfig(json.RawMessage(`{
			"tenant_id":"tenant",
			"client_id":"client",
			"client_secret":"secret",
			"root_path":"team/docs"
		}`))
		if err != nil {
			t.Fatalf("expected valid client credential config, got error: %v", err)
		}
	})

	t.Run("requires scope information", func(t *testing.T) {
		_, err := sanitizeMicrosoftConfig(json.RawMessage(`{
			"access_token":"token"
		}`))
		if err == nil {
			t.Fatal("expected error when root_path, scope_paths and selected_paths are missing")
		}
	})

	t.Run("requires full client credentials tuple", func(t *testing.T) {
		_, err := sanitizeMicrosoftConfig(json.RawMessage(`{
			"client_id":"client",
			"root_path":"team/docs"
		}`))
		if err == nil {
			t.Fatal("expected error for partial microsoft client credentials")
		}
	})

	t.Run("requires auth", func(t *testing.T) {
		_, err := sanitizeMicrosoftConfig(json.RawMessage(`{
			"drive_id":"drive-1",
			"root_path":"team/docs"
		}`))
		if err == nil {
			t.Fatal("expected error when auth is missing")
		}
	})
}

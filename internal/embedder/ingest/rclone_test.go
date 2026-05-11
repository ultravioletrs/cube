// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest

import "testing"

func TestNormalizeRcloneScopes(t *testing.T) {
	t.Run("normalizes and deduplicates", func(t *testing.T) {
		scopes, err := normalizeRcloneScopes("team/docs", []string{
			"team/docs/specs",
			"/team/docs/guides/../guides",
			"team/docs",
		})
		if err != nil {
			t.Fatalf("normalizeRcloneScopes returned error: %v", err)
		}
		if len(scopes) != 3 {
			t.Fatalf("expected 3 scopes, got %d: %#v", len(scopes), scopes)
		}
		if scopes[0] != "team/docs" || scopes[1] != "team/docs/guides" || scopes[2] != "team/docs/specs" {
			t.Fatalf("unexpected scopes: %#v", scopes)
		}
	})

	t.Run("rejects scope outside root", func(t *testing.T) {
		_, err := normalizeRcloneScopes("team/docs", []string{"team/other"})
		if err == nil {
			t.Fatal("expected error for scope outside root")
		}
	})
}

func TestSanitizeRcloneRemote(t *testing.T) {
	if got := sanitizeRcloneRemote("prod_remote-1.2"); got != "prod_remote-1.2" {
		t.Fatalf("expected valid remote to pass through, got %q", got)
	}
	if got := sanitizeRcloneRemote("prod:remote"); got != "" {
		t.Fatalf("expected invalid remote to be rejected, got %q", got)
	}
}

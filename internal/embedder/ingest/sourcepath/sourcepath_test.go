// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package sourcepath

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"":               "",
		"/":              "",
		".":              "",
		"/team/docs/":    "team/docs",
		"team/docs":      "team/docs",
		"team/docs/../x": "team/x",
		"//team//docs//": "team/docs",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeScopes(t *testing.T) {
	t.Run("normalizes and deduplicates", func(t *testing.T) {
		scopes, err := NormalizeScopes("team/docs", []string{
			"team/docs/specs",
			"/team/docs/guides/../guides",
			"team/docs",
		})
		if err != nil {
			t.Fatalf("NormalizeScopes returned error: %v", err)
		}
		if len(scopes) != 3 {
			t.Fatalf("expected 3 scopes, got %d: %#v", len(scopes), scopes)
		}
		if scopes[0] != "team/docs" || scopes[1] != "team/docs/guides" || scopes[2] != "team/docs/specs" {
			t.Fatalf("unexpected scopes: %#v", scopes)
		}
	})

	t.Run("rejects scope outside root", func(t *testing.T) {
		_, err := NormalizeScopes("team/docs", []string{"team/other"})
		if err == nil {
			t.Fatal("expected error for scope outside root")
		}
	})

	t.Run("empty scopes fall back to root", func(t *testing.T) {
		scopes, err := NormalizeScopes("team/docs", nil)
		if err != nil || len(scopes) != 1 || scopes[0] != "team/docs" {
			t.Fatalf("unexpected: %#v err=%v", scopes, err)
		}
		scopes, err = NormalizeScopes("", nil)
		if err != nil || len(scopes) != 1 || scopes[0] != "" {
			t.Fatalf("unexpected whole-tree: %#v err=%v", scopes, err)
		}
	})
}

func TestIsWithinRoot(t *testing.T) {
	if !IsWithinRoot("", "anything/here") {
		t.Error("empty root must contain everything")
	}
	if !IsWithinRoot("a/b", "a/b/c") {
		t.Error("nested path must be within root")
	}
	if IsWithinRoot("a/b", "a/c") {
		t.Error("sibling path must not be within root")
	}
	if !IsWithinRoot("a/b", "a/b") {
		t.Error("exact root must be within")
	}
}

func TestNormalizeList(t *testing.T) {
	if got := NormalizeList(nil); got != nil {
		t.Errorf("expected nil, got %#v", got)
	}
	got := NormalizeList([]string{"/b/", "a", "b", "", "."})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected dedup/sort: %#v", got)
	}
}

func TestValidateScopesWithinRoot(t *testing.T) {
	if err := ValidateScopesWithinRoot("", []string{"x"}); err != nil {
		t.Errorf("empty root must pass: %v", err)
	}
	if err := ValidateScopesWithinRoot("a/b", []string{"a/b/c"}); err != nil {
		t.Errorf("nested scope must pass: %v", err)
	}
	if err := ValidateScopesWithinRoot("a/b", []string{"a/c"}); err == nil {
		t.Error("scope outside root must fail")
	}
}

// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

// Package sourcepath holds the path-normalization and scope-validation helpers
// shared by the object/cloud source providers (S3, Microsoft) and the source
// service. They were previously copy-pasted per package.
package sourcepath

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// Normalize trims, cleans and de-roots a slash path, returning "" for the root.
func Normalize(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == "/" {
		return ""
	}
	clean := path.Clean("/" + strings.TrimPrefix(value, "/"))
	if clean == "/" {
		return ""
	}
	return strings.TrimPrefix(clean, "/")
}

// IsWithinRoot reports whether scopedPath is the root or nested under it.
// An empty root or scope is treated as the whole tree (within).
func IsWithinRoot(rootPath, scopedPath string) bool {
	rootPath = Normalize(rootPath)
	scopedPath = Normalize(scopedPath)
	if rootPath == "" || scopedPath == "" {
		return true
	}
	rootAbs := "/" + rootPath
	scopeAbs := "/" + scopedPath
	return scopeAbs == rootAbs || strings.HasPrefix(scopeAbs, rootAbs+"/")
}

// NormalizeScopes normalizes, validates-against-root and de-duplicates scope
// paths. With no scopes it returns the root (or [""] for whole-tree). It errors
// if any scope falls outside rootPath.
func NormalizeScopes(rootPath string, scopePaths []string) ([]string, error) {
	rootPath = Normalize(rootPath)
	rootAbs := "/" + rootPath
	if rootPath == "" {
		rootAbs = "/"
	}
	if len(scopePaths) == 0 {
		if rootPath == "" {
			return []string{""}, nil
		}
		return []string{rootPath}, nil
	}

	seen := make(map[string]struct{}, len(scopePaths))
	out := make([]string, 0, len(scopePaths))
	for _, raw := range scopePaths {
		scope := Normalize(raw)
		scopeAbs := "/" + scope
		if scope == "" {
			scopeAbs = "/"
		}

		if rootAbs != "/" {
			if scopeAbs != rootAbs && !strings.HasPrefix(scopeAbs, rootAbs+"/") {
				return nil, fmt.Errorf("scope path %q is outside approved root %q", raw, rootPath)
			}
		}

		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}

	sort.Strings(out)
	return out, nil
}

// NormalizeList normalizes, drops empties, de-duplicates and sorts a path list.
// Returns nil when nothing remains.
func NormalizeList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		item := Normalize(value)
		if item == "" {
			continue
		}
		set[item] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

// SelectionContains reports whether filePath is covered by the selected paths:
// an exact match of a selected file, or nested under a selected folder. An empty
// selected entry ("") matches the whole tree. Used so picking a folder ingests
// its entire subtree, not just an (impossible) exact file match.
func SelectionContains(selected []string, filePath string) bool {
	fp := Normalize(filePath)
	for _, sel := range selected {
		s := Normalize(sel)
		if s == "" {
			return true
		}
		if fp == s || strings.HasPrefix(fp, s+"/") {
			return true
		}
	}
	return false
}

// ValidateScopesWithinRoot errors if any scope path is outside rootPath.
// Empty root or empty scope list passes.
func ValidateScopesWithinRoot(rootPath string, scopePaths []string) error {
	rootPath = Normalize(rootPath)
	if rootPath == "" || len(scopePaths) == 0 {
		return nil
	}
	rootAbs := "/" + rootPath

	for _, scope := range scopePaths {
		scopeAbs := "/" + Normalize(scope)
		if scopeAbs == "/" {
			continue
		}
		if scopeAbs != rootAbs && !strings.HasPrefix(scopeAbs, rootAbs+"/") {
			return fmt.Errorf("scope path %q is outside root_path %q", scope, rootPath)
		}
	}
	return nil
}

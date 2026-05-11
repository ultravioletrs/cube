// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type localStore struct {
	root string
}

func newLocalStore(root string) (*localStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("local storage directory is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &localStore{root: root}, nil
}

func (s *localStore) Put(_ context.Context, key, _ string, _ int64, body io.Reader) error {
	path, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, body)
	return err
}

func (s *localStore) Get(_ context.Context, key string) ([]byte, error) {
	path, err := s.resolve(key)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

func (s *localStore) Delete(_ context.Context, key string) error {
	path, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *localStore) resolve(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("object key is required")
	}

	// Path-clean in slash form first to reject traversal.
	clean := strings.TrimPrefix(path.Clean("/"+key), "/")
	if clean == "" || clean == "." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("invalid object key")
	}

	full := filepath.Join(s.root, filepath.FromSlash(clean))
	rootClean := filepath.Clean(s.root)
	fullClean := filepath.Clean(full)
	if fullClean != rootClean && !strings.HasPrefix(fullClean, rootClean+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid object key path")
	}
	return full, nil
}

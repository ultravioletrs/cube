// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"context"
	"time"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

// SourceFile is a normalized external file representation used by the sync flow.
type SourceFile struct {
	ExternalID       string
	Name             string
	ExternalURL      string
	ExternalRef      string
	MimeType         string
	SourceVersion    string
	SourceModifiedAt *time.Time
}

// SourceProviderCapabilities describes what integration operations are supported.
type SourceProviderCapabilities struct {
	SupportsList     bool
	SupportsDownload bool
	SupportsBrowse   bool
}

// SourceProvider hides provider-specific file listing and download behavior.
type SourceProvider interface {
	Type() domain.SourceType
	Capabilities() SourceProviderCapabilities
	ListFiles(ctx context.Context, userID string, src domain.Source) ([]SourceFile, error)
	DownloadRecord(ctx context.Context, rec domain.Record, src domain.Source) (string, *int, error)
	PrunesStaleRecords() bool
}

// SourceProviderRegistry stores providers keyed by source type.
type SourceProviderRegistry struct {
	providers map[domain.SourceType]SourceProvider
	aliases   map[domain.SourceType]domain.SourceType
}

// NewSourceProviderRegistry creates a registry from the provided providers.
func NewSourceProviderRegistry(providers ...SourceProvider) *SourceProviderRegistry {
	reg := &SourceProviderRegistry{
		providers: make(map[domain.SourceType]SourceProvider, len(providers)),
		aliases:   make(map[domain.SourceType]domain.SourceType),
	}
	for _, provider := range providers {
		reg.Register(provider)
	}
	return reg
}

// Register adds or overwrites a provider for the provider's source type.
func (r *SourceProviderRegistry) Register(provider SourceProvider) {
	if r == nil || provider == nil {
		return
	}
	r.providers[provider.Type()] = provider
}

// Provider returns a provider for source type t.
func (r *SourceProviderRegistry) Provider(t domain.SourceType) (SourceProvider, bool) {
	if r == nil {
		return nil, false
	}
	provider, ok := r.providers[t]
	if ok {
		return provider, true
	}

	visited := make(map[domain.SourceType]struct{}, 4)
	current := t
	for i := 0; i < 4; i++ {
		target, hasAlias := r.aliases[current]
		if !hasAlias {
			return nil, false
		}
		if _, seen := visited[target]; seen {
			return nil, false
		}
		visited[target] = struct{}{}
		provider, ok = r.providers[target]
		if ok {
			return provider, true
		}
		current = target
	}
	return nil, false
}

// RegisterAlias maps one source type to another provider-backed type.
func (r *SourceProviderRegistry) RegisterAlias(alias, target domain.SourceType) {
	if r == nil || alias == "" || target == "" {
		return
	}
	if r.aliases == nil {
		r.aliases = make(map[domain.SourceType]domain.SourceType)
	}
	r.aliases[alias] = target
}

// Capabilities resolves provider capabilities for a given source type.
func (r *SourceProviderRegistry) Capabilities(t domain.SourceType) (SourceProviderCapabilities, bool) {
	provider, ok := r.Provider(t)
	if !ok {
		return SourceProviderCapabilities{}, false
	}
	return provider.Capabilities(), true
}

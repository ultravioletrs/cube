// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"context"
	"testing"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

func TestSourceProviderRegistry_AliasResolution(t *testing.T) {
	reg := NewSourceProviderRegistry(&testSourceProvider{
		tp: domain.SourceTypeRclone,
		caps: SourceProviderCapabilities{
			SupportsList:     true,
			SupportsDownload: true,
		},
	})
	reg.RegisterAlias(domain.SourceTypeDropbox, domain.SourceTypeRclone)

	provider, ok := reg.Provider(domain.SourceTypeDropbox)
	if !ok {
		t.Fatal("expected alias provider to resolve")
	}
	if provider.Type() != domain.SourceTypeRclone {
		t.Fatalf("expected alias to resolve to rclone provider, got %q", provider.Type())
	}

	caps, ok := reg.Capabilities(domain.SourceTypeDropbox)
	if !ok {
		t.Fatal("expected capabilities to resolve via alias")
	}
	if !caps.SupportsList || !caps.SupportsDownload {
		t.Fatalf("unexpected alias capabilities: %+v", caps)
	}
}

type testSourceProvider struct {
	tp   domain.SourceType
	caps SourceProviderCapabilities
}

func (p *testSourceProvider) Type() domain.SourceType {
	return p.tp
}

func (p *testSourceProvider) Capabilities() SourceProviderCapabilities {
	return p.caps
}

func (p *testSourceProvider) ListFiles(_ context.Context, _ string, _ domain.Source) ([]SourceFile, error) {
	return nil, nil
}

func (p *testSourceProvider) DownloadRecord(_ context.Context, _ domain.Record, _ domain.Source) (string, *int, error) {
	return "", nil, nil
}

func (p *testSourceProvider) PrunesStaleRecords() bool {
	return true
}

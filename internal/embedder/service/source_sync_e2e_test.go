// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/ingest"
)

func TestSourceSyncService_NativeProvidersE2E(t *testing.T) {
	cases := []struct {
		name       string
		sourceType domain.SourceType
		config     json.RawMessage
	}{
		{
			name:       "google drive",
			sourceType: domain.SourceTypeGoogleDrive,
			config:     json.RawMessage(`{"folder_id":"root","access_token":"token"}`),
		},
		{
			name:       "s3",
			sourceType: domain.SourceTypeS3,
			config:     json.RawMessage(`{"bucket":"docs","root_path":"team/docs"}`),
		},
		{
			name:       "microsoft",
			sourceType: domain.SourceTypeMicrosoft,
			config:     json.RawMessage(`{"access_token":"token","drive_id":"drive-1","root_path":"team/docs"}`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := domain.Source{
				ID:       "src-1",
				DomainID: "domain-1",
				UserID:   "user-1",
				Type:     tc.sourceType,
				Name:     "Docs",
				Config:   tc.config,
				Status:   domain.SourceStatusActive,
			}
			sources := &sourceRepoSyncStub{source: source}
			records := &recordRepoSyncStub{}
			provider := &staticSourceProvider{
				providerType: tc.sourceType,
				files: []ingest.SourceFile{{
					ExternalID:    "file-1",
					Name:          "doc.txt",
					ExternalRef:   "docs/doc.txt",
					MimeType:      "text/plain",
					SourceVersion: "v1",
				}},
				prunesStale: tc.sourceType != domain.SourceTypeGoogleDrive,
			}
			providers := ingest.NewSourceProviderRegistry(provider)

			svc := NewSourceSyncService(sources, records, providers)
			res, err := svc.Sync(context.Background(), source.ID, source.DomainID)
			if err != nil {
				t.Fatalf("sync failed: %v", err)
			}

			if res.Discovered != 1 || res.Queued != 1 {
				t.Fatalf("expected discovered=1 queued=1, got discovered=%d queued=%d", res.Discovered, res.Queued)
			}
			if len(records.upserts) != 1 {
				t.Fatalf("expected one record upsert, got %d", len(records.upserts))
			}
			if records.upserts[0].ExternalID != "file-1" {
				t.Fatalf("unexpected record external_id: %q", records.upserts[0].ExternalID)
			}
			if sources.lastStatus != domain.SourceStatusActive {
				t.Fatalf("expected source status active after sync, got %q", sources.lastStatus)
			}
		})
	}
}

func TestSourceSyncService_AliasProviderE2E(t *testing.T) {
	source := domain.Source{
		ID:       "src-od",
		DomainID: "domain-1",
		UserID:   "user-1",
		Type:     domain.SourceTypeOneDrive,
		Name:     "OneDrive Docs",
		Config:   json.RawMessage(`{"access_token":"token","drive_id":"drive-1","root_path":"team/docs"}`),
		Status:   domain.SourceStatusActive,
	}
	sources := &sourceRepoSyncStub{source: source}
	records := &recordRepoSyncStub{}
	providers := ingest.NewSourceProviderRegistry(&staticSourceProvider{
		providerType: domain.SourceTypeMicrosoft,
		files: []ingest.SourceFile{{
			ExternalID:    "m-file-1",
			Name:          "ms.docx",
			ExternalRef:   "team/docs/ms.docx",
			MimeType:      "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			SourceVersion: "etag-1",
		}},
		prunesStale: true,
	})
	for alias, target := range domain.SourceProviderAliases() {
		providers.RegisterAlias(alias, target)
	}

	svc := NewSourceSyncService(sources, records, providers)
	res, err := svc.Sync(context.Background(), source.ID, source.DomainID)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if res.Discovered != 1 || res.Queued != 1 {
		t.Fatalf("expected discovered=1 queued=1, got discovered=%d queued=%d", res.Discovered, res.Queued)
	}
}

type staticSourceProvider struct {
	providerType domain.SourceType
	files        []ingest.SourceFile
	prunesStale  bool
}

func (p *staticSourceProvider) Type() domain.SourceType {
	return p.providerType
}

func (p *staticSourceProvider) Capabilities() ingest.SourceProviderCapabilities {
	return ingest.SourceProviderCapabilities{
		SupportsList:     true,
		SupportsDownload: true,
	}
}

func (p *staticSourceProvider) ListFiles(_ context.Context, _ string, _ domain.Source) ([]ingest.SourceFile, error) {
	return p.files, nil
}

func (p *staticSourceProvider) DownloadRecord(_ context.Context, _ domain.Record, _ domain.Source) (string, *int, error) {
	return "test", nil, nil
}

func (p *staticSourceProvider) PrunesStaleRecords() bool {
	return p.prunesStale
}

type sourceRepoSyncStub struct {
	source     domain.Source
	lastStatus domain.SourceStatus
}

func (s *sourceRepoSyncStub) Create(_ context.Context, src domain.Source) (domain.Source, error) {
	s.source = src
	return src, nil
}

func (s *sourceRepoSyncStub) GetByID(_ context.Context, id, domainID string) (domain.Source, error) {
	if s.source.ID != id || s.source.DomainID != domainID {
		return domain.Source{}, domain.ErrNotFound
	}
	return s.source, nil
}

func (s *sourceRepoSyncStub) List(_ context.Context, domainID string, _ domain.Page) (domain.SourcePage, error) {
	if s.source.DomainID != domainID {
		return domain.SourcePage{Sources: []domain.Source{}, Total: 0}, nil
	}
	return domain.SourcePage{Sources: []domain.Source{s.source}, Total: 1}, nil
}

func (s *sourceRepoSyncStub) Delete(_ context.Context, _, _ string) error {
	return nil
}

func (s *sourceRepoSyncStub) UpdateSyncResult(
	_ context.Context,
	id, domainID string,
	status domain.SourceStatus,
	lastSyncAt time.Time,
	lastSyncError *string,
) (domain.Source, error) {
	if s.source.ID != id || s.source.DomainID != domainID {
		return domain.Source{}, domain.ErrNotFound
	}
	s.source.Status = status
	s.source.LastSyncAt = &lastSyncAt
	s.source.LastSyncError = lastSyncError
	s.lastStatus = status
	return s.source, nil
}

func (s *sourceRepoSyncStub) UpdateConfig(_ context.Context, id, domainID string, config json.RawMessage) (domain.Source, error) {
	if s.source.ID != id || s.source.DomainID != domainID {
		return domain.Source{}, domain.ErrNotFound
	}
	s.source.Config = config
	return s.source, nil
}

type recordRepoSyncStub struct {
	upserts []domain.Record
}

func (r *recordRepoSyncStub) Create(_ context.Context, rec domain.Record) (domain.Record, error) {
	return rec, nil
}

func (r *recordRepoSyncStub) GetByID(_ context.Context, _, _ string) (domain.Record, error) {
	return domain.Record{}, domain.ErrNotFound
}

func (r *recordRepoSyncStub) List(_ context.Context, _ string, _ domain.RecordFilter, _ domain.Page) (domain.RecordPage, error) {
	return domain.RecordPage{Records: []domain.Record{}, Total: 0}, nil
}

func (r *recordRepoSyncStub) Delete(_ context.Context, _, _ string) error {
	return nil
}

func (r *recordRepoSyncStub) DeleteBySourceExternalIDs(_ context.Context, _, _ string, externalIDs []string) (int, error) {
	return len(externalIDs), nil
}

func (r *recordRepoSyncStub) UpsertFromSource(_ context.Context, rec domain.Record) (domain.RecordUpsertResult, error) {
	for _, existing := range r.upserts {
		if existing.SourceID == rec.SourceID && existing.ExternalID == rec.ExternalID {
			return domain.RecordUpsertResult{Record: rec, State: domain.RecordUpsertUnchanged}, nil
		}
	}
	r.upserts = append(r.upserts, rec)
	return domain.RecordUpsertResult{Record: rec, State: domain.RecordUpsertCreated}, nil
}

func (r *recordRepoSyncStub) ListQueued(_ context.Context, _ int) ([]domain.Record, error) {
	return nil, nil
}

func (r *recordRepoSyncStub) UpdateStatus(_ context.Context, _ string, _ domain.RecordStatus, _ string) error {
	return nil
}

func (r *recordRepoSyncStub) UpdateAfterIngest(_ context.Context, _ string, _ domain.IngestResult) error {
	return nil
}

var _ domain.SourceRepository = (*sourceRepoSyncStub)(nil)
var _ domain.RecordRepository = (*recordRepoSyncStub)(nil)
var _ ingest.SourceProvider = (*staticSourceProvider)(nil)

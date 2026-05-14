// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/ingest"
	embedmetrics "github.com/ultravioletrs/cube/internal/embedder/metrics"
)

type sourceSyncService struct {
	sources   domain.SourceRepository
	records   domain.RecordRepository
	providers *ingest.SourceProviderRegistry
}

// NewSourceSyncService creates a source synchronization service.
func NewSourceSyncService(
	sources domain.SourceRepository,
	records domain.RecordRepository,
	providers *ingest.SourceProviderRegistry,
) domain.SourceSyncService {
	return &sourceSyncService{
		sources:   sources,
		records:   records,
		providers: providers,
	}
}

func (s *sourceSyncService) Sync(ctx context.Context, id, domainID string) (res domain.SourceSyncResult, err error) {
	src, err := s.sources.GetByID(ctx, id, domainID)
	if err != nil {
		return domain.SourceSyncResult{}, err
	}

	provider, ok := s.providers.Provider(src.Type)
	if !ok {
		return domain.SourceSyncResult{}, fmt.Errorf("source type %q does not support sync", src.Type)
	}
	syncStartedAt := time.Now().UTC()
	defer func() {
		embedmetrics.ObserveSourceSync(
			string(src.Type),
			string(provider.Type()),
			time.Since(syncStartedAt),
			err,
		)
		embedmetrics.AddSourceSyncFiles(
			string(src.Type),
			string(provider.Type()),
			res.Discovered,
			res.Queued,
			res.Updated,
			res.Unchanged,
			res.Deleted,
		)
	}()

	now := time.Now().UTC()
	files, err := provider.ListFiles(ctx, src.UserID, src)
	if err != nil {
		msg := err.Error()
		updatedSource, markErr := s.sources.UpdateSyncResult(ctx, src.ID, src.DomainID, domain.SourceStatusError, now, &msg)
		if markErr == nil {
			src = updatedSource
		}
		return domain.SourceSyncResult{}, err
	}

	result := domain.SourceSyncResult{
		Source:     src,
		Discovered: uint64(len(files)),
	}
	liveExternalIDs := make(map[string]struct{}, len(files))

	for _, file := range files {
		externalID := strings.TrimSpace(file.ExternalID)
		if externalID == "" {
			continue
		}
		liveExternalIDs[externalID] = struct{}{}

		upsert, err := s.records.UpsertFromSource(ctx, domain.Record{
			DomainID:         src.DomainID,
			UserID:           src.UserID,
			SourceID:         src.ID,
			Name:             file.Name,
			Format:           DetectRecordFormat(file.Name, file.MimeType),
			Status:           domain.RecordStatusQueued,
			ExternalID:       externalID,
			ExternalURL:      file.ExternalURL,
			ExternalRef:      file.ExternalRef,
			MimeType:         file.MimeType,
			SourceVersion:    file.SourceVersion,
			SourceModifiedAt: file.SourceModifiedAt,
		})
		if err != nil {
			msg := err.Error()
			updatedSource, markErr := s.sources.UpdateSyncResult(ctx, src.ID, src.DomainID, domain.SourceStatusError, now, &msg)
			if markErr == nil {
				src = updatedSource
			}
			return domain.SourceSyncResult{}, err
		}

		switch upsert.State {
		case domain.RecordUpsertCreated:
			result.Queued++
		case domain.RecordUpsertUpdated:
			result.Queued++
			result.Updated++
		case domain.RecordUpsertUnchanged:
			result.Unchanged++
		}
	}

	if provider.PrunesStaleRecords() {
		toDelete, err := s.resolveStaleSourceExternalIDs(ctx, src.DomainID, src.ID, liveExternalIDs)
		if err != nil {
			msg := err.Error()
			updatedSource, markErr := s.sources.UpdateSyncResult(ctx, src.ID, src.DomainID, domain.SourceStatusError, now, &msg)
			if markErr == nil {
				src = updatedSource
			}
			return domain.SourceSyncResult{}, err
		}
		deleted, err := s.records.DeleteBySourceExternalIDs(ctx, src.DomainID, src.ID, toDelete)
		if err != nil {
			msg := err.Error()
			updatedSource, markErr := s.sources.UpdateSyncResult(ctx, src.ID, src.DomainID, domain.SourceStatusError, now, &msg)
			if markErr == nil {
				src = updatedSource
			}
			return domain.SourceSyncResult{}, err
		}
		result.Deleted = uint64(deleted)
	}

	updatedSource, updateErr := s.sources.UpdateSyncResult(ctx, src.ID, src.DomainID, domain.SourceStatusActive, now, nil)
	if updateErr == nil {
		result.Source = updatedSource
	}

	return result, nil
}

func recordFormatFromDriveFile(file ingest.DriveFile) domain.RecordFormat {
	return DetectRecordFormat(file.Name, file.MimeType)
}

func filterDriveFilesBySelection(files []ingest.DriveFile, selectedIDs []string) []ingest.DriveFile {
	if len(selectedIDs) == 0 {
		return files
	}

	selected := make(map[string]struct{}, len(selectedIDs))
	for _, id := range selectedIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		selected[id] = struct{}{}
	}
	if len(selected) == 0 {
		return files
	}

	filtered := make([]ingest.DriveFile, 0, len(files))
	for _, file := range files {
		if _, ok := selected[file.ID]; ok {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func (s *sourceSyncService) resolveStaleSourceExternalIDs(
	ctx context.Context,
	domainID, sourceID string,
	live map[string]struct{},
) ([]string, error) {
	filter := domain.RecordFilter{SourceID: &sourceID}
	const pageLimit = uint64(200)
	var offset uint64

	stale := make([]string, 0)
	for {
		page, err := s.records.List(ctx, domainID, filter, domain.Page{
			Offset: offset,
			Limit:  pageLimit,
		})
		if err != nil {
			return nil, fmt.Errorf("list records by source for stale detection: %w", err)
		}
		for _, rec := range page.Records {
			externalID := strings.TrimSpace(rec.ExternalID)
			if externalID == "" {
				continue
			}
			if _, ok := live[externalID]; ok {
				continue
			}
			stale = append(stale, externalID)
		}
		offset += uint64(len(page.Records))
		if offset >= page.Total || len(page.Records) == 0 {
			break
		}
	}

	if len(stale) == 0 {
		return nil, nil
	}
	sort.Strings(stale)
	return stale, nil
}

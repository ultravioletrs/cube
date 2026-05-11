package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/ingest"
)

type sourceSyncService struct {
	sources domain.SourceRepository
	records domain.RecordRepository
	rclone  ingest.RcloneClient
}

// NewSourceSyncService creates a source synchronization service.
func NewSourceSyncService(
	sources domain.SourceRepository,
	records domain.RecordRepository,
	rclone ingest.RcloneClient,
) domain.SourceSyncService {
	return &sourceSyncService{
		sources: sources,
		records: records,
		rclone:  rclone,
	}
}

func (s *sourceSyncService) Sync(ctx context.Context, id, userID string) (domain.SourceSyncResult, error) {
	src, err := s.sources.GetByID(ctx, id, userID)
	if err != nil {
		return domain.SourceSyncResult{}, err
	}
	if src.Type != domain.SourceTypeGoogleDrive {
		if src.Type == domain.SourceTypeRclone {
			return s.syncRclone(ctx, src, userID)
		}
		return domain.SourceSyncResult{}, fmt.Errorf("source type %q does not support sync", src.Type)
	}

	var cfg domain.GoogleDriveConfig
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return domain.SourceSyncResult{}, fmt.Errorf("decode source config: %w", err)
	}

	reader, err := ingest.NewDriveReaderFromConfig(ctx, cfg)
	if err != nil {
		return domain.SourceSyncResult{}, err
	}

	files, err := reader.ListFiles(ctx, cfg.FolderID)
	now := time.Now().UTC()
	if err != nil {
		msg := err.Error()
		updatedSource, markErr := s.sources.UpdateSyncResult(ctx, src.ID, userID, domain.SourceStatusError, now, &msg)
		if markErr == nil {
			src = updatedSource
		}
		return domain.SourceSyncResult{}, err
	}

	files, err = applyDriveSelection(ctx, reader, files, cfg)
	if err != nil {
		msg := err.Error()
		updatedSource, markErr := s.sources.UpdateSyncResult(ctx, src.ID, userID, domain.SourceStatusError, now, &msg)
		if markErr == nil {
			src = updatedSource
		}
		return domain.SourceSyncResult{}, err
	}

	result := domain.SourceSyncResult{
		Source:     src,
		Discovered: uint64(len(files)),
	}

	for _, file := range files {
		record, err := s.records.UpsertFromSource(ctx, domain.Record{
			UserID:           userID,
			SourceID:         src.ID,
			Name:             file.Name,
			Format:           recordFormatFromDriveFile(file),
			Status:           domain.RecordStatusQueued,
			ExternalID:       file.ID,
			ExternalURL:      file.WebViewLink,
			ExternalRef:      strings.Join(file.Parents, ","),
			MimeType:         file.MimeType,
			SourceVersion:    file.Version,
			SourceModifiedAt: parseTimePtr(file.ModifiedTime),
		})
		if err != nil {
			msg := err.Error()
			updatedSource, markErr := s.sources.UpdateSyncResult(ctx, src.ID, userID, domain.SourceStatusError, now, &msg)
			if markErr == nil {
				src = updatedSource
			}
			return domain.SourceSyncResult{}, err
		}
		_ = record

		switch record.State {
		case domain.RecordUpsertCreated:
			result.Queued++
		case domain.RecordUpsertUpdated:
			result.Queued++
			result.Updated++
		case domain.RecordUpsertUnchanged:
			result.Unchanged++
		}
	}

	updatedSource, err := s.sources.UpdateSyncResult(ctx, src.ID, userID, domain.SourceStatusActive, now, nil)
	if err == nil {
		result.Source = updatedSource
	} else {
		result.Source = src
	}

	return result, nil
}

func recordFormatFromDriveFile(file ingest.DriveFile) domain.RecordFormat {
	return DetectRecordFormat(file.Name, file.MimeType)
}

func parseTimePtr(value string) *time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return &t
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

func applyDriveSelection(
	ctx context.Context,
	reader *ingest.DriveReader,
	baseFiles []ingest.DriveFile,
	cfg domain.GoogleDriveConfig,
) ([]ingest.DriveFile, error) {
	baseByID := make(map[string]ingest.DriveFile, len(baseFiles))
	for _, file := range baseFiles {
		baseByID[file.ID] = file
	}

	selectedFiles := normalizeSelectionIDs(cfg.SelectedFileIDs)
	selectedFolders := normalizeSelectionIDs(cfg.SelectedFolderIDs)
	if len(selectedFiles) == 0 && len(selectedFolders) == 0 {
		return baseFiles, nil
	}

	collected := make(map[string]ingest.DriveFile)
	for _, id := range selectedFiles {
		if file, ok := baseByID[id]; ok {
			collected[file.ID] = file
		}
	}

	for _, folderID := range selectedFolders {
		files, err := reader.ListFilesRecursive(ctx, folderID)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			collected[file.ID] = file
		}
	}

	if len(collected) == 0 {
		return []ingest.DriveFile{}, nil
	}

	result := make([]ingest.DriveFile, 0, len(collected))
	for _, file := range collected {
		result = append(result, file)
	}
	return result, nil
}

func normalizeSelectionIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		id := strings.TrimSpace(value)
		if id == "" {
			continue
		}
		set[id] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	return out
}

func (s *sourceSyncService) syncRclone(
	ctx context.Context,
	src domain.Source,
	userID string,
) (domain.SourceSyncResult, error) {
	if s.rclone == nil {
		return domain.SourceSyncResult{}, fmt.Errorf("rclone source sync is not configured")
	}

	var cfg domain.RcloneConfig
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return domain.SourceSyncResult{}, fmt.Errorf("decode rclone config: %w", err)
	}

	result := domain.SourceSyncResult{Source: src}
	now := time.Now().UTC()

	files, err := s.rclone.ListFiles(ctx, ingest.RcloneListRequest{
		UserID:     userID,
		SourceID:   src.ID,
		Remote:     cfg.Remote,
		RootPath:   cfg.RootPath,
		ScopePaths: cfg.ScopePaths,
	})
	if err != nil {
		msg := err.Error()
		updatedSource, markErr := s.sources.UpdateSyncResult(ctx, src.ID, userID, domain.SourceStatusError, now, &msg)
		if markErr == nil {
			src = updatedSource
		}
		return domain.SourceSyncResult{}, err
	}
	files = filterRcloneFilesBySelection(files, cfg.SelectedPaths)

	liveExternalIDs := make(map[string]struct{}, len(files))
	for _, file := range files {
		if strings.TrimSpace(file.ExternalID) == "" {
			continue
		}
		liveExternalIDs[file.ExternalID] = struct{}{}

		upsert, err := s.records.UpsertFromSource(ctx, domain.Record{
			UserID:           userID,
			SourceID:         src.ID,
			Name:             file.Name,
			Format:           DetectRecordFormat(file.Name, file.MimeType),
			Status:           domain.RecordStatusQueued,
			ExternalID:       file.ExternalID,
			ExternalRef:      file.Path,
			MimeType:         file.MimeType,
			SourceVersion:    file.Version,
			SourceModifiedAt: file.ModifiedAt,
		})
		if err != nil {
			msg := err.Error()
			updatedSource, markErr := s.sources.UpdateSyncResult(ctx, src.ID, userID, domain.SourceStatusError, now, &msg)
			if markErr == nil {
				src = updatedSource
			}
			return domain.SourceSyncResult{}, err
		}

		result.Discovered++
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

	toDelete, err := s.resolveStaleSourceExternalIDs(ctx, userID, src.ID, liveExternalIDs)
	if err != nil {
		msg := err.Error()
		updatedSource, markErr := s.sources.UpdateSyncResult(ctx, src.ID, userID, domain.SourceStatusError, now, &msg)
		if markErr == nil {
			src = updatedSource
		}
		return domain.SourceSyncResult{}, err
	}
	deleted, err := s.records.DeleteBySourceExternalIDs(ctx, userID, src.ID, toDelete)
	if err != nil {
		msg := err.Error()
		updatedSource, markErr := s.sources.UpdateSyncResult(ctx, src.ID, userID, domain.SourceStatusError, now, &msg)
		if markErr == nil {
			src = updatedSource
		}
		return domain.SourceSyncResult{}, err
	}
	result.Deleted = uint64(deleted)

	updatedSource, err := s.sources.UpdateSyncResult(ctx, src.ID, userID, domain.SourceStatusActive, now, nil)
	if err == nil {
		result.Source = updatedSource
	}

	return result, nil
}

func filterRcloneFilesBySelection(files []ingest.RcloneFile, selectedPaths []string) []ingest.RcloneFile {
	if len(selectedPaths) == 0 {
		return files
	}
	selected := make(map[string]struct{}, len(selectedPaths))
	for _, path := range selectedPaths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		selected[path] = struct{}{}
	}
	if len(selected) == 0 {
		return files
	}

	filtered := make([]ingest.RcloneFile, 0, len(files))
	for _, file := range files {
		if _, ok := selected[strings.TrimSpace(file.Path)]; ok {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func (s *sourceSyncService) resolveStaleSourceExternalIDs(
	ctx context.Context,
	userID, sourceID string,
	live map[string]struct{},
) ([]string, error) {
	filter := domain.RecordFilter{SourceID: &sourceID}
	const pageLimit = uint64(200)
	var offset uint64

	stale := make([]string, 0)
	for {
		page, err := s.records.List(ctx, userID, filter, domain.Page{
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

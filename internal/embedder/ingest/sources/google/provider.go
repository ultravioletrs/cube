// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/ingest"
)

type sourceProvider struct{}

// NewSourceProvider creates Google Drive provider implementation.
func NewSourceProvider() ingest.SourceProvider {
	return &sourceProvider{}
}

func (p *sourceProvider) Type() domain.SourceType {
	return domain.SourceTypeGoogleDrive
}

func (p *sourceProvider) Capabilities() ingest.SourceProviderCapabilities {
	return ingest.SourceProviderCapabilities{
		SupportsList:     true,
		SupportsDownload: true,
	}
}

func (p *sourceProvider) PrunesStaleRecords() bool {
	// Keep current behavior: Drive sync does not delete stale records.
	return false
}

func (p *sourceProvider) ListFiles(
	ctx context.Context,
	_ string,
	src domain.Source,
) ([]ingest.SourceFile, error) {
	var cfg domain.GoogleDriveConfig
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decode source config: %w", err)
	}

	reader, err := ingest.NewDriveReaderFromConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	files, err := reader.ListFiles(ctx, cfg.FolderID)
	if err != nil {
		return nil, err
	}

	files, err = applyDriveSelection(ctx, reader, files, cfg)
	if err != nil {
		return nil, err
	}

	out := make([]ingest.SourceFile, 0, len(files))
	for _, file := range files {
		folderID := file.FolderID
		if folderID == "" && len(file.Parents) > 0 {
			folderID = file.Parents[0]
		}
		out = append(out, ingest.SourceFile{
			ExternalID:       file.ID,
			Name:             file.Name,
			ExternalURL:      file.WebViewLink,
			ExternalRef:      strings.Join(file.Parents, ","),
			MimeType:         file.MimeType,
			SourceVersion:    file.Version,
			SourceModifiedAt: parseRFC3339Ptr(file.ModifiedTime),
			FolderPath:       file.FolderPath,
			FolderID:         folderID,
		})
	}
	return out, nil
}

func (p *sourceProvider) DownloadRecord(
	ctx context.Context,
	rec domain.Record,
	src domain.Source,
) (string, *int, error) {
	body, err := p.DownloadRecordContent(ctx, rec, src)
	if err != nil {
		return "", nil, err
	}

	doc, err := ingest.ExtractText(ingest.FileMeta{
		ID:       rec.ExternalID,
		Name:     rec.Name,
		MimeType: rec.MimeType,
	}, body)
	if err != nil {
		return "", nil, err
	}
	return doc.Text, doc.PageCount, nil
}

func (p *sourceProvider) DownloadRecordContent(
	ctx context.Context,
	rec domain.Record,
	src domain.Source,
) ([]byte, error) {
	if rec.ExternalID == "" {
		return nil, fmt.Errorf("record %s is missing external_id", rec.ID)
	}

	var cfg domain.GoogleDriveConfig
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return nil, err
	}

	reader, err := ingest.NewDriveReaderFromConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	body, err := reader.DownloadFile(ctx, ingest.DriveFile{
		ID:       rec.ExternalID,
		MimeType: rec.MimeType,
	})
	if err != nil {
		return nil, err
	}
	return body, nil
}

func parseRFC3339Ptr(value string) *time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return &t
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
			continue
		}
		file, err := reader.GetFile(ctx, id)
		if err != nil {
			// Skip a selected file that has gone away or can't be ingested rather
			// than failing the entire sync over one stale reference.
			if errors.Is(err, ingest.ErrDriveNotFound) || errors.Is(err, ingest.ErrUnsupportedDriveFile) {
				continue
			}
			return nil, err
		}
		collected[file.ID] = file
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

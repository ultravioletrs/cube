// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package rclone

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/ingest"
)

type sourceProvider struct {
	rclone ingest.RcloneClient
}

// NewSourceProvider creates rclone provider implementation.
func NewSourceProvider(rclone ingest.RcloneClient) ingest.SourceProvider {
	return &sourceProvider{rclone: rclone}
}

func (p *sourceProvider) Type() domain.SourceType {
	return domain.SourceTypeRclone
}

func (p *sourceProvider) Capabilities() ingest.SourceProviderCapabilities {
	return ingest.SourceProviderCapabilities{
		SupportsList:     true,
		SupportsDownload: true,
		SupportsBrowse:   true,
	}
}

func (p *sourceProvider) PrunesStaleRecords() bool {
	return true
}

func (p *sourceProvider) ListFiles(
	ctx context.Context,
	userID string,
	src domain.Source,
) ([]ingest.SourceFile, error) {
	if p.rclone == nil {
		return nil, fmt.Errorf("rclone source sync is not configured")
	}

	var cfg domain.RcloneConfig
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decode rclone config: %w", err)
	}

	files, err := p.rclone.ListFiles(ctx, ingest.RcloneListRequest{
		UserID:     userID,
		SourceID:   src.ID,
		Remote:     cfg.Remote,
		RootPath:   cfg.RootPath,
		ScopePaths: cfg.ScopePaths,
	})
	if err != nil {
		return nil, err
	}
	files = filterRcloneFilesBySelection(files, cfg.SelectedPaths)

	out := make([]ingest.SourceFile, 0, len(files))
	for _, file := range files {
		out = append(out, ingest.SourceFile{
			ExternalID:       file.ExternalID,
			Name:             file.Name,
			ExternalRef:      file.Path,
			MimeType:         file.MimeType,
			SourceVersion:    file.Version,
			SourceModifiedAt: file.ModifiedAt,
		})
	}
	return out, nil
}

func (p *sourceProvider) DownloadRecord(
	ctx context.Context,
	rec domain.Record,
	src domain.Source,
) (string, *int, error) {
	return ingest.DownloadFromRcloneSource(ctx, rec, src)
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

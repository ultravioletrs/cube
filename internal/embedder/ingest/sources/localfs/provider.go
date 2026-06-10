// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

// Package localfs implements the SourceProvider for direct file uploads
// (SourceTypeLocalFS). Unlike the cloud providers it does not discover files via
// a remote API: records are created by the upload handler, so ListFiles is a
// no-op and the provider only serves content downloads.
package localfs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/ingest"
	objstore "github.com/ultravioletrs/cube/internal/embedder/storage"
)

type sourceProvider struct {
	store objstore.Store
}

// NewSourceProvider creates the local-filesystem/upload provider. store is the
// object storage backend that holds direct uploads; it may be nil only if no
// object-store-backed uploads exist (legacy upload_dir records still work).
func NewSourceProvider(store objstore.Store) ingest.SourceProvider {
	return &sourceProvider{store: store}
}

func (p *sourceProvider) Type() domain.SourceType {
	return domain.SourceTypeLocalFS
}

func (p *sourceProvider) Capabilities() ingest.SourceProviderCapabilities {
	return ingest.SourceProviderCapabilities{
		SupportsDownload: true,
	}
}

func (p *sourceProvider) PrunesStaleRecords() bool {
	return false
}

// ListFiles returns no files: local uploads are pushed in by the upload handler,
// not discovered. Returning empty makes a stray Sync a safe no-op.
func (p *sourceProvider) ListFiles(
	_ context.Context,
	_ string,
	_ domain.Source,
) ([]ingest.SourceFile, error) {
	return nil, nil
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

type uploadConfig struct {
	Kind      string `json:"kind,omitempty"`
	UploadDir string `json:"upload_dir"`
}

func (p *sourceProvider) DownloadRecordContent(
	ctx context.Context,
	rec domain.Record,
	src domain.Source,
) ([]byte, error) {
	if rec.ExternalID == "" {
		return nil, fmt.Errorf("record %s is missing external_id", rec.ID)
	}

	// New direct uploads store external_id as an object key.
	if strings.Contains(rec.ExternalID, "/") {
		if p.store == nil {
			return nil, fmt.Errorf("object storage is not configured")
		}
		return p.store.Get(ctx, rec.ExternalID)
	}

	// Legacy local_fs records might still point to upload_dir + file name.
	var cfg uploadConfig
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.UploadDir) == "" {
		return nil, fmt.Errorf("local source %s is missing upload_dir config", src.ID)
	}

	fileName := filepath.Base(rec.ExternalID)
	if fileName != rec.ExternalID {
		return nil, fmt.Errorf("invalid local upload file id")
	}

	path := filepath.Join(cfg.UploadDir, rec.UserID, fileName)
	return os.ReadFile(path)
}

// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

type googleDriveSourceProvider struct{}

// NewGoogleDriveSourceProvider creates Google Drive provider implementation.
func NewGoogleDriveSourceProvider() SourceProvider {
	return &googleDriveSourceProvider{}
}

func (p *googleDriveSourceProvider) Type() domain.SourceType {
	return domain.SourceTypeGoogleDrive
}

func (p *googleDriveSourceProvider) Capabilities() SourceProviderCapabilities {
	return SourceProviderCapabilities{
		SupportsList:     true,
		SupportsDownload: true,
	}
}

func (p *googleDriveSourceProvider) PrunesStaleRecords() bool {
	// Keep current behavior: Drive sync does not delete stale records.
	return false
}

func (p *googleDriveSourceProvider) ListFiles(
	ctx context.Context,
	_ string,
	src domain.Source,
) ([]SourceFile, error) {
	var cfg domain.GoogleDriveConfig
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decode source config: %w", err)
	}

	reader, err := NewDriveReaderFromConfig(ctx, cfg)
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

	out := make([]SourceFile, 0, len(files))
	for _, file := range files {
		out = append(out, SourceFile{
			ExternalID:       file.ID,
			Name:             file.Name,
			ExternalURL:      file.WebViewLink,
			ExternalRef:      strings.Join(file.Parents, ","),
			MimeType:         file.MimeType,
			SourceVersion:    file.Version,
			SourceModifiedAt: parseRFC3339Ptr(file.ModifiedTime),
		})
	}
	return out, nil
}

func (p *googleDriveSourceProvider) DownloadRecord(
	ctx context.Context,
	rec domain.Record,
	src domain.Source,
) (string, *int, error) {
	if rec.ExternalID == "" {
		return "", nil, fmt.Errorf("record %s is missing external_id", rec.ID)
	}

	var cfg domain.GoogleDriveConfig
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return "", nil, err
	}

	reader, err := NewDriveReaderFromConfig(ctx, cfg)
	if err != nil {
		return "", nil, err
	}

	body, err := reader.DownloadFile(ctx, DriveFile{
		ID:       rec.ExternalID,
		MimeType: rec.MimeType,
	})
	if err != nil {
		return "", nil, err
	}

	doc, err := ExtractText(DriveFile{
		ID:       rec.ExternalID,
		Name:     rec.Name,
		MimeType: rec.MimeType,
	}, body)
	if err != nil {
		return "", nil, err
	}
	return doc.Text, doc.PageCount, nil
}

type rcloneSourceProvider struct {
	rclone RcloneClient
}

// NewRcloneSourceProvider creates rclone provider implementation.
func NewRcloneSourceProvider(rclone RcloneClient) SourceProvider {
	return &rcloneSourceProvider{rclone: rclone}
}

func (p *rcloneSourceProvider) Type() domain.SourceType {
	return domain.SourceTypeRclone
}

func (p *rcloneSourceProvider) Capabilities() SourceProviderCapabilities {
	return SourceProviderCapabilities{
		SupportsList:     true,
		SupportsDownload: true,
		SupportsBrowse:   true,
	}
}

func (p *rcloneSourceProvider) PrunesStaleRecords() bool {
	return true
}

func (p *rcloneSourceProvider) ListFiles(
	ctx context.Context,
	userID string,
	src domain.Source,
) ([]SourceFile, error) {
	if p.rclone == nil {
		return nil, fmt.Errorf("rclone source sync is not configured")
	}

	var cfg domain.RcloneConfig
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decode rclone config: %w", err)
	}

	files, err := p.rclone.ListFiles(ctx, RcloneListRequest{
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

	out := make([]SourceFile, 0, len(files))
	for _, file := range files {
		out = append(out, SourceFile{
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

func (p *rcloneSourceProvider) DownloadRecord(
	ctx context.Context,
	rec domain.Record,
	src domain.Source,
) (string, *int, error) {
	return downloadFromRcloneSource(ctx, rec, src)
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
	reader *DriveReader,
	baseFiles []DriveFile,
	cfg domain.GoogleDriveConfig,
) ([]DriveFile, error) {
	baseByID := make(map[string]DriveFile, len(baseFiles))
	for _, file := range baseFiles {
		baseByID[file.ID] = file
	}

	selectedFiles := normalizeSelectionIDs(cfg.SelectedFileIDs)
	selectedFolders := normalizeSelectionIDs(cfg.SelectedFolderIDs)
	if len(selectedFiles) == 0 && len(selectedFolders) == 0 {
		return baseFiles, nil
	}

	collected := make(map[string]DriveFile)
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
		return []DriveFile{}, nil
	}

	result := make([]DriveFile, 0, len(collected))
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

func filterRcloneFilesBySelection(files []RcloneFile, selectedPaths []string) []RcloneFile {
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

	filtered := make([]RcloneFile, 0, len(files))
	for _, file := range files {
		if _, ok := selected[strings.TrimSpace(file.Path)]; ok {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

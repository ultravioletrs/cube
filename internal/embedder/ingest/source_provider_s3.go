// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

type s3SourceProvider struct{}

// NewS3SourceProvider creates native S3 provider implementation.
func NewS3SourceProvider() SourceProvider {
	return &s3SourceProvider{}
}

func (p *s3SourceProvider) Type() domain.SourceType {
	return domain.SourceTypeS3
}

func (p *s3SourceProvider) Capabilities() SourceProviderCapabilities {
	return SourceProviderCapabilities{
		SupportsList:     true,
		SupportsDownload: true,
		SupportsBrowse:   true,
	}
}

func (p *s3SourceProvider) PrunesStaleRecords() bool {
	return true
}

func (p *s3SourceProvider) ListFiles(
	ctx context.Context,
	_ string,
	src domain.Source,
) ([]SourceFile, error) {
	var cfg domain.S3Config
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decode s3 config: %w", err)
	}
	return listS3Files(ctx, cfg)
}

func (p *s3SourceProvider) DownloadRecord(
	ctx context.Context,
	rec domain.Record,
	src domain.Source,
) (string, *int, error) {
	if rec.ExternalID == "" {
		return "", nil, fmt.Errorf("record %s is missing external_id", rec.ID)
	}

	var cfg domain.S3Config
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return "", nil, fmt.Errorf("decode s3 config: %w", err)
	}

	client, err := newS3Client(cfg)
	if err != nil {
		return "", nil, err
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	obj, err := client.GetObject(ctx, bucket, normalizeRclonePath(rec.ExternalID), minio.GetObjectOptions{})
	if err != nil {
		return "", nil, err
	}
	defer obj.Close()

	body, err := io.ReadAll(obj)
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

// S3BrowseEntry is a normalized object/prefix entry returned by browse previews.
type S3BrowseEntry struct {
	Name       string
	Path       string
	IsDir      bool
	MimeType   string
	Size       int64
	ModifiedAt *time.Time
}

// BrowseS3Path browses one path level in an S3 bucket using configured credentials.
func BrowseS3Path(ctx context.Context, cfg domain.S3Config, currentPath string) ([]S3BrowseEntry, error) {
	client, err := newS3Client(cfg)
	if err != nil {
		return nil, err
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	currentPath = normalizeRclonePath(currentPath)

	root := normalizeRclonePath(cfg.RootPath)
	if root != "" && !isPathWithinRoot(root, currentPath) {
		return nil, fmt.Errorf("browse path %q is outside root_path %q", currentPath, root)
	}

	prefix := currentPath
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	children := make(map[string]S3BrowseEntry)
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: false,
	}
	for obj := range client.ListObjects(ctx, bucket, opts) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		trimmed := strings.TrimPrefix(obj.Key, prefix)
		trimmed = strings.TrimPrefix(trimmed, "/")
		if trimmed == "" {
			continue
		}

		segment := trimmed
		if idx := strings.Index(segment, "/"); idx >= 0 {
			segment = segment[:idx]
			childPath := normalizeRclonePath(path.Join(currentPath, segment))
			children[childPath] = S3BrowseEntry{
				Name:  segment,
				Path:  childPath,
				IsDir: true,
			}
			continue
		}

		childPath := normalizeRclonePath(path.Join(currentPath, segment))
		modified := obj.LastModified.UTC()
		children[childPath] = S3BrowseEntry{
			Name:       segment,
			Path:       childPath,
			IsDir:      false,
			MimeType:   strings.TrimSpace(obj.ContentType),
			Size:       obj.Size,
			ModifiedAt: &modified,
		}
	}

	out := make([]S3BrowseEntry, 0, len(children))
	for _, entry := range children {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

func listS3Files(ctx context.Context, cfg domain.S3Config) ([]SourceFile, error) {
	client, err := newS3Client(cfg)
	if err != nil {
		return nil, err
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	if bucket == "" {
		return nil, fmt.Errorf("s3 bucket is required")
	}

	rootPath := normalizeRclonePath(cfg.RootPath)
	scopes, err := normalizeRcloneScopes(rootPath, cfg.ScopePaths)
	if err != nil {
		return nil, err
	}
	if len(scopes) == 0 {
		scopes = []string{rootPath}
	}

	aggregated := make(map[string]SourceFile)
	for _, scope := range scopes {
		prefix := scope
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		opts := minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: true,
		}
		for obj := range client.ListObjects(ctx, bucket, opts) {
			if obj.Err != nil {
				return nil, obj.Err
			}
			key := normalizeRclonePath(obj.Key)
			if key == "" {
				continue
			}

			modified := obj.LastModified.UTC()
			file := SourceFile{
				ExternalID:       key,
				Name:             path.Base(key),
				ExternalRef:      key,
				MimeType:         strings.TrimSpace(obj.ContentType),
				SourceVersion:    strings.TrimSpace(obj.ETag),
				SourceModifiedAt: &modified,
			}
			existing, ok := aggregated[file.ExternalID]
			if !ok || sourceFileNewer(file, existing) {
				aggregated[file.ExternalID] = file
			}
		}
	}

	files := make([]SourceFile, 0, len(aggregated))
	for _, file := range aggregated {
		files = append(files, file)
	}
	files = filterSourceFilesBySelectedPaths(files, cfg.SelectedPaths)
	sort.Slice(files, func(i, j int) bool {
		return files[i].ExternalID < files[j].ExternalID
	})
	return files, nil
}

// ListS3FilesPreview lists S3 files for source configuration preview API.
func ListS3FilesPreview(ctx context.Context, cfg domain.S3Config) ([]SourceFile, error) {
	return listS3Files(ctx, cfg)
}

func newS3Client(cfg domain.S3Config) (*minio.Client, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = "s3.amazonaws.com"
	}
	secure := true
	if cfg.UseSSL != nil {
		secure = *cfg.UseSSL
	}
	pathStyle := true
	if cfg.PathStyle != nil {
		pathStyle = *cfg.PathStyle
	}

	accessKeyID := strings.TrimSpace(cfg.AccessKeyID)
	secretAccessKey := strings.TrimSpace(cfg.SecretAccessKey)
	sessionToken := strings.TrimSpace(cfg.SessionToken)
	var creds *credentials.Credentials
	switch {
	case accessKeyID != "" && secretAccessKey != "":
		creds = credentials.NewStaticV4(accessKeyID, secretAccessKey, sessionToken)
	case accessKeyID == "" && secretAccessKey == "":
		creds = credentials.NewChainCredentials([]credentials.Provider{
			&credentials.EnvAWS{},
			&credentials.EnvMinio{},
			&credentials.IAM{},
		})
	default:
		return nil, fmt.Errorf("s3 access_key_id and secret_access_key must both be set or both omitted")
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:        creds,
		Secure:       secure,
		Region:       strings.TrimSpace(cfg.Region),
		BucketLookup: s3BucketLookup(pathStyle),
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func s3BucketLookup(forcePathStyle bool) minio.BucketLookupType {
	if forcePathStyle {
		return minio.BucketLookupPath
	}
	return minio.BucketLookupAuto
}

func sourceFileNewer(a, b SourceFile) bool {
	if a.SourceModifiedAt != nil && b.SourceModifiedAt != nil {
		if a.SourceModifiedAt.After(*b.SourceModifiedAt) {
			return true
		}
		if b.SourceModifiedAt.After(*a.SourceModifiedAt) {
			return false
		}
	}
	if a.SourceVersion != b.SourceVersion {
		return a.SourceVersion > b.SourceVersion
	}
	return a.ExternalRef > b.ExternalRef
}

func filterSourceFilesBySelectedPaths(files []SourceFile, selectedPaths []string) []SourceFile {
	if len(selectedPaths) == 0 {
		return files
	}
	selected := make(map[string]struct{}, len(selectedPaths))
	for _, p := range selectedPaths {
		p = normalizeRclonePath(p)
		if p == "" {
			continue
		}
		selected[p] = struct{}{}
	}
	if len(selected) == 0 {
		return files
	}

	filtered := make([]SourceFile, 0, len(files))
	for _, file := range files {
		if _, ok := selected[normalizeRclonePath(file.ExternalRef)]; ok {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func isPathWithinRoot(rootPath, scopedPath string) bool {
	rootPath = normalizeRclonePath(rootPath)
	scopedPath = normalizeRclonePath(scopedPath)
	if rootPath == "" || scopedPath == "" {
		return true
	}
	rootAbs := "/" + rootPath
	scopeAbs := "/" + scopedPath
	return scopeAbs == rootAbs || strings.HasPrefix(scopeAbs, rootAbs+"/")
}

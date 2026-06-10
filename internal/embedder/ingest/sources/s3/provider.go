// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package s3

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
	"github.com/ultravioletrs/cube/internal/embedder/ingest"
	"github.com/ultravioletrs/cube/internal/embedder/ingest/sourcepath"
)

type sourceProvider struct{}

// NewSourceProvider creates native S3 provider implementation.
func NewSourceProvider() ingest.SourceProvider {
	return &sourceProvider{}
}

func (p *sourceProvider) Type() domain.SourceType {
	return domain.SourceTypeS3
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
	_ string,
	src domain.Source,
) ([]ingest.SourceFile, error) {
	var cfg domain.S3Config
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decode s3 config: %w", err)
	}
	return listS3Files(ctx, cfg)
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

	var cfg domain.S3Config
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decode s3 config: %w", err)
	}

	client, err := newS3Client(cfg)
	if err != nil {
		return nil, err
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	obj, err := client.GetObject(ctx, bucket, sourcepath.Normalize(rec.ExternalID), minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()

	body, err := io.ReadAll(obj)
	if err != nil {
		return nil, err
	}
	return body, nil
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

// S3BrowsePage is one bounded page of a flat S3 path browse.
type S3BrowsePage struct {
	Entries       []S3BrowseEntry
	NextPageToken string
	HasMore       bool
}

const (
	defaultBrowsePageSize = 100
	maxBrowsePageSize     = 500
)

// BrowseS3Path browses one path level in an S3 bucket using configured credentials.
func BrowseS3Path(ctx context.Context, cfg domain.S3Config, currentPath string) ([]S3BrowseEntry, error) {
	client, err := newS3Client(cfg)
	if err != nil {
		return nil, err
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	currentPath = sourcepath.Normalize(currentPath)

	root := sourcepath.Normalize(cfg.RootPath)
	if root != "" && !sourcepath.IsWithinRoot(root, currentPath) {
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
		entry, ok := s3BrowseEntryFromObject(currentPath, prefix, obj)
		if !ok {
			continue
		}
		children[entry.Path] = entry
	}

	out := make([]S3BrowseEntry, 0, len(children))
	for _, entry := range children {
		out = append(out, entry)
	}
	sortS3BrowseEntries(out)
	return out, nil
}

// BrowseS3PathPage browses one path level in bounded pages. pageToken is the
// previously returned S3 key/prefix to start after.
func BrowseS3PathPage(
	ctx context.Context,
	cfg domain.S3Config,
	currentPath string,
	pageSize int,
	pageToken string,
) (S3BrowsePage, error) {
	client, err := newS3Client(cfg)
	if err != nil {
		return S3BrowsePage{}, err
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	currentPath = sourcepath.Normalize(currentPath)

	root := sourcepath.Normalize(cfg.RootPath)
	if root != "" && !sourcepath.IsWithinRoot(root, currentPath) {
		return S3BrowsePage{}, fmt.Errorf("browse path %q is outside root_path %q", currentPath, root)
	}

	prefix := currentPath
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	pageToken = strings.TrimSpace(pageToken)
	if pageToken != "" && prefix != "" && !strings.HasPrefix(pageToken, prefix) {
		return S3BrowsePage{}, fmt.Errorf("page_token is outside browse path")
	}
	if pageSize <= 0 {
		pageSize = defaultBrowsePageSize
	}
	if pageSize > maxBrowsePageSize {
		pageSize = maxBrowsePageSize
	}

	entries := make([]S3BrowseEntry, 0, pageSize)
	nextToken := ""
	hasMore := false
	opts := minio.ListObjectsOptions{
		Prefix:     prefix,
		Recursive:  false,
		MaxKeys:    pageSize + 1,
		StartAfter: pageToken,
	}
	for obj := range client.ListObjectsIter(ctx, bucket, opts) {
		if obj.Err != nil {
			return S3BrowsePage{}, obj.Err
		}
		entry, ok := s3BrowseEntryFromObject(currentPath, prefix, obj)
		if !ok {
			continue
		}
		if len(entries) >= pageSize {
			hasMore = true
			break
		}
		entries = append(entries, entry)
		nextToken = strings.TrimSpace(obj.Key)
	}

	if !hasMore {
		nextToken = ""
	}

	sortS3BrowseEntries(entries)
	return S3BrowsePage{
		Entries:       entries,
		NextPageToken: nextToken,
		HasMore:       hasMore,
	}, nil
}

func s3BrowseEntryFromObject(currentPath, prefix string, obj minio.ObjectInfo) (S3BrowseEntry, bool) {
	trimmed := strings.TrimPrefix(obj.Key, prefix)
	trimmed = strings.TrimPrefix(trimmed, "/")
	if trimmed == "" {
		return S3BrowseEntry{}, false
	}

	segment := trimmed
	if idx := strings.Index(segment, "/"); idx >= 0 {
		segment = segment[:idx]
		childPath := sourcepath.Normalize(path.Join(currentPath, segment))
		return S3BrowseEntry{
			Name:  segment,
			Path:  childPath,
			IsDir: true,
		}, true
	}

	childPath := sourcepath.Normalize(path.Join(currentPath, segment))
	modified := obj.LastModified.UTC()
	return S3BrowseEntry{
		Name:       segment,
		Path:       childPath,
		IsDir:      false,
		MimeType:   ingest.NormalizeFileMIMEType(segment, obj.ContentType),
		Size:       obj.Size,
		ModifiedAt: &modified,
	}, true
}

func sortS3BrowseEntries(entries []S3BrowseEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Path < entries[j].Path
	})
}

func listS3Files(ctx context.Context, cfg domain.S3Config) ([]ingest.SourceFile, error) {
	client, err := newS3Client(cfg)
	if err != nil {
		return nil, err
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	if bucket == "" {
		return nil, fmt.Errorf("s3 bucket is required")
	}

	rootPath := sourcepath.Normalize(cfg.RootPath)
	scopes, err := sourcepath.NormalizeScopes(rootPath, cfg.ScopePaths)
	if err != nil {
		return nil, err
	}
	if len(scopes) == 0 {
		scopes = []string{rootPath}
	}

	aggregated := make(map[string]ingest.SourceFile)
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
			key := sourcepath.Normalize(obj.Key)
			if key == "" {
				continue
			}

			modified := obj.LastModified.UTC()
			name := path.Base(key)
			file := ingest.SourceFile{
				ExternalID:       key,
				Name:             name,
				ExternalRef:      key,
				MimeType:         ingest.NormalizeFileMIMEType(name, obj.ContentType),
				SourceVersion:    strings.TrimSpace(obj.ETag),
				SourceModifiedAt: &modified,
			}
			existing, ok := aggregated[file.ExternalID]
			if !ok || sourceFileNewer(file, existing) {
				aggregated[file.ExternalID] = file
			}
		}
	}

	files := make([]ingest.SourceFile, 0, len(aggregated))
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
func ListS3FilesPreview(ctx context.Context, cfg domain.S3Config) ([]ingest.SourceFile, error) {
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

func sourceFileNewer(a, b ingest.SourceFile) bool {
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

func filterSourceFilesBySelectedPaths(files []ingest.SourceFile, selectedPaths []string) []ingest.SourceFile {
	selectedPaths = sourcepath.NormalizeList(selectedPaths)
	if len(selectedPaths) == 0 {
		return files
	}

	filtered := make([]ingest.SourceFile, 0, len(files))
	for _, file := range files {
		if sourcepath.SelectionContains(selectedPaths, file.ExternalRef) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/ingest/sourcepath"
)

type sourcesService struct {
	repo domain.SourceRepository
}

// NewSourcesService returns a SourceService backed by the given repository.
func NewSourcesService(repo domain.SourceRepository) domain.SourceService {
	return &sourcesService{repo: repo}
}

func (s *sourcesService) Create(ctx context.Context, src domain.Source) (domain.Source, error) {
	if src.DomainID == "" {
		return domain.Source{}, fmt.Errorf("domain_id is required")
	}
	if src.UserID == "" {
		return domain.Source{}, fmt.Errorf("user_id is required")
	}
	if src.Name == "" {
		return domain.Source{}, fmt.Errorf("name is required")
	}
	if src.Type == "" {
		return domain.Source{}, fmt.Errorf("source_type is required")
	}
	if !validSourceType(src.Type) {
		return domain.Source{}, fmt.Errorf("unsupported source_type %q", src.Type)
	}
	if src.Config == nil {
		src.Config = json.RawMessage("{}")
	}
	if src.Type == domain.SourceTypeGoogleDrive {
		sanitized, err := sanitizeGoogleDriveConfig(src.Config)
		if err != nil {
			return domain.Source{}, err
		}
		src.Config = sanitized
	}
	if src.Type == domain.SourceTypeS3 {
		sanitized, err := sanitizeS3Config(src.Config)
		if err != nil {
			return domain.Source{}, err
		}
		src.Config = sanitized
	}
	if src.Type == domain.SourceTypeMicrosoft ||
		src.Type == domain.SourceTypeOneDrive ||
		src.Type == domain.SourceTypeSharePoint {
		sanitized, err := sanitizeMicrosoftConfig(src.Config)
		if err != nil {
			return domain.Source{}, err
		}
		src.Config = sanitized
	}
	if src.Status == "" {
		src.Status = domain.SourceStatusActive
	}
	return s.repo.Create(ctx, src)
}

func (s *sourcesService) GetByID(ctx context.Context, id, domainID string) (domain.Source, error) {
	return s.repo.GetByID(ctx, id, domainID)
}

func (s *sourcesService) List(ctx context.Context, domainID string, p domain.Page) (domain.SourcePage, error) {
	return s.repo.List(ctx, domainID, p)
}

func (s *sourcesService) Delete(ctx context.Context, id, domainID string) error {
	return s.repo.Delete(ctx, id, domainID)
}

func (s *sourcesService) UpdateGoogleDriveCredentials(
	ctx context.Context,
	id, domainID string,
	update domain.GoogleDriveCredentialUpdate,
) (domain.Source, error) {
	src, err := s.repo.GetByID(ctx, id, domainID)
	if err != nil {
		return domain.Source{}, err
	}
	if src.Type != domain.SourceTypeGoogleDrive {
		return domain.Source{}, fmt.Errorf("source type %q does not support credential updates", src.Type)
	}

	var cfg domain.GoogleDriveConfig
	if len(src.Config) > 0 {
		if err := json.Unmarshal(src.Config, &cfg); err != nil {
			return domain.Source{}, fmt.Errorf("decode source config: %w", err)
		}
	}

	accessToken := normalizeOAuthValue(update.AccessToken)
	if accessToken != "" {
		cfg.AccessToken = accessToken
	}
	refreshToken := normalizeOAuthValue(update.RefreshToken)
	if refreshToken != "" {
		cfg.RefreshToken = refreshToken
	}
	clientID := normalizeOAuthValue(update.ClientID)
	if clientID != "" {
		cfg.ClientID = clientID
	}
	clientSecret := normalizeOAuthValue(update.ClientSecret)
	if clientSecret != "" {
		cfg.ClientSecret = clientSecret
	}

	if strings.TrimSpace(cfg.AccessToken) == "" {
		return domain.Source{}, fmt.Errorf("access_token is required")
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		return domain.Source{}, fmt.Errorf("encode source config: %w", err)
	}
	return s.repo.UpdateConfig(ctx, id, domainID, raw)
}

func (s *sourcesService) UpdateGoogleDriveSelection(
	ctx context.Context,
	id, domainID string,
	update domain.GoogleDriveSelectionUpdate,
) (domain.Source, error) {
	src, err := s.repo.GetByID(ctx, id, domainID)
	if err != nil {
		return domain.Source{}, err
	}
	if src.Type != domain.SourceTypeGoogleDrive {
		return domain.Source{}, fmt.Errorf("source type %q does not support selection updates", src.Type)
	}

	var cfg domain.GoogleDriveConfig
	if len(src.Config) > 0 {
		if err := json.Unmarshal(src.Config, &cfg); err != nil {
			return domain.Source{}, fmt.Errorf("decode source config: %w", err)
		}
	}

	cfg.SelectedFileIDs = normalizeIDList(update.SelectedFileIDs)
	cfg.SelectedFolderIDs = normalizeIDList(update.SelectedFolderIDs)

	raw, err := json.Marshal(cfg)
	if err != nil {
		return domain.Source{}, fmt.Errorf("encode source config: %w", err)
	}
	return s.repo.UpdateConfig(ctx, id, domainID, raw)
}

func sanitizeGoogleDriveConfig(raw json.RawMessage) (json.RawMessage, error) {
	var cfg domain.GoogleDriveConfig
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("decode source config: %w", err)
		}
	}

	cfg.AccessToken = normalizeOAuthValue(cfg.AccessToken)
	cfg.RefreshToken = normalizeOAuthValue(cfg.RefreshToken)
	cfg.ClientID = normalizeOAuthValue(cfg.ClientID)
	cfg.ClientSecret = normalizeOAuthValue(cfg.ClientSecret)
	cfg.FolderID = normalizeOAuthValue(cfg.FolderID)
	cfg.FolderLink = strings.TrimSpace(cfg.FolderLink)
	cfg.SelectedFileIDs = normalizeIDList(cfg.SelectedFileIDs)
	cfg.SelectedFolderIDs = normalizeIDList(cfg.SelectedFolderIDs)

	sanitized, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("encode source config: %w", err)
	}
	return sanitized, nil
}

func normalizeOAuthValue(value string) string {
	cleaned := strings.TrimSpace(value)
	for {
		next := strings.TrimSpace(strings.TrimSuffix(cleaned, ","))
		next = strings.Trim(next, `"'`)
		next = strings.TrimSpace(next)
		if next == cleaned {
			return cleaned
		}
		cleaned = next
	}
}

func normalizeIDList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	unique := make(map[string]struct{}, len(values))
	for _, value := range values {
		id := normalizeOAuthValue(value)
		if id == "" {
			continue
		}
		unique[id] = struct{}{}
	}

	if len(unique) == 0 {
		return nil
	}

	result := make([]string, 0, len(unique))
	for id := range unique {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

func validSourceType(t domain.SourceType) bool {
	return domain.IsSupportedSourceType(t)
}

func sanitizeS3Config(raw json.RawMessage) (json.RawMessage, error) {
	var cfg domain.S3Config
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("decode s3 config: %w", err)
		}
	}

	endpoint, inferredTLS, err := normalizeS3Endpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	cfg.Endpoint = endpoint
	cfg.Region = strings.TrimSpace(cfg.Region)
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	cfg.Bucket = strings.TrimSpace(cfg.Bucket)
	cfg.AccessKeyID = strings.TrimSpace(cfg.AccessKeyID)
	cfg.SecretAccessKey = strings.TrimSpace(cfg.SecretAccessKey)
	cfg.SessionToken = strings.TrimSpace(cfg.SessionToken)
	cfg.RootPath = sourcepath.Normalize(cfg.RootPath)
	cfg.ScopePaths = sourcepath.NormalizeList(cfg.ScopePaths)
	cfg.SelectedPaths = sourcepath.NormalizeList(cfg.SelectedPaths)
	cfg.ConfigRef = strings.TrimSpace(cfg.ConfigRef)

	if cfg.UseSSL == nil {
		cfg.UseSSL = boolPtr(true)
	}
	if inferredTLS != nil {
		cfg.UseSSL = inferredTLS
	}
	if cfg.PathStyle == nil {
		cfg.PathStyle = boolPtr(true)
	}

	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3 bucket is required")
	}
	if cfg.RootPath == "" && len(cfg.ScopePaths) == 0 && len(cfg.SelectedPaths) == 0 {
		return nil, fmt.Errorf("s3 root_path, scope_paths or selected_paths is required")
	}
	if cfg.AccessKeyID == "" && cfg.SecretAccessKey != "" {
		return nil, fmt.Errorf("s3 access_key_id is required when secret_access_key is set")
	}
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("s3 secret_access_key is required when access_key_id is set")
	}
	if err := sourcepath.ValidateScopesWithinRoot(cfg.RootPath, cfg.ScopePaths); err != nil {
		return nil, fmt.Errorf("invalid s3 scope_paths: %w", err)
	}
	if err := sourcepath.ValidateScopesWithinRoot(cfg.RootPath, cfg.SelectedPaths); err != nil {
		return nil, fmt.Errorf("invalid s3 selected_paths: %w", err)
	}

	sanitized, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("encode s3 config: %w", err)
	}
	return sanitized, nil
}

func normalizeS3Endpoint(raw string) (string, *bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "s3.amazonaws.com", nil, nil
	}
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", nil, fmt.Errorf("invalid s3 endpoint: %w", err)
		}
		if parsed.Host == "" {
			return "", nil, fmt.Errorf("invalid s3 endpoint: host is required")
		}
		if parsed.Path != "" && parsed.Path != "/" {
			return "", nil, fmt.Errorf("invalid s3 endpoint: path is not allowed")
		}
		if parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
			return "", nil, fmt.Errorf("invalid s3 endpoint: query, fragment and userinfo are not allowed")
		}
		switch parsed.Scheme {
		case "http":
			return parsed.Host, boolPtr(false), nil
		case "https":
			return parsed.Host, boolPtr(true), nil
		default:
			return "", nil, fmt.Errorf("invalid s3 endpoint scheme %q", parsed.Scheme)
		}
	}
	return raw, nil, nil
}

func boolPtr(v bool) *bool {
	return &v
}

func sanitizeMicrosoftConfig(raw json.RawMessage) (json.RawMessage, error) {
	var cfg domain.MicrosoftConfig
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("decode microsoft config: %w", err)
		}
	}

	cfg.TenantID = normalizeOAuthValue(cfg.TenantID)
	cfg.ClientID = normalizeOAuthValue(cfg.ClientID)
	cfg.ClientSecret = normalizeOAuthValue(cfg.ClientSecret)
	cfg.AccessToken = normalizeOAuthValue(cfg.AccessToken)
	cfg.RefreshToken = normalizeOAuthValue(cfg.RefreshToken)
	cfg.DriveID = strings.TrimSpace(cfg.DriveID)
	cfg.SiteID = strings.TrimSpace(cfg.SiteID)
	cfg.RootPath = sourcepath.Normalize(cfg.RootPath)
	cfg.ScopePaths = sourcepath.NormalizeList(cfg.ScopePaths)
	cfg.SelectedPaths = sourcepath.NormalizeList(cfg.SelectedPaths)
	cfg.ConfigRef = strings.TrimSpace(cfg.ConfigRef)

	if cfg.RootPath == "" && len(cfg.ScopePaths) == 0 && len(cfg.SelectedPaths) == 0 {
		return nil, fmt.Errorf("microsoft root_path, scope_paths or selected_paths is required")
	}
	if err := sourcepath.ValidateScopesWithinRoot(cfg.RootPath, cfg.ScopePaths); err != nil {
		return nil, fmt.Errorf("invalid microsoft scope_paths: %w", err)
	}
	if err := sourcepath.ValidateScopesWithinRoot(cfg.RootPath, cfg.SelectedPaths); err != nil {
		return nil, fmt.Errorf("invalid microsoft selected_paths: %w", err)
	}

	clientCredParts := 0
	if cfg.TenantID != "" {
		clientCredParts++
	}
	if cfg.ClientID != "" {
		clientCredParts++
	}
	if cfg.ClientSecret != "" {
		clientCredParts++
	}
	if clientCredParts > 0 && clientCredParts < 3 {
		return nil, fmt.Errorf("microsoft tenant_id, client_id and client_secret must all be set together")
	}
	if cfg.AccessToken == "" && clientCredParts != 3 {
		return nil, fmt.Errorf("microsoft access_token or tenant_id/client_id/client_secret is required")
	}
	if cfg.RefreshToken != "" && clientCredParts != 3 {
		return nil, fmt.Errorf("microsoft refresh_token requires tenant_id, client_id and client_secret")
	}

	sanitized, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("encode microsoft config: %w", err)
	}
	return sanitized, nil
}

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

type sourcesService struct {
	repo domain.SourceRepository
}

// NewSourcesService returns a SourceService backed by the given repository.
func NewSourcesService(repo domain.SourceRepository) domain.SourceService {
	return &sourcesService{repo: repo}
}

func (s *sourcesService) Create(ctx context.Context, src domain.Source) (domain.Source, error) {
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
	if src.Type == domain.SourceTypeRclone {
		sanitized, err := sanitizeRcloneConfig(src.Config)
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

func (s *sourcesService) GetByID(ctx context.Context, id, userID string) (domain.Source, error) {
	return s.repo.GetByID(ctx, id, userID)
}

func (s *sourcesService) List(ctx context.Context, userID string, p domain.Page) (domain.SourcePage, error) {
	return s.repo.List(ctx, userID, p)
}

func (s *sourcesService) Delete(ctx context.Context, id, userID string) error {
	return s.repo.Delete(ctx, id, userID)
}

func (s *sourcesService) UpdateGoogleDriveCredentials(
	ctx context.Context,
	id, userID string,
	update domain.GoogleDriveCredentialUpdate,
) (domain.Source, error) {
	src, err := s.repo.GetByID(ctx, id, userID)
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
	return s.repo.UpdateConfig(ctx, id, userID, raw)
}

func (s *sourcesService) UpdateGoogleDriveSelection(
	ctx context.Context,
	id, userID string,
	update domain.GoogleDriveSelectionUpdate,
) (domain.Source, error) {
	src, err := s.repo.GetByID(ctx, id, userID)
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
	return s.repo.UpdateConfig(ctx, id, userID, raw)
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
	switch t {
	case domain.SourceTypeLocalFS, domain.SourceTypeRclone, domain.SourceTypeGoogleDrive:
		return true
	}
	return false
}

func sanitizeRcloneConfig(raw json.RawMessage) (json.RawMessage, error) {
	var cfg domain.RcloneConfig
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("decode rclone config: %w", err)
		}
	}

	cfg.Remote = strings.TrimSpace(cfg.Remote)
	cfg.RootPath = normalizeRclonePath(cfg.RootPath)
	cfg.ScopePaths = normalizeRcloneScopeList(cfg.ScopePaths)
	cfg.SelectedPaths = normalizeRclonePathList(cfg.SelectedPaths)
	cfg.ConfigRef = strings.TrimSpace(cfg.ConfigRef)

	if cfg.Remote == "" {
		return nil, fmt.Errorf("rclone remote is required")
	}
	if cfg.RootPath == "" && len(cfg.ScopePaths) == 0 && len(cfg.SelectedPaths) == 0 {
		return nil, fmt.Errorf("rclone root_path, scope_paths or selected_paths is required")
	}
	if err := validateRcloneScopesWithinRoot(cfg.RootPath, cfg.ScopePaths); err != nil {
		return nil, err
	}
	if err := validateRcloneScopesWithinRoot(cfg.RootPath, cfg.SelectedPaths); err != nil {
		return nil, err
	}

	sanitized, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("encode rclone config: %w", err)
	}
	return sanitized, nil
}

func normalizeRclonePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == "/" {
		return ""
	}
	clean := path.Clean("/" + strings.TrimPrefix(value, "/"))
	if clean == "/" {
		return ""
	}
	return strings.TrimPrefix(clean, "/")
}

func normalizeRcloneScopeList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		scope := normalizeRclonePath(value)
		if scope == "" {
			continue
		}
		set[scope] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for scope := range set {
		out = append(out, scope)
	}
	sort.Strings(out)
	return out
}

func normalizeRclonePathList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		item := normalizeRclonePath(value)
		if item == "" {
			continue
		}
		set[item] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func validateRcloneScopesWithinRoot(rootPath string, scopePaths []string) error {
	rootPath = normalizeRclonePath(rootPath)
	if rootPath == "" || len(scopePaths) == 0 {
		return nil
	}
	rootAbs := "/" + rootPath

	for _, scope := range scopePaths {
		scopeAbs := "/" + normalizeRclonePath(scope)
		if scopeAbs == "/" {
			continue
		}
		if scopeAbs != rootAbs && !strings.HasPrefix(scopeAbs, rootAbs+"/") {
			return fmt.Errorf("rclone scope path %q is outside root_path %q", scope, rootPath)
		}
	}
	return nil
}

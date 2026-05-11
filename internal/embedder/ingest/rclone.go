package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type RcloneListRequest struct {
	UserID     string
	SourceID   string
	Remote     string
	RootPath   string
	ScopePaths []string
}

type RcloneBrowseRequest struct {
	UserID   string
	SourceID string
	Remote   string
	Path     string
}

type RcloneFile struct {
	ExternalID string
	Name       string
	Path       string
	MimeType   string
	Size       int64
	Version    string
	ModifiedAt *time.Time
}

type RcloneEntry struct {
	Name       string
	Path       string
	IsDir      bool
	MimeType   string
	Size       int64
	ModifiedAt *time.Time
}

type RcloneClient interface {
	ListFiles(ctx context.Context, req RcloneListRequest) ([]RcloneFile, error)
	Browse(ctx context.Context, req RcloneBrowseRequest) ([]RcloneEntry, error)
}

type CommandRcloneClient struct {
	binaryPath string
	configDir  string
	timeout    time.Duration
}

func NewCommandRcloneClient(binaryPath, configDir string, timeout time.Duration) *CommandRcloneClient {
	binaryPath = strings.TrimSpace(binaryPath)
	if binaryPath == "" {
		binaryPath = "rclone"
	}
	configDir = strings.TrimSpace(configDir)
	if configDir == "" {
		configDir = "/etc/cube/rclone"
	}
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	return &CommandRcloneClient{
		binaryPath: binaryPath,
		configDir:  configDir,
		timeout:    timeout,
	}
}

type rcloneLSJSONEntry struct {
	Path     string            `json:"Path"`
	Name     string            `json:"Name"`
	Size     int64             `json:"Size"`
	MimeType string            `json:"MimeType"`
	ModTime  string            `json:"ModTime"`
	IsDir    bool              `json:"IsDir"`
	ID       string            `json:"ID"`
	Hashes   map[string]string `json:"Hashes"`
}

func (c *CommandRcloneClient) ListFiles(ctx context.Context, req RcloneListRequest) ([]RcloneFile, error) {
	remote := sanitizeRcloneRemote(req.Remote)
	if remote == "" {
		return nil, fmt.Errorf("rclone remote is required")
	}

	rootPath := normalizeRclonePath(req.RootPath)
	scopes, err := normalizeRcloneScopes(rootPath, req.ScopePaths)
	if err != nil {
		return nil, err
	}
	if len(scopes) == 0 {
		scopes = []string{rootPath}
	}

	cfgPath, err := c.resolveConfigPath(req.UserID, req.SourceID)
	if err != nil {
		return nil, err
	}

	aggregated := make(map[string]RcloneFile)
	for _, scope := range scopes {
		files, err := c.listScope(ctx, cfgPath, remote, scope)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			if file.ExternalID == "" {
				continue
			}
			existing, ok := aggregated[file.ExternalID]
			if !ok || isRcloneFileNewer(file, existing) {
				aggregated[file.ExternalID] = file
			}
		}
	}

	out := make([]RcloneFile, 0, len(aggregated))
	for _, file := range aggregated {
		out = append(out, file)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ExternalID < out[j].ExternalID
	})

	return out, nil
}

func (c *CommandRcloneClient) Browse(ctx context.Context, req RcloneBrowseRequest) ([]RcloneEntry, error) {
	remote := sanitizeRcloneRemote(req.Remote)
	if remote == "" {
		return nil, fmt.Errorf("rclone remote is required")
	}

	cfgPath, err := c.resolveConfigPath(req.UserID, req.SourceID)
	if err != nil {
		return nil, err
	}

	basePath := normalizeRclonePath(req.Path)
	target := remote + ":"
	if basePath != "" {
		target += basePath
	}

	entries, err := c.lsjson(ctx, cfgPath, target, basePath, false, false)
	if err != nil {
		return nil, fmt.Errorf("rclone lsjson failed for path %q: %w", basePath, err)
	}

	out := make([]RcloneEntry, 0, len(entries))
	for _, entry := range entries {
		entryPath := normalizeJoinedRclonePath(basePath, entry.Path)
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			name = path.Base(entryPath)
		}
		out = append(out, RcloneEntry{
			Name:       name,
			Path:       entryPath,
			IsDir:      entry.IsDir,
			MimeType:   strings.TrimSpace(entry.MimeType),
			Size:       entry.Size,
			ModifiedAt: parseRcloneTime(entry.ModTime),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

func (c *CommandRcloneClient) listScope(
	ctx context.Context,
	cfgPath, remote, scope string,
) ([]RcloneFile, error) {
	scopeTarget := remote + ":"
	if scope != "" {
		scopeTarget += scope
	}

	entries, err := c.lsjson(ctx, cfgPath, scopeTarget, scope, true, true)
	if err != nil {
		return nil, fmt.Errorf("rclone lsjson failed for scope %q: %w", scope, err)
	}

	files := make([]RcloneFile, 0, len(entries))
	for _, entry := range entries {
		files = append(files, RcloneFile{
			ExternalID: strings.TrimSpace(entry.ID),
			Name:       strings.TrimSpace(entry.Name),
			Path:       normalizeJoinedRclonePath(scope, entry.Path),
			MimeType:   strings.TrimSpace(entry.MimeType),
			Size:       entry.Size,
			Version:    rcloneVersion(entry),
			ModifiedAt: parseRcloneTime(entry.ModTime),
		})
		if files[len(files)-1].ExternalID == "" {
			files[len(files)-1].ExternalID = files[len(files)-1].Path
		}
		if files[len(files)-1].Name == "" {
			files[len(files)-1].Name = path.Base(files[len(files)-1].Path)
		}
	}

	return files, nil
}

func (c *CommandRcloneClient) resolveConfigPath(userID, sourceID string) (string, error) {
	cfgPath := filepath.Join(
		c.configDir,
		sanitizePathSegment(userID),
		sanitizePathSegment(sourceID),
		"rclone.conf",
	)
	if _, statErr := os.Stat(cfgPath); statErr != nil {
		if os.IsNotExist(statErr) {
			cfgPath = filepath.Join(c.configDir, "rclone.conf")
		} else {
			return "", fmt.Errorf("check rclone config path: %w", statErr)
		}
	}
	return cfgPath, nil
}

func (c *CommandRcloneClient) lsjson(
	ctx context.Context,
	cfgPath, target, basePath string,
	filesOnly bool,
	recursive bool,
) ([]rcloneLSJSONEntry, error) {
	runCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{
		"--config", cfgPath,
		"lsjson",
		target,
		"--metadata",
	}
	if filesOnly {
		args = append(args, "--files-only")
	}
	if recursive {
		args = append(args, "--recursive")
	}

	cmd := exec.CommandContext(runCtx, c.binaryPath, args...)
	out, err := cmd.Output()
	if err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("rclone lsjson timed out for %q", basePath)
		}
		return nil, err
	}

	var entries []rcloneLSJSONEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("decode rclone lsjson output: %w", err)
	}
	return entries, nil
}

func parseRcloneTime(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	return &t
}

func rcloneVersion(entry rcloneLSJSONEntry) string {
	if len(entry.Hashes) > 0 {
		keys := make([]string, 0, len(entry.Hashes))
		for key := range entry.Hashes {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			val := strings.TrimSpace(entry.Hashes[key])
			if val == "" {
				continue
			}
			parts = append(parts, key+":"+val)
		}
		if len(parts) > 0 {
			return strings.Join(parts, "|")
		}
	}
	if mod := strings.TrimSpace(entry.ModTime); mod != "" {
		return fmt.Sprintf("%s|%d", mod, entry.Size)
	}
	return fmt.Sprintf("size:%d", entry.Size)
}

func isRcloneFileNewer(a, b RcloneFile) bool {
	if a.ModifiedAt != nil && b.ModifiedAt != nil {
		if a.ModifiedAt.After(*b.ModifiedAt) {
			return true
		}
		if b.ModifiedAt.After(*a.ModifiedAt) {
			return false
		}
	}
	if a.Version != b.Version {
		return a.Version > b.Version
	}
	return a.Path > b.Path
}

func sanitizeRcloneRemote(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_', r == '-', r == '.':
		default:
			return ""
		}
	}
	return value
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

func normalizeJoinedRclonePath(base, rel string) string {
	base = normalizeRclonePath(base)
	rel = normalizeRclonePath(rel)
	if base == "" {
		return rel
	}
	if rel == "" {
		return base
	}
	return normalizeRclonePath(base + "/" + rel)
}

func normalizeRcloneScopes(rootPath string, scopePaths []string) ([]string, error) {
	rootPath = normalizeRclonePath(rootPath)
	rootAbs := "/" + rootPath
	if rootPath == "" {
		rootAbs = "/"
	}
	if len(scopePaths) == 0 {
		if rootPath == "" {
			return []string{""}, nil
		}
		return []string{rootPath}, nil
	}

	seen := make(map[string]struct{}, len(scopePaths))
	out := make([]string, 0, len(scopePaths))
	for _, raw := range scopePaths {
		scope := normalizeRclonePath(raw)
		scopeAbs := "/" + scope
		if scope == "" {
			scopeAbs = "/"
		}

		if rootAbs != "/" {
			if scopeAbs != rootAbs && !strings.HasPrefix(scopeAbs, rootAbs+"/") {
				return nil, fmt.Errorf("scope path %q is outside approved root %q", raw, rootPath)
			}
		}

		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}

	sort.Strings(out)
	return out, nil
}

func sanitizePathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "_"
	}
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	sanitized := strings.Trim(b.String(), "_")
	if sanitized == "" {
		return "_"
	}
	return sanitized
}

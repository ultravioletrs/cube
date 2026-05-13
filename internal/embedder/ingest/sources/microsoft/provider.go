// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/ingest"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	microsoftGraphBaseURL      = "https://graph.microsoft.com/v1.0"
	microsoftGraphDefaultScope = "https://graph.microsoft.com/.default"
	driveFileMaxBytes          = 200 << 20
)

type sourceProvider struct {
	httpClient *http.Client
}

// NewSourceProvider creates native Microsoft Graph provider implementation.
func NewSourceProvider() ingest.SourceProvider {
	return &sourceProvider{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// NewSourceProviderWithHTTPClient creates provider with custom HTTP client.
func NewSourceProviderWithHTTPClient(httpClient *http.Client) ingest.SourceProvider {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &sourceProvider{httpClient: httpClient}
}

func (p *sourceProvider) Type() domain.SourceType {
	return domain.SourceTypeMicrosoft
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
	var cfg domain.MicrosoftConfig
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decode microsoft config: %w", err)
	}
	return listMicrosoftFiles(ctx, p.httpClient, cfg)
}

func (p *sourceProvider) DownloadRecord(
	ctx context.Context,
	rec domain.Record,
	src domain.Source,
) (string, *int, error) {
	if strings.TrimSpace(rec.ExternalID) == "" {
		return "", nil, fmt.Errorf("record %s is missing external_id", rec.ID)
	}

	var cfg domain.MicrosoftConfig
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return "", nil, fmt.Errorf("decode microsoft config: %w", err)
	}

	graph, err := newMicrosoftGraphClient(ctx, p.httpClient, cfg)
	if err != nil {
		return "", nil, err
	}

	body, err := graph.DownloadContent(ctx, strings.TrimSpace(rec.ExternalID))
	if err != nil {
		return "", nil, err
	}

	doc, err := ingest.ExtractText(ingest.DriveFile{
		ID:       rec.ExternalID,
		Name:     rec.Name,
		MimeType: rec.MimeType,
	}, body)
	if err != nil {
		return "", nil, err
	}
	return doc.Text, doc.PageCount, nil
}

// MicrosoftBrowseEntry is a normalized folder/file entry returned by browse previews.
type MicrosoftBrowseEntry struct {
	Name       string
	Path       string
	IsDir      bool
	MimeType   string
	Size       int64
	ModifiedAt *time.Time
}

// ListMicrosoftFilesPreview lists Microsoft OneDrive/SharePoint files for source configuration preview APIs.
func ListMicrosoftFilesPreview(ctx context.Context, cfg domain.MicrosoftConfig) ([]ingest.SourceFile, error) {
	return listMicrosoftFiles(ctx, &http.Client{Timeout: 30 * time.Second}, cfg)
}

// BrowseMicrosoftPath browses one path level in a Microsoft OneDrive/SharePoint drive.
func BrowseMicrosoftPath(ctx context.Context, cfg domain.MicrosoftConfig, currentPath string) ([]MicrosoftBrowseEntry, error) {
	graph, err := newMicrosoftGraphClient(ctx, &http.Client{Timeout: 30 * time.Second}, cfg)
	if err != nil {
		return nil, err
	}

	currentPath = normalizeRclonePath(currentPath)
	root := normalizeRclonePath(cfg.RootPath)
	if root != "" && !isPathWithinRoot(root, currentPath) {
		return nil, fmt.Errorf("browse path %q is outside root_path %q", currentPath, root)
	}

	item, err := graph.ResolveItemByPath(ctx, currentPath)
	if err != nil {
		return nil, err
	}
	if item.Folder == nil {
		modified := parseRFC3339Ptr(item.LastModifiedDateTime)
		return []MicrosoftBrowseEntry{{
			Name:       item.Name,
			Path:       currentPath,
			IsDir:      false,
			MimeType:   microsoftMimeType(item),
			Size:       item.Size,
			ModifiedAt: modified,
		}}, nil
	}

	children, err := graph.ListChildren(ctx, item.ID)
	if err != nil {
		return nil, err
	}

	entries := make([]MicrosoftBrowseEntry, 0, len(children))
	for _, child := range children {
		childPath := normalizeRclonePath(path.Join(currentPath, child.Name))
		entries = append(entries, MicrosoftBrowseEntry{
			Name:       child.Name,
			Path:       childPath,
			IsDir:      child.Folder != nil,
			MimeType:   microsoftMimeType(child),
			Size:       child.Size,
			ModifiedAt: parseRFC3339Ptr(child.LastModifiedDateTime),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Path < entries[j].Path
	})
	return entries, nil
}

func listMicrosoftFiles(ctx context.Context, baseClient *http.Client, cfg domain.MicrosoftConfig) ([]ingest.SourceFile, error) {
	graph, err := newMicrosoftGraphClient(ctx, baseClient, cfg)
	if err != nil {
		return nil, err
	}

	rootPath := normalizeRclonePath(cfg.RootPath)
	scopes, err := normalizeRcloneScopes(rootPath, cfg.ScopePaths)
	if err != nil {
		return nil, err
	}
	if len(scopes) == 0 {
		scopes = []string{rootPath}
	}

	aggregated := make(map[string]ingest.SourceFile)
	for _, scope := range scopes {
		if err := graph.ListScope(ctx, scope, aggregated); err != nil {
			return nil, err
		}
	}

	files := make([]ingest.SourceFile, 0, len(aggregated))
	for _, file := range aggregated {
		files = append(files, file)
	}
	files = filterSourceFilesBySelectedPaths(files, cfg.SelectedPaths)
	sort.Slice(files, func(i, j int) bool {
		return files[i].ExternalRef < files[j].ExternalRef
	})
	return files, nil
}

type microsoftGraphClient struct {
	httpClient *http.Client
	driveID    string
}

func newMicrosoftGraphClient(ctx context.Context, baseClient *http.Client, cfg domain.MicrosoftConfig) (*microsoftGraphClient, error) {
	cfg = normalizeMicrosoftConfig(cfg)
	tokenSource, err := newMicrosoftTokenSource(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if baseClient == nil {
		baseClient = &http.Client{Timeout: 30 * time.Second}
	}
	oauthCtx := context.WithValue(ctx, oauth2.HTTPClient, baseClient)
	httpClient := oauth2.NewClient(oauthCtx, tokenSource)

	graph := &microsoftGraphClient{httpClient: httpClient}
	driveID, err := graph.resolveDriveID(ctx, cfg)
	if err != nil {
		return nil, err
	}
	graph.driveID = driveID
	return graph, nil
}

func normalizeMicrosoftConfig(cfg domain.MicrosoftConfig) domain.MicrosoftConfig {
	cfg.TenantID = strings.TrimSpace(cfg.TenantID)
	cfg.ClientID = strings.TrimSpace(cfg.ClientID)
	cfg.ClientSecret = strings.TrimSpace(cfg.ClientSecret)
	cfg.AccessToken = strings.TrimSpace(cfg.AccessToken)
	cfg.RefreshToken = strings.TrimSpace(cfg.RefreshToken)
	cfg.DriveID = strings.TrimSpace(cfg.DriveID)
	cfg.SiteID = strings.TrimSpace(cfg.SiteID)
	cfg.RootPath = normalizeRclonePath(cfg.RootPath)
	cfg.ScopePaths = normalizeMicrosoftPathList(cfg.ScopePaths)
	cfg.SelectedPaths = normalizeMicrosoftPathList(cfg.SelectedPaths)
	return cfg
}

func newMicrosoftTokenSource(ctx context.Context, cfg domain.MicrosoftConfig) (oauth2.TokenSource, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", cfg.TenantID)

	if cfg.AccessToken != "" {
		token := &oauth2.Token{
			AccessToken:  cfg.AccessToken,
			RefreshToken: cfg.RefreshToken,
		}
		if cfg.RefreshToken != "" && cfg.ClientID != "" && cfg.ClientSecret != "" && cfg.TenantID != "" {
			oauthCfg := &oauth2.Config{
				ClientID:     cfg.ClientID,
				ClientSecret: cfg.ClientSecret,
				Endpoint: oauth2.Endpoint{
					TokenURL: tokenURL,
				},
			}
			return oauthCfg.TokenSource(ctx, token), nil
		}
		return oauth2.StaticTokenSource(token), nil
	}

	if cfg.TenantID == "" || cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("microsoft access_token or tenant_id/client_id/client_secret is required")
	}
	ccfg := clientcredentials.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     tokenURL,
		Scopes:       []string{microsoftGraphDefaultScope},
	}
	return ccfg.TokenSource(ctx), nil
}

func (g *microsoftGraphClient) resolveDriveID(ctx context.Context, cfg domain.MicrosoftConfig) (string, error) {
	if cfg.DriveID != "" {
		return cfg.DriveID, nil
	}

	var endpoint string
	if cfg.SiteID != "" {
		endpoint = fmt.Sprintf("%s/sites/%s/drive?$select=id", microsoftGraphBaseURL, url.PathEscape(cfg.SiteID))
	} else {
		endpoint = microsoftGraphBaseURL + "/me/drive?$select=id"
	}

	var drive struct {
		ID string `json:"id"`
	}
	if err := g.getJSON(ctx, endpoint, &drive); err != nil {
		return "", err
	}
	drive.ID = strings.TrimSpace(drive.ID)
	if drive.ID == "" {
		return "", fmt.Errorf("microsoft drive id is empty in Graph response")
	}
	return drive.ID, nil
}

func (g *microsoftGraphClient) ListScope(ctx context.Context, scopePath string, out map[string]ingest.SourceFile) error {
	item, err := g.ResolveItemByPath(ctx, scopePath)
	if err != nil {
		return err
	}

	if item.Folder == nil {
		entry := microsoftItemToSourceFile(item, scopePath)
		if entry.ExternalID != "" {
			out[entry.ExternalID] = entry
		}
		return nil
	}

	return g.collectFilesRecursive(ctx, item.ID, normalizeRclonePath(scopePath), out)
}

func (g *microsoftGraphClient) collectFilesRecursive(
	ctx context.Context,
	parentID string,
	parentPath string,
	out map[string]ingest.SourceFile,
) error {
	children, err := g.ListChildren(ctx, parentID)
	if err != nil {
		return err
	}

	for _, child := range children {
		childPath := normalizeRclonePath(path.Join(parentPath, child.Name))
		if child.Folder != nil {
			if err := g.collectFilesRecursive(ctx, child.ID, childPath, out); err != nil {
				return err
			}
			continue
		}

		entry := microsoftItemToSourceFile(child, childPath)
		if entry.ExternalID == "" {
			continue
		}
		existing, ok := out[entry.ExternalID]
		if !ok || sourceFileNewer(entry, existing) {
			out[entry.ExternalID] = entry
		}
	}

	return nil
}

func (g *microsoftGraphClient) ResolveItemByPath(ctx context.Context, relPath string) (microsoftDriveItem, error) {
	relPath = normalizeRclonePath(relPath)
	selectQ := "$select=id,name,webUrl,lastModifiedDateTime,eTag,size,file,folder"

	var endpoint string
	if relPath == "" {
		endpoint = fmt.Sprintf("%s/drives/%s/root?%s", microsoftGraphBaseURL, url.PathEscape(g.driveID), selectQ)
	} else {
		endpoint = fmt.Sprintf(
			"%s/drives/%s/root:/%s:?%s",
			microsoftGraphBaseURL,
			url.PathEscape(g.driveID),
			encodeMicrosoftPath(relPath),
			selectQ,
		)
	}

	var item microsoftDriveItem
	if err := g.getJSON(ctx, endpoint, &item); err != nil {
		return microsoftDriveItem{}, err
	}
	item.ID = strings.TrimSpace(item.ID)
	if item.ID == "" {
		return microsoftDriveItem{}, fmt.Errorf("microsoft drive item id is empty for path %q", relPath)
	}
	return item, nil
}

func (g *microsoftGraphClient) ListChildren(ctx context.Context, parentID string) ([]microsoftDriveItem, error) {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return nil, fmt.Errorf("microsoft parent item id is required")
	}

	endpoint := fmt.Sprintf(
		"%s/drives/%s/items/%s/children?$top=200&$select=id,name,webUrl,lastModifiedDateTime,eTag,size,file,folder",
		microsoftGraphBaseURL,
		url.PathEscape(g.driveID),
		url.PathEscape(parentID),
	)

	items := make([]microsoftDriveItem, 0, 200)
	for endpoint != "" {
		var page struct {
			Value    []microsoftDriveItem `json:"value"`
			NextLink string               `json:"@odata.nextLink"`
		}
		if err := g.getJSON(ctx, endpoint, &page); err != nil {
			return nil, err
		}
		items = append(items, page.Value...)
		endpoint = strings.TrimSpace(page.NextLink)
	}
	return items, nil
}

func (g *microsoftGraphClient) DownloadContent(ctx context.Context, itemID string) ([]byte, error) {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return nil, fmt.Errorf("microsoft item id is required")
	}

	reqURL := fmt.Sprintf(
		"%s/drives/%s/items/%s/content",
		microsoftGraphBaseURL,
		url.PathEscape(g.driveID),
		url.PathEscape(itemID),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("microsoft download request: %w", err)
	}
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("microsoft download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("microsoft download status %d: %s", resp.StatusCode, body)
	}

	lim := &io.LimitedReader{R: resp.Body, N: driveFileMaxBytes + 1}
	body, err := io.ReadAll(lim)
	if err != nil {
		return nil, fmt.Errorf("microsoft download read: %w", err)
	}
	if int64(len(body)) > driveFileMaxBytes {
		return nil, fmt.Errorf("microsoft file too large: %d bytes (max %d)", len(body), driveFileMaxBytes)
	}
	return body, nil
}

func (g *microsoftGraphClient) getJSON(ctx context.Context, reqURL string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("microsoft graph status %d: %s", resp.StatusCode, microsoftGraphErrorMessage(body))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode microsoft graph response: %w", err)
	}
	return nil
}

func microsoftGraphErrorMessage(body []byte) string {
	msg := strings.TrimSpace(string(body))
	var parsed struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return msg
	}
	if strings.TrimSpace(parsed.Error.Message) != "" {
		if strings.TrimSpace(parsed.Error.Code) != "" {
			return parsed.Error.Code + ": " + parsed.Error.Message
		}
		return parsed.Error.Message
	}
	return msg
}

func encodeMicrosoftPath(relPath string) string {
	relPath = normalizeRclonePath(relPath)
	if relPath == "" {
		return ""
	}
	parts := strings.Split(relPath, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func normalizeMicrosoftPathList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = normalizeRclonePath(value)
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func microsoftMimeType(item microsoftDriveItem) string {
	if item.File == nil {
		return ""
	}
	return strings.TrimSpace(item.File.MimeType)
}

func microsoftItemToSourceFile(item microsoftDriveItem, itemPath string) ingest.SourceFile {
	itemPath = normalizeRclonePath(itemPath)
	return ingest.SourceFile{
		ExternalID:       strings.TrimSpace(item.ID),
		Name:             strings.TrimSpace(item.Name),
		ExternalURL:      strings.TrimSpace(item.WebURL),
		ExternalRef:      itemPath,
		MimeType:         microsoftMimeType(item),
		SourceVersion:    strings.TrimSpace(item.ETag),
		SourceModifiedAt: parseRFC3339Ptr(item.LastModifiedDateTime),
	}
}

func parseRFC3339Ptr(value string) *time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return &t
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

func filterSourceFilesBySelectedPaths(files []ingest.SourceFile, selectedPaths []string) []ingest.SourceFile {
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

	filtered := make([]ingest.SourceFile, 0, len(files))
	for _, file := range files {
		if _, ok := selected[normalizeRclonePath(file.ExternalRef)]; ok {
			filtered = append(filtered, file)
		}
	}
	return filtered
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

type microsoftDriveItem struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	WebURL               string `json:"webUrl"`
	LastModifiedDateTime string `json:"lastModifiedDateTime"`
	ETag                 string `json:"eTag"`
	Size                 int64  `json:"size"`
	File                 *struct {
		MimeType string `json:"mimeType"`
	} `json:"file"`
	Folder *struct {
		ChildCount int `json:"childCount"`
	} `json:"folder"`
}

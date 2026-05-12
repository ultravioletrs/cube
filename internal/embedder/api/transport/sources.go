// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ultravioletrs/cube/internal/embedder/auth"
	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/ingest"
)

// MountSources registers Source routes on the given router.
// All routes require an authenticated user (auth.Middleware must run first).
func MountSources(
	r chi.Router,
	svc domain.SourceService,
	syncSvc domain.SourceSyncService,
	trigger func(),
	googleOAuthClientID string,
	googleOAuthClientSecret string,
) {
	oauth := newGoogleOAuth(googleOAuthClientID, googleOAuthClientSecret)
	rclone := ingest.NewCommandRcloneClient("", "", 0)

	r.Get("/api/v1/sources", listSources(svc))
	r.Post("/api/v1/sources", createSource(svc, oauth))
	r.Post("/api/v1/sources/google/oauth/url", googleOAuthURL(oauth))
	r.Post("/api/v1/sources/google/oauth/exchange", googleOAuthExchange(oauth))
	r.Post("/api/v1/sources/google/files", listDriveFiles(oauth))
	r.Post("/api/v1/sources/s3/browse", browseS3Path())
	r.Post("/api/v1/sources/s3/files", listS3Files())
	r.Post("/api/v1/sources/microsoft/browse", browseMicrosoftPath())
	r.Post("/api/v1/sources/microsoft/files", listMicrosoftFiles())
	r.Post("/api/v1/sources/rclone/browse", browseRclonePath(rclone))
	r.Post("/api/v1/sources/rclone/files", listRcloneFiles(rclone))
	r.Post("/api/v1/sources/{id}/sync", syncSource(syncSvc, trigger))
	r.Get("/api/v1/sources/{id}", getSource(svc))
	r.Put("/api/v1/sources/{id}/credentials", updateSourceCredentials(svc))
	r.Put("/api/v1/sources/{id}/selection", updateSourceSelection(svc))
	r.Delete("/api/v1/sources/{id}", deleteSource(svc))
}

type googleOAuth struct {
	clientID     string
	clientSecret string
	mu           sync.Mutex
	states       map[string]googleOAuthState
}

type googleOAuthState struct {
	UserID    string
	ExpiresAt time.Time
}

func newGoogleOAuth(clientID, clientSecret string) *googleOAuth {
	return &googleOAuth{
		clientID:     strings.TrimSpace(clientID),
		clientSecret: strings.TrimSpace(clientSecret),
		states:       make(map[string]googleOAuthState),
	}
}

func (g *googleOAuth) enabled() bool {
	return g.clientID != "" && g.clientSecret != ""
}

func (g *googleOAuth) createState(userID string) (string, error) {
	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	state := fmt.Sprintf("%x", tokenBytes)

	g.mu.Lock()
	defer g.mu.Unlock()
	g.cleanupLocked()
	g.states[state] = googleOAuthState{
		UserID:    userID,
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute),
	}
	return state, nil
}

func (g *googleOAuth) consumeState(userID, state string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cleanupLocked()

	entry, ok := g.states[state]
	if !ok {
		return false
	}
	delete(g.states, state)
	return entry.UserID == userID
}

func (g *googleOAuth) cleanupLocked() {
	now := time.Now().UTC()
	for key, value := range g.states {
		if now.After(value.ExpiresAt) {
			delete(g.states, key)
		}
	}
}

func listDriveFiles(oauth *googleOAuth) http.HandlerFunc {
	type request struct {
		FolderID     string `json:"folder_id,omitempty"`
		FolderLink   string `json:"folder_link,omitempty"`
		BrowseFolder string `json:"browse_folder_id,omitempty"`
		BrowseMode   bool   `json:"browse_mode,omitempty"`
		FolderQuery  string `json:"folder_query,omitempty"`
		FoldersOnly  bool   `json:"folders_only,omitempty"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token,omitempty"`
		ClientID     string `json:"client_id,omitempty"`
		ClientSecret string `json:"client_secret,omitempty"`
	}
	type fileResponse struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		MimeType     string   `json:"mime_type"`
		ModifiedTime string   `json:"modified_time"`
		WebViewLink  string   `json:"web_view_link"`
		Parents      []string `json:"parents"`
	}
	type response struct {
		Files          []fileResponse `json:"files"`
		Folders        []fileResponse `json:"folders"`
		CurrentFolder  string         `json:"current_folder_id,omitempty"`
		RequestedScope string         `json:"requested_scope,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}

		cfg := domain.GoogleDriveConfig{
			FolderID:     parseDriveFolderID(req.FolderID, req.FolderLink),
			FolderLink:   strings.TrimSpace(req.FolderLink),
			AccessToken:  strings.TrimSpace(req.AccessToken),
			RefreshToken: strings.TrimSpace(req.RefreshToken),
			ClientID:     strings.TrimSpace(req.ClientID),
			ClientSecret: strings.TrimSpace(req.ClientSecret),
		}
		if cfg.RefreshToken != "" && oauth.enabled() {
			if cfg.ClientID == "" {
				cfg.ClientID = oauth.clientID
			}
			if cfg.ClientSecret == "" {
				cfg.ClientSecret = oauth.clientSecret
			}
		}
		if cfg.AccessToken == "" {
			writeJSON(w, http.StatusBadRequest, errBody("access_token is required"))
			return
		}

		reader, err := ingest.NewDriveReaderFromConfig(r.Context(), cfg)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errBody(err.Error()))
			return
		}

		browseFolderID := strings.TrimSpace(req.BrowseFolder)
		folderQuery := strings.TrimSpace(req.FolderQuery)
		if req.BrowseMode {
			if folderQuery != "" || req.FoldersOnly {
				folders, err := reader.SearchFolders(r.Context(), browseFolderID, folderQuery)
				if err != nil {
					writeJSON(w, http.StatusUnprocessableEntity, errBody(err.Error()))
					return
				}
				resp := response{
					Files:          []fileResponse{},
					Folders:        make([]fileResponse, 0, len(folders)),
					CurrentFolder:  browseFolderID,
					RequestedScope: cfg.FolderID,
				}
				for _, folder := range folders {
					resp.Folders = append(resp.Folders, fileResponse{
						ID:           folder.ID,
						Name:         folder.Name,
						MimeType:     folder.MimeType,
						ModifiedTime: folder.ModifiedTime,
						WebViewLink:  folder.WebViewLink,
						Parents:      folder.Parents,
					})
				}
				writeJSON(w, http.StatusOK, resp)
				return
			}

			folders, files, err := reader.ListFolderContent(r.Context(), browseFolderID)
			if err != nil {
				writeJSON(w, http.StatusUnprocessableEntity, errBody(err.Error()))
				return
			}

			resp := response{
				Files:          make([]fileResponse, 0, len(files)),
				Folders:        make([]fileResponse, 0, len(folders)),
				CurrentFolder:  browseFolderID,
				RequestedScope: cfg.FolderID,
			}
			for _, folder := range folders {
				resp.Folders = append(resp.Folders, fileResponse{
					ID:           folder.ID,
					Name:         folder.Name,
					MimeType:     folder.MimeType,
					ModifiedTime: folder.ModifiedTime,
					WebViewLink:  folder.WebViewLink,
					Parents:      folder.Parents,
				})
			}
			for _, file := range files {
				resp.Files = append(resp.Files, fileResponse{
					ID:           file.ID,
					Name:         file.Name,
					MimeType:     file.MimeType,
					ModifiedTime: file.ModifiedTime,
					WebViewLink:  file.WebViewLink,
					Parents:      file.Parents,
				})
			}
			writeJSON(w, http.StatusOK, resp)
			return
		}

		files, err := reader.ListFiles(r.Context(), cfg.FolderID)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errBody(err.Error()))
			return
		}

		resp := response{
			Files:          make([]fileResponse, 0, len(files)),
			Folders:        []fileResponse{},
			RequestedScope: cfg.FolderID,
		}
		for _, file := range files {
			resp.Files = append(resp.Files, fileResponse{
				ID:           file.ID,
				Name:         file.Name,
				MimeType:     file.MimeType,
				ModifiedTime: file.ModifiedTime,
				WebViewLink:  file.WebViewLink,
				Parents:      file.Parents,
			})
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func listRcloneFiles(rclone ingest.RcloneClient) http.HandlerFunc {
	type request struct {
		Remote     string   `json:"remote"`
		RootPath   string   `json:"root_path,omitempty"`
		ScopePaths []string `json:"scope_paths,omitempty"`
	}
	type fileResponse struct {
		ExternalID   string `json:"external_id"`
		Name         string `json:"name"`
		Path         string `json:"path"`
		MimeType     string `json:"mime_type"`
		Size         int64  `json:"size"`
		ModifiedTime string `json:"modified_time,omitempty"`
	}
	type response struct {
		Files []fileResponse `json:"files"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if rclone == nil {
			writeJSON(w, http.StatusUnprocessableEntity, errBody("rclone source browse is not configured"))
			return
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}

		req.Remote = strings.TrimSpace(req.Remote)
		req.RootPath = strings.TrimSpace(req.RootPath)
		scopePaths := make([]string, 0, len(req.ScopePaths))
		for _, scope := range req.ScopePaths {
			scope = strings.TrimSpace(scope)
			if scope == "" {
				continue
			}
			scopePaths = append(scopePaths, scope)
		}
		if req.Remote == "" {
			writeJSON(w, http.StatusBadRequest, errBody("rclone remote is required"))
			return
		}
		if req.RootPath == "" && len(scopePaths) == 0 {
			writeJSON(w, http.StatusBadRequest, errBody("rclone root_path or scope_paths is required"))
			return
		}

		files, err := rclone.ListFiles(r.Context(), ingest.RcloneListRequest{
			UserID:     auth.UserID(r.Context()),
			SourceID:   "preview",
			Remote:     req.Remote,
			RootPath:   req.RootPath,
			ScopePaths: scopePaths,
		})
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errBody(err.Error()))
			return
		}

		resp := response{Files: make([]fileResponse, 0, len(files))}
		for _, file := range files {
			item := fileResponse{
				ExternalID: file.ExternalID,
				Name:       file.Name,
				Path:       file.Path,
				MimeType:   file.MimeType,
				Size:       file.Size,
			}
			if file.ModifiedAt != nil {
				item.ModifiedTime = file.ModifiedAt.UTC().Format(time.RFC3339)
			}
			resp.Files = append(resp.Files, item)
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func listS3Files() http.HandlerFunc {
	type request struct {
		Endpoint        string   `json:"endpoint,omitempty"`
		Region          string   `json:"region,omitempty"`
		Bucket          string   `json:"bucket"`
		AccessKeyID     string   `json:"access_key_id,omitempty"`
		SecretAccessKey string   `json:"secret_access_key,omitempty"`
		SessionToken    string   `json:"session_token,omitempty"`
		UseSSL          *bool    `json:"use_ssl,omitempty"`
		PathStyle       *bool    `json:"path_style,omitempty"`
		RootPath        string   `json:"root_path,omitempty"`
		ScopePaths      []string `json:"scope_paths,omitempty"`
		SelectedPaths   []string `json:"selected_paths,omitempty"`
	}
	type fileResponse struct {
		ExternalID   string `json:"external_id"`
		Name         string `json:"name"`
		Path         string `json:"path"`
		MimeType     string `json:"mime_type"`
		ModifiedTime string `json:"modified_time,omitempty"`
	}
	type response struct {
		Files []fileResponse `json:"files"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}

		cfg, err := sanitizeS3PreviewConfig(
			req.Endpoint,
			req.Region,
			req.Bucket,
			req.AccessKeyID,
			req.SecretAccessKey,
			req.SessionToken,
			req.UseSSL,
			req.PathStyle,
			req.RootPath,
			req.ScopePaths,
			req.SelectedPaths,
		)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errBody(err.Error()))
			return
		}

		files, err := ingest.ListS3FilesPreview(r.Context(), cfg)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errBody(err.Error()))
			return
		}

		resp := response{Files: make([]fileResponse, 0, len(files))}
		for _, file := range files {
			item := fileResponse{
				ExternalID: file.ExternalID,
				Name:       file.Name,
				Path:       file.ExternalRef,
				MimeType:   file.MimeType,
			}
			if file.SourceModifiedAt != nil {
				item.ModifiedTime = file.SourceModifiedAt.UTC().Format(time.RFC3339)
			}
			resp.Files = append(resp.Files, item)
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func browseS3Path() http.HandlerFunc {
	type request struct {
		Endpoint        string `json:"endpoint,omitempty"`
		Region          string `json:"region,omitempty"`
		Bucket          string `json:"bucket"`
		AccessKeyID     string `json:"access_key_id,omitempty"`
		SecretAccessKey string `json:"secret_access_key,omitempty"`
		SessionToken    string `json:"session_token,omitempty"`
		UseSSL          *bool  `json:"use_ssl,omitempty"`
		PathStyle       *bool  `json:"path_style,omitempty"`
		RootPath        string `json:"root_path,omitempty"`
		Path            string `json:"path,omitempty"`
	}
	type folderResponse struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	type fileResponse struct {
		Name         string `json:"name"`
		Path         string `json:"path"`
		MimeType     string `json:"mime_type"`
		Size         int64  `json:"size"`
		ModifiedTime string `json:"modified_time,omitempty"`
	}
	type response struct {
		CurrentPath string           `json:"current_path"`
		ParentPath  string           `json:"parent_path,omitempty"`
		Folders     []folderResponse `json:"folders"`
		Files       []fileResponse   `json:"files"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}

		cfg, err := sanitizeS3PreviewConfig(
			req.Endpoint,
			req.Region,
			req.Bucket,
			req.AccessKeyID,
			req.SecretAccessKey,
			req.SessionToken,
			req.UseSSL,
			req.PathStyle,
			req.RootPath,
			nil,
			nil,
		)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errBody(err.Error()))
			return
		}

		entries, err := ingest.BrowseS3Path(r.Context(), cfg, req.Path)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errBody(err.Error()))
			return
		}

		currentPath := normalizeBrowsePath(req.Path)
		resp := response{
			CurrentPath: currentPath,
			Folders:     make([]folderResponse, 0, len(entries)),
			Files:       make([]fileResponse, 0, len(entries)),
		}
		if currentPath != "" {
			parent := path.Dir("/" + currentPath)
			if parent != "/" {
				resp.ParentPath = strings.TrimPrefix(parent, "/")
			}
		}

		for _, entry := range entries {
			if entry.IsDir {
				resp.Folders = append(resp.Folders, folderResponse{
					Name: entry.Name,
					Path: entry.Path,
				})
				continue
			}
			item := fileResponse{
				Name:     entry.Name,
				Path:     entry.Path,
				MimeType: entry.MimeType,
				Size:     entry.Size,
			}
			if entry.ModifiedAt != nil {
				item.ModifiedTime = entry.ModifiedAt.UTC().Format(time.RFC3339)
			}
			resp.Files = append(resp.Files, item)
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func listMicrosoftFiles() http.HandlerFunc {
	type request struct {
		TenantID      string   `json:"tenant_id,omitempty"`
		ClientID      string   `json:"client_id,omitempty"`
		ClientSecret  string   `json:"client_secret,omitempty"`
		AccessToken   string   `json:"access_token,omitempty"`
		RefreshToken  string   `json:"refresh_token,omitempty"`
		DriveID       string   `json:"drive_id,omitempty"`
		SiteID        string   `json:"site_id,omitempty"`
		RootPath      string   `json:"root_path,omitempty"`
		ScopePaths    []string `json:"scope_paths,omitempty"`
		SelectedPaths []string `json:"selected_paths,omitempty"`
	}
	type fileResponse struct {
		ExternalID   string `json:"external_id"`
		Name         string `json:"name"`
		Path         string `json:"path"`
		MimeType     string `json:"mime_type"`
		ModifiedTime string `json:"modified_time,omitempty"`
	}
	type response struct {
		Files []fileResponse `json:"files"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}

		cfg, err := sanitizeMicrosoftPreviewConfig(
			req.TenantID,
			req.ClientID,
			req.ClientSecret,
			req.AccessToken,
			req.RefreshToken,
			req.DriveID,
			req.SiteID,
			req.RootPath,
			req.ScopePaths,
			req.SelectedPaths,
		)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errBody(err.Error()))
			return
		}

		files, err := ingest.ListMicrosoftFilesPreview(r.Context(), cfg)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errBody(err.Error()))
			return
		}

		resp := response{Files: make([]fileResponse, 0, len(files))}
		for _, file := range files {
			item := fileResponse{
				ExternalID: file.ExternalID,
				Name:       file.Name,
				Path:       file.ExternalRef,
				MimeType:   file.MimeType,
			}
			if file.SourceModifiedAt != nil {
				item.ModifiedTime = file.SourceModifiedAt.UTC().Format(time.RFC3339)
			}
			resp.Files = append(resp.Files, item)
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func browseMicrosoftPath() http.HandlerFunc {
	type request struct {
		TenantID     string `json:"tenant_id,omitempty"`
		ClientID     string `json:"client_id,omitempty"`
		ClientSecret string `json:"client_secret,omitempty"`
		AccessToken  string `json:"access_token,omitempty"`
		RefreshToken string `json:"refresh_token,omitempty"`
		DriveID      string `json:"drive_id,omitempty"`
		SiteID       string `json:"site_id,omitempty"`
		RootPath     string `json:"root_path,omitempty"`
		Path         string `json:"path,omitempty"`
	}
	type folderResponse struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	type fileResponse struct {
		Name         string `json:"name"`
		Path         string `json:"path"`
		MimeType     string `json:"mime_type"`
		Size         int64  `json:"size"`
		ModifiedTime string `json:"modified_time,omitempty"`
	}
	type response struct {
		CurrentPath string           `json:"current_path"`
		ParentPath  string           `json:"parent_path,omitempty"`
		Folders     []folderResponse `json:"folders"`
		Files       []fileResponse   `json:"files"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}

		cfg, err := sanitizeMicrosoftPreviewConfig(
			req.TenantID,
			req.ClientID,
			req.ClientSecret,
			req.AccessToken,
			req.RefreshToken,
			req.DriveID,
			req.SiteID,
			req.RootPath,
			nil,
			nil,
		)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errBody(err.Error()))
			return
		}

		entries, err := ingest.BrowseMicrosoftPath(r.Context(), cfg, req.Path)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errBody(err.Error()))
			return
		}

		currentPath := normalizeBrowsePath(req.Path)
		resp := response{
			CurrentPath: currentPath,
			Folders:     make([]folderResponse, 0, len(entries)),
			Files:       make([]fileResponse, 0, len(entries)),
		}
		if currentPath != "" {
			parent := path.Dir("/" + currentPath)
			if parent != "/" {
				resp.ParentPath = strings.TrimPrefix(parent, "/")
			}
		}

		for _, entry := range entries {
			if entry.IsDir {
				resp.Folders = append(resp.Folders, folderResponse{
					Name: entry.Name,
					Path: entry.Path,
				})
				continue
			}
			item := fileResponse{
				Name:     entry.Name,
				Path:     entry.Path,
				MimeType: entry.MimeType,
				Size:     entry.Size,
			}
			if entry.ModifiedAt != nil {
				item.ModifiedTime = entry.ModifiedAt.UTC().Format(time.RFC3339)
			}
			resp.Files = append(resp.Files, item)
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func browseRclonePath(rclone ingest.RcloneClient) http.HandlerFunc {
	type request struct {
		Remote string `json:"remote"`
		Path   string `json:"path,omitempty"`
	}
	type folderResponse struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	type fileResponse struct {
		Name         string `json:"name"`
		Path         string `json:"path"`
		MimeType     string `json:"mime_type"`
		Size         int64  `json:"size"`
		ModifiedTime string `json:"modified_time,omitempty"`
	}
	type response struct {
		CurrentPath string           `json:"current_path"`
		ParentPath  string           `json:"parent_path,omitempty"`
		Folders     []folderResponse `json:"folders"`
		Files       []fileResponse   `json:"files"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if rclone == nil {
			writeJSON(w, http.StatusUnprocessableEntity, errBody("rclone source browse is not configured"))
			return
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}

		req.Remote = strings.TrimSpace(req.Remote)
		req.Path = strings.TrimSpace(req.Path)
		if req.Remote == "" {
			writeJSON(w, http.StatusBadRequest, errBody("rclone remote is required"))
			return
		}

		entries, err := rclone.Browse(r.Context(), ingest.RcloneBrowseRequest{
			UserID:   auth.UserID(r.Context()),
			SourceID: "preview",
			Remote:   req.Remote,
			Path:     req.Path,
		})
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errBody(err.Error()))
			return
		}

		currentPath := normalizeBrowsePath(req.Path)
		resp := response{
			CurrentPath: currentPath,
			Folders:     make([]folderResponse, 0, len(entries)),
			Files:       make([]fileResponse, 0, len(entries)),
		}
		if currentPath != "" {
			parent := path.Dir("/" + currentPath)
			if parent != "/" {
				resp.ParentPath = strings.TrimPrefix(parent, "/")
			}
		}

		for _, entry := range entries {
			if entry.IsDir {
				resp.Folders = append(resp.Folders, folderResponse{
					Name: entry.Name,
					Path: entry.Path,
				})
				continue
			}
			item := fileResponse{
				Name:     entry.Name,
				Path:     entry.Path,
				MimeType: entry.MimeType,
				Size:     entry.Size,
			}
			if entry.ModifiedAt != nil {
				item.ModifiedTime = entry.ModifiedAt.UTC().Format(time.RFC3339)
			}
			resp.Files = append(resp.Files, item)
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

func normalizeBrowsePath(value string) string {
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

func sanitizeS3PreviewConfig(
	endpoint, region, bucket, accessKeyID, secretAccessKey, sessionToken string,
	useSSL, pathStyle *bool,
	rootPath string,
	scopePaths, selectedPaths []string,
) (domain.S3Config, error) {
	cfg := domain.S3Config{
		Endpoint:        strings.TrimSpace(endpoint),
		Region:          strings.TrimSpace(region),
		Bucket:          strings.TrimSpace(bucket),
		AccessKeyID:     strings.TrimSpace(accessKeyID),
		SecretAccessKey: strings.TrimSpace(secretAccessKey),
		SessionToken:    strings.TrimSpace(sessionToken),
		UseSSL:          useSSL,
		PathStyle:       pathStyle,
		RootPath:        normalizeBrowsePath(rootPath),
		ScopePaths:      normalizeBrowsePathList(scopePaths),
		SelectedPaths:   normalizeBrowsePathList(selectedPaths),
	}
	if cfg.Bucket == "" {
		return domain.S3Config{}, fmt.Errorf("s3 bucket is required")
	}
	if cfg.AccessKeyID == "" && cfg.SecretAccessKey != "" {
		return domain.S3Config{}, fmt.Errorf("s3 access_key_id is required when secret_access_key is set")
	}
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey == "" {
		return domain.S3Config{}, fmt.Errorf("s3 secret_access_key is required when access_key_id is set")
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.UseSSL == nil {
		v := true
		cfg.UseSSL = &v
	}
	if cfg.PathStyle == nil {
		v := true
		cfg.PathStyle = &v
	}
	return cfg, nil
}

func sanitizeMicrosoftPreviewConfig(
	tenantID, clientID, clientSecret, accessToken, refreshToken, driveID, siteID, rootPath string,
	scopePaths, selectedPaths []string,
) (domain.MicrosoftConfig, error) {
	cfg := domain.MicrosoftConfig{
		TenantID:      strings.TrimSpace(tenantID),
		ClientID:      strings.TrimSpace(clientID),
		ClientSecret:  strings.TrimSpace(clientSecret),
		AccessToken:   strings.TrimSpace(accessToken),
		RefreshToken:  strings.TrimSpace(refreshToken),
		DriveID:       strings.TrimSpace(driveID),
		SiteID:        strings.TrimSpace(siteID),
		RootPath:      normalizeBrowsePath(rootPath),
		ScopePaths:    normalizeBrowsePathList(scopePaths),
		SelectedPaths: normalizeBrowsePathList(selectedPaths),
	}

	if cfg.RootPath == "" && len(cfg.ScopePaths) == 0 && len(cfg.SelectedPaths) == 0 {
		return domain.MicrosoftConfig{}, fmt.Errorf("microsoft root_path, scope_paths or selected_paths is required")
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
		return domain.MicrosoftConfig{}, fmt.Errorf("microsoft tenant_id, client_id and client_secret must all be set together")
	}
	if cfg.AccessToken == "" && clientCredParts != 3 {
		return domain.MicrosoftConfig{}, fmt.Errorf("microsoft access_token or tenant_id/client_id/client_secret is required")
	}
	if cfg.RefreshToken != "" && clientCredParts != 3 {
		return domain.MicrosoftConfig{}, fmt.Errorf("microsoft refresh_token requires tenant_id, client_id and client_secret")
	}

	return cfg, nil
}

func normalizeBrowsePathList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = normalizeBrowsePath(value)
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

func googleOAuthURL(oauth *googleOAuth) http.HandlerFunc {
	type request struct {
		RedirectURI string `json:"redirect_uri"`
	}
	type response struct {
		AuthURL string `json:"auth_url"`
		State   string `json:"state"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if !oauth.enabled() {
			writeJSON(w, http.StatusNotImplemented, errBody("google oauth is not configured"))
			return
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}
		redirectURI := strings.TrimSpace(req.RedirectURI)
		if redirectURI == "" {
			writeJSON(w, http.StatusBadRequest, errBody("redirect_uri is required"))
			return
		}

		userID := auth.UserID(r.Context())
		state, err := oauth.createState(userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("failed to generate oauth state"))
			return
		}

		query := url.Values{
			"client_id":              {oauth.clientID},
			"redirect_uri":           {redirectURI},
			"response_type":          {"code"},
			"scope":                  {"https://www.googleapis.com/auth/drive.readonly"},
			"access_type":            {"offline"},
			"include_granted_scopes": {"true"},
			"prompt":                 {"consent"},
			"state":                  {state},
		}
		authURL := "https://accounts.google.com/o/oauth2/v2/auth?" + query.Encode()
		writeJSON(w, http.StatusOK, response{AuthURL: authURL, State: state})
	}
}

func googleOAuthExchange(oauth *googleOAuth) http.HandlerFunc {
	type request struct {
		Code        string `json:"code"`
		State       string `json:"state"`
		RedirectURI string `json:"redirect_uri"`
	}
	type response struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token,omitempty"`
		ExpiresIn    int    `json:"expires_in,omitempty"`
		TokenType    string `json:"token_type,omitempty"`
		Scope        string `json:"scope,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if !oauth.enabled() {
			writeJSON(w, http.StatusNotImplemented, errBody("google oauth is not configured"))
			return
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}

		req.Code = strings.TrimSpace(req.Code)
		req.State = strings.TrimSpace(req.State)
		req.RedirectURI = strings.TrimSpace(req.RedirectURI)
		if req.Code == "" || req.State == "" || req.RedirectURI == "" {
			writeJSON(w, http.StatusBadRequest, errBody("code, state and redirect_uri are required"))
			return
		}

		userID := auth.UserID(r.Context())
		if !oauth.consumeState(userID, req.State) {
			writeJSON(w, http.StatusUnauthorized, errBody("invalid oauth state"))
			return
		}

		token, err := exchangeGoogleAuthCode(r.Context(), oauth.clientID, oauth.clientSecret, req.Code, req.RedirectURI)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errBody(err.Error()))
			return
		}

		writeJSON(w, http.StatusOK, response{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			ExpiresIn:    token.ExpiresIn,
			TokenType:    token.TokenType,
			Scope:        token.Scope,
		})
	}
}

type googleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	Error        string `json:"error"`
}

func exchangeGoogleAuthCode(
	ctx context.Context,
	clientID string,
	clientSecret string,
	code string,
	redirectURI string,
) (googleTokenResponse, error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return googleTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return googleTokenResponse{}, fmt.Errorf("google oauth exchange failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var parsed googleTokenResponse
	_ = json.Unmarshal(body, &parsed)

	if resp.StatusCode != http.StatusOK {
		if parsed.Error != "" {
			return googleTokenResponse{}, fmt.Errorf("google oauth exchange failed: %s", parsed.Error)
		}
		return googleTokenResponse{}, fmt.Errorf("google oauth exchange failed with status %d", resp.StatusCode)
	}
	if strings.TrimSpace(parsed.AccessToken) == "" {
		return googleTokenResponse{}, fmt.Errorf("google oauth exchange returned empty access token")
	}
	return parsed, nil
}

// sourceResponse is the JSON shape returned by source endpoints.
type sourceResponse struct {
	ID               string          `json:"id"`
	UserID           string          `json:"user_id"`
	SourceType       string          `json:"source_type"`
	Name             string          `json:"name"`
	Config           json.RawMessage `json:"config"`
	Status           string          `json:"status"`
	SyncEnabled      bool            `json:"sync_enabled"`
	AutoSyncInterval int             `json:"auto_sync_interval"`
	LastSyncAt       *string         `json:"last_sync_at,omitempty"`
	LastSyncError    *string         `json:"last_sync_error,omitempty"`
	NextSyncAt       *string         `json:"next_sync_at,omitempty"`
	CreatedAt        string          `json:"created_at"`
	UpdatedAt        string          `json:"updated_at"`
}

func toSourceResponse(s domain.Source) sourceResponse {
	r := sourceResponse{
		ID:               s.ID,
		UserID:           s.UserID,
		SourceType:       string(s.Type),
		Name:             s.Name,
		Config:           s.Config,
		Status:           string(s.Status),
		SyncEnabled:      s.SyncEnabled,
		AutoSyncInterval: s.AutoSyncInterval,
		LastSyncError:    s.LastSyncError,
		CreatedAt:        s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:        s.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if s.LastSyncAt != nil {
		t := s.LastSyncAt.UTC().Format("2006-01-02T15:04:05Z")
		r.LastSyncAt = &t
	}
	if s.NextSyncAt != nil {
		t := s.NextSyncAt.UTC().Format("2006-01-02T15:04:05Z")
		r.NextSyncAt = &t
	}
	return r
}

func createSource(svc domain.SourceService, oauth *googleOAuth) http.HandlerFunc {
	type request struct {
		SourceType       string          `json:"source_type"`
		Name             string          `json:"name"`
		Config           json.RawMessage `json:"config"`
		SyncEnabled      bool            `json:"sync_enabled"`
		AutoSyncInterval int             `json:"auto_sync_interval"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}
		if !domain.IsUserCreatableSourceType(domain.SourceType(req.SourceType)) {
			writeJSON(
				w,
				http.StatusBadRequest,
				errBody("source_type must be "+domain.HumanSourceTypeList(domain.UserCreatableSourceTypes())),
			)
			return
		}

		if domain.SourceType(req.SourceType) == domain.SourceTypeGoogleDrive && len(req.Config) > 0 {
			var cfg domain.GoogleDriveConfig
			if err := json.Unmarshal(req.Config, &cfg); err != nil {
				writeJSON(w, http.StatusBadRequest, errBody("invalid google_drive config"))
				return
			}
			if strings.TrimSpace(cfg.RefreshToken) != "" && oauth.enabled() {
				if strings.TrimSpace(cfg.ClientID) == "" {
					cfg.ClientID = oauth.clientID
				}
				if strings.TrimSpace(cfg.ClientSecret) == "" {
					cfg.ClientSecret = oauth.clientSecret
				}
				raw, err := json.Marshal(cfg)
				if err != nil {
					writeJSON(w, http.StatusBadRequest, errBody("invalid google_drive config"))
					return
				}
				req.Config = raw
			}
		}

		src := domain.Source{
			UserID:           auth.UserID(r.Context()),
			Type:             domain.SourceType(req.SourceType),
			Name:             req.Name,
			Config:           req.Config,
			SyncEnabled:      req.SyncEnabled,
			AutoSyncInterval: req.AutoSyncInterval,
			Status:           domain.SourceStatusActive,
		}

		created, err := svc.Create(r.Context(), src)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrConflict):
				writeJSON(w, http.StatusConflict, errBody(err.Error()))
			default:
				writeJSON(w, http.StatusUnprocessableEntity, errBody(err.Error()))
			}
			return
		}
		writeJSON(w, http.StatusCreated, toSourceResponse(created))
	}
}

func getSource(svc domain.SourceService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		userID := auth.UserID(r.Context())

		src, err := svc.GetByID(r.Context(), id, userID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, errBody("source not found"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}
		writeJSON(w, http.StatusOK, toSourceResponse(src))
	}
}

func listSources(svc domain.SourceService) http.HandlerFunc {
	type response struct {
		Sources []sourceResponse `json:"sources"`
		Total   uint64           `json:"total"`
		Offset  uint64           `json:"offset"`
		Limit   uint64           `json:"limit"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		p := parsePage(r)
		userID := auth.UserID(r.Context())

		page, err := svc.List(r.Context(), userID, p)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}

		resp := response{Total: page.Total, Offset: p.Offset, Limit: p.Limit}
		for _, s := range page.Sources {
			resp.Sources = append(resp.Sources, toSourceResponse(s))
		}
		if resp.Sources == nil {
			resp.Sources = []sourceResponse{}
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func deleteSource(svc domain.SourceService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		userID := auth.UserID(r.Context())

		if err := svc.Delete(r.Context(), id, userID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, errBody("source not found"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func syncSource(syncSvc domain.SourceSyncService, trigger func()) http.HandlerFunc {
	type response struct {
		Source     sourceResponse `json:"source"`
		Discovered uint64         `json:"discovered"`
		Queued     uint64         `json:"queued"`
		Updated    uint64         `json:"updated"`
		Unchanged  uint64         `json:"unchanged"`
		Deleted    uint64         `json:"deleted"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		userID := auth.UserID(r.Context())

		res, err := syncSvc.Sync(r.Context(), id, userID)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrNotFound):
				writeJSON(w, http.StatusNotFound, errBody("source not found"))
			default:
				writeJSON(w, http.StatusUnprocessableEntity, errBody(err.Error()))
			}
			return
		}

		if trigger != nil && res.Queued > 0 {
			trigger()
		}

		writeJSON(w, http.StatusOK, response{
			Source:     toSourceResponse(res.Source),
			Discovered: res.Discovered,
			Queued:     res.Queued,
			Updated:    res.Updated,
			Unchanged:  res.Unchanged,
			Deleted:    res.Deleted,
		})
	}
}

func updateSourceCredentials(svc domain.SourceService) http.HandlerFunc {
	type request struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token,omitempty"`
		ClientID     string `json:"client_id,omitempty"`
		ClientSecret string `json:"client_secret,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		userID := auth.UserID(r.Context())

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}

		updated, err := svc.UpdateGoogleDriveCredentials(r.Context(), id, userID, domain.GoogleDriveCredentialUpdate{
			AccessToken:  req.AccessToken,
			RefreshToken: req.RefreshToken,
			ClientID:     req.ClientID,
			ClientSecret: req.ClientSecret,
		})
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrNotFound):
				writeJSON(w, http.StatusNotFound, errBody("source not found"))
			default:
				writeJSON(w, http.StatusUnprocessableEntity, errBody(err.Error()))
			}
			return
		}

		writeJSON(w, http.StatusOK, toSourceResponse(updated))
	}
}

func updateSourceSelection(svc domain.SourceService) http.HandlerFunc {
	type request struct {
		SelectedFileIDs   []string `json:"selected_file_ids"`
		SelectedFolderIDs []string `json:"selected_folder_ids,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		userID := auth.UserID(r.Context())

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}

		updated, err := svc.UpdateGoogleDriveSelection(r.Context(), id, userID, domain.GoogleDriveSelectionUpdate{
			SelectedFileIDs:   req.SelectedFileIDs,
			SelectedFolderIDs: req.SelectedFolderIDs,
		})
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrNotFound):
				writeJSON(w, http.StatusNotFound, errBody("source not found"))
			default:
				writeJSON(w, http.StatusUnprocessableEntity, errBody(err.Error()))
			}
			return
		}

		writeJSON(w, http.StatusOK, toSourceResponse(updated))
	}
}

func parsePage(r *http.Request) domain.Page {
	limit, _ := strconv.ParseUint(r.URL.Query().Get("limit"), 10, 64)
	offset, _ := strconv.ParseUint(r.URL.Query().Get("offset"), 10, 64)
	if limit == 0 {
		limit = 20
	}
	if limit > 1000 {
		limit = 1000
	}
	return domain.Page{Limit: limit, Offset: offset}
}

func parseDriveFolderID(folderID, folderLink string) string {
	if trimmed := strings.TrimSpace(folderID); trimmed != "" {
		return trimmed
	}
	raw := strings.TrimSpace(folderLink)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		return raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "folders" {
			return parts[i+1]
		}
	}
	return raw
}

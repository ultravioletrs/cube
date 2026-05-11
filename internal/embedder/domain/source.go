package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// Sentinel errors returned by repositories and services.
var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

// SourceType identifies the external system a source connects to.
type SourceType string

const (
	SourceTypeGoogleDrive SourceType = "google_drive"
	SourceTypeLocalFS     SourceType = "local_fs"
	SourceTypeSharePoint  SourceType = "sharepoint"
	SourceTypeRclone      SourceType = "rclone"
)

// SourceStatus represents the current operational state of a source.
type SourceStatus string

const (
	SourceStatusActive       SourceStatus = "active"
	SourceStatusSyncing      SourceStatus = "syncing"
	SourceStatusError        SourceStatus = "error"
	SourceStatusDisconnected SourceStatus = "disconnected"
)

// Source represents an external integration origin from which documents are ingested.
// Only Google Drive is actively implemented; the model is intentionally open to
// additional source types (SharePoint, local filesystem, etc.).
type Source struct {
	ID string
	// UserID scopes this source to a single authenticated user.
	UserID string
	Type   SourceType
	Name   string
	// Config holds type-specific configuration as JSON (JSONB in PostgreSQL).
	// See GoogleDriveConfig and future *Config types for the expected shapes.
	Config           json.RawMessage
	Status           SourceStatus
	SyncEnabled      bool
	AutoSyncInterval int // minutes between auto-syncs; 0 = manual only
	LastSyncAt       *time.Time
	LastSyncError    *string
	NextSyncAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// GoogleDriveConfig is the expected shape of Source.Config for SourceTypeGoogleDrive.
// OAuth credentials are populated externally by the auth/OAuth flow.
type GoogleDriveConfig struct {
	ServiceAccountJSON string   `json:"service_account_json,omitempty"`
	FolderID           string   `json:"folder_id,omitempty"`
	FolderLink         string   `json:"folder_link,omitempty"`
	SelectedFileIDs    []string `json:"selected_file_ids,omitempty"`
	SelectedFolderIDs  []string `json:"selected_folder_ids,omitempty"`
	ClientID           string   `json:"client_id,omitempty"`
	ClientSecret       string   `json:"client_secret,omitempty"`
	AccessToken        string   `json:"access_token,omitempty"`
	RefreshToken       string   `json:"refresh_token,omitempty"`
}

// RcloneConfig is the expected shape of Source.Config for SourceTypeRclone.
// It describes a constrained sync target within one rclone remote.
type RcloneConfig struct {
	Remote        string   `json:"remote,omitempty"`
	RootPath      string   `json:"root_path,omitempty"`
	ScopePaths    []string `json:"scope_paths,omitempty"`
	SelectedPaths []string `json:"selected_paths,omitempty"`
	ConfigRef     string   `json:"config_ref,omitempty"`
}

// GoogleDriveCredentialUpdate carries mutable OAuth fields for an existing source.
type GoogleDriveCredentialUpdate struct {
	AccessToken  string
	RefreshToken string
	ClientID     string
	ClientSecret string
}

// GoogleDriveSelectionUpdate carries mutable file/folder scope fields.
type GoogleDriveSelectionUpdate struct {
	SelectedFileIDs   []string
	SelectedFolderIDs []string
}

// Page carries pagination parameters for list queries.
type Page struct {
	Offset uint64
	Limit  uint64
}

// SourcePage is a paginated result set of sources.
type SourcePage struct {
	Sources []Source
	Total   uint64
}

// SourceSyncResult summarizes a source synchronization pass.
type SourceSyncResult struct {
	Source     Source
	Discovered uint64
	Queued     uint64
	Updated    uint64
	Unchanged  uint64
	Deleted    uint64
}

// SourceRepository defines the persistence contract for sources.
type SourceRepository interface {
	Create(ctx context.Context, s Source) (Source, error)
	GetByID(ctx context.Context, id, userID string) (Source, error)
	List(ctx context.Context, userID string, p Page) (SourcePage, error)
	Delete(ctx context.Context, id, userID string) error
	UpdateSyncResult(
		ctx context.Context,
		id, userID string,
		status SourceStatus,
		lastSyncAt time.Time,
		lastSyncError *string,
	) (Source, error)
	UpdateConfig(ctx context.Context, id, userID string, config json.RawMessage) (Source, error)
}

// SourceService defines the business-logic contract for sources.
type SourceService interface {
	Create(ctx context.Context, s Source) (Source, error)
	GetByID(ctx context.Context, id, userID string) (Source, error)
	List(ctx context.Context, userID string, p Page) (SourcePage, error)
	Delete(ctx context.Context, id, userID string) error
	UpdateGoogleDriveCredentials(ctx context.Context, id, userID string, update GoogleDriveCredentialUpdate) (Source, error)
	UpdateGoogleDriveSelection(ctx context.Context, id, userID string, update GoogleDriveSelectionUpdate) (Source, error)
}

// SourceSyncService defines source-specific synchronization behavior.
type SourceSyncService interface {
	Sync(ctx context.Context, id, userID string) (SourceSyncResult, error)
}

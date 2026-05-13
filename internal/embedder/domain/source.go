// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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
	SourceTypeMicrosoft   SourceType = "microsoft"
	SourceTypeOneDrive    SourceType = "onedrive"
	SourceTypeSharePoint  SourceType = "sharepoint"
	SourceTypeDropbox     SourceType = "dropbox"
	SourceTypeS3          SourceType = "s3"
	SourceTypeRclone      SourceType = "rclone"
)

var supportedSourceTypes = []SourceType{
	SourceTypeLocalFS,
	SourceTypeRclone,
	SourceTypeDropbox,
	SourceTypeGoogleDrive,
	SourceTypeS3,
	SourceTypeMicrosoft,
	SourceTypeOneDrive,
	SourceTypeSharePoint,
}

var userCreatableSourceTypes = []SourceType{
	SourceTypeGoogleDrive,
	SourceTypeS3,
	SourceTypeMicrosoft,
	SourceTypeOneDrive,
	SourceTypeSharePoint,
	SourceTypeDropbox,
	SourceTypeRclone,
}

var sourceProviderAliases = map[SourceType]SourceType{
	SourceTypeOneDrive:   SourceTypeMicrosoft,
	SourceTypeSharePoint: SourceTypeMicrosoft,
	SourceTypeDropbox:    SourceTypeRclone,
}

// SupportedSourceTypes returns all source types accepted by the service layer.
func SupportedSourceTypes() []SourceType {
	out := make([]SourceType, len(supportedSourceTypes))
	copy(out, supportedSourceTypes)
	return out
}

// UserCreatableSourceTypes returns source types accepted by the create-source API.
func UserCreatableSourceTypes() []SourceType {
	out := make([]SourceType, len(userCreatableSourceTypes))
	copy(out, userCreatableSourceTypes)
	return out
}

// SourceProviderAliases returns source-type aliases mapped to canonical provider-backed types.
func SourceProviderAliases() map[SourceType]SourceType {
	out := make(map[SourceType]SourceType, len(sourceProviderAliases))
	for alias, target := range sourceProviderAliases {
		out[alias] = target
	}
	return out
}

// IsSupportedSourceType checks whether the source type is accepted by the service layer.
func IsSupportedSourceType(t SourceType) bool {
	return containsSourceType(supportedSourceTypes, t)
}

// IsUserCreatableSourceType checks whether the source type is allowed through source-create APIs.
func IsUserCreatableSourceType(t SourceType) bool {
	return containsSourceType(userCreatableSourceTypes, t)
}

// IsRcloneBackedSourceType reports whether a source type is implemented by rclone.
func IsRcloneBackedSourceType(t SourceType) bool {
	return t == SourceTypeRclone || t == SourceTypeDropbox
}

// HumanSourceTypeList formats a list of source types for error/help messages.
func HumanSourceTypeList(types []SourceType) string {
	if len(types) == 0 {
		return ""
	}
	values := make([]string, 0, len(types))
	for _, t := range types {
		values = append(values, string(t))
	}
	if len(values) == 1 {
		return values[0]
	}
	return strings.Join(values[:len(values)-1], ", ") + " or " + values[len(values)-1]
}

func containsSourceType(values []SourceType, target SourceType) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// SourceStatus represents the current operational state of a source.
type SourceStatus string

const (
	SourceStatusActive       SourceStatus = "active"
	SourceStatusSyncing      SourceStatus = "syncing"
	SourceStatusError        SourceStatus = "error"
	SourceStatusDisconnected SourceStatus = "disconnected"
)

// Source represents an external integration origin from which documents are ingested.
// The model is intentionally open to additional source types.
type Source struct {
	ID string
	// DomainID scopes this source to a domain (workspace); all domain members share it.
	DomainID string
	// UserID records the creating user for audit purposes.
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
	Backend       string   `json:"backend,omitempty"`
	Remote        string   `json:"remote,omitempty"`
	RootPath      string   `json:"root_path,omitempty"`
	ScopePaths    []string `json:"scope_paths,omitempty"`
	SelectedPaths []string `json:"selected_paths,omitempty"`
	ConfigRef     string   `json:"config_ref,omitempty"`
}

// S3Config is the expected shape of Source.Config for SourceTypeS3.
// It describes one S3 bucket and constrained object-key scopes.
type S3Config struct {
	Endpoint        string   `json:"endpoint,omitempty"`
	Region          string   `json:"region,omitempty"`
	Bucket          string   `json:"bucket,omitempty"`
	AccessKeyID     string   `json:"access_key_id,omitempty"`
	SecretAccessKey string   `json:"secret_access_key,omitempty"`
	SessionToken    string   `json:"session_token,omitempty"`
	UseSSL          *bool    `json:"use_ssl,omitempty"`
	PathStyle       *bool    `json:"path_style,omitempty"`
	RootPath        string   `json:"root_path,omitempty"`
	ScopePaths      []string `json:"scope_paths,omitempty"`
	SelectedPaths   []string `json:"selected_paths,omitempty"`
	ConfigRef       string   `json:"config_ref,omitempty"`
}

// MicrosoftConfig is the expected shape of Source.Config for SourceTypeMicrosoft.
// It supports OneDrive and SharePoint drives through Microsoft Graph.
type MicrosoftConfig struct {
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
	GetByID(ctx context.Context, id, domainID string) (Source, error)
	List(ctx context.Context, domainID string, p Page) (SourcePage, error)
	Delete(ctx context.Context, id, domainID string) error
	UpdateSyncResult(
		ctx context.Context,
		id, domainID string,
		status SourceStatus,
		lastSyncAt time.Time,
		lastSyncError *string,
	) (Source, error)
	UpdateConfig(ctx context.Context, id, domainID string, config json.RawMessage) (Source, error)
}

// SourceService defines the business-logic contract for sources.
type SourceService interface {
	Create(ctx context.Context, s Source) (Source, error)
	GetByID(ctx context.Context, id, domainID string) (Source, error)
	List(ctx context.Context, domainID string, p Page) (SourcePage, error)
	Delete(ctx context.Context, id, domainID string) error
	UpdateGoogleDriveCredentials(ctx context.Context, id, domainID string, update GoogleDriveCredentialUpdate) (Source, error)
	UpdateGoogleDriveSelection(ctx context.Context, id, domainID string, update GoogleDriveSelectionUpdate) (Source, error)
}

// SourceSyncService defines source-specific synchronization behavior.
type SourceSyncService interface {
	Sync(ctx context.Context, id, domainID string) (SourceSyncResult, error)
}

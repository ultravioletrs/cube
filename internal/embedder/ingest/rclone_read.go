// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

// DownloadFromRcloneSource downloads and extracts a record via rclone.
func DownloadFromRcloneSource(
	ctx context.Context,
	rec domain.Record,
	src domain.Source,
) (string, *int, error) {
	if rec.ExternalID == "" && rec.ExternalRef == "" {
		return "", nil, fmt.Errorf("record %s is missing external source reference", rec.ID)
	}

	var cfg domain.RcloneConfig
	if err := json.Unmarshal(src.Config, &cfg); err != nil {
		return "", nil, fmt.Errorf("decode rclone source config: %w", err)
	}

	remote := sanitizeRcloneRemote(cfg.Remote)
	if remote == "" {
		return "", nil, fmt.Errorf("rclone source %s has invalid remote", src.ID)
	}

	path := normalizeRclonePath(rec.ExternalRef)
	if path == "" {
		path = normalizeRclonePath(rec.ExternalID)
	}
	target := remote + ":"
	if path != "" {
		target += path
	}

	configDir := strings.TrimSpace(os.Getenv("EMBEDDER_RCLONE_CONFIG_DIR"))
	if configDir == "" {
		configDir = "/etc/cube/rclone"
	}
	configPath := filepath.Join(
		configDir,
		sanitizePathSegment(rec.UserID),
		sanitizePathSegment(src.ID),
		"rclone.conf",
	)
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			configPath = filepath.Join(configDir, "rclone.conf")
		} else {
			return "", nil, fmt.Errorf("check rclone config path: %w", err)
		}
	}

	timeout := 2 * time.Minute
	if raw := strings.TrimSpace(os.Getenv("EMBEDDER_RCLONE_TIMEOUT")); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			timeout = parsed
		}
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	binary := strings.TrimSpace(os.Getenv("EMBEDDER_RCLONE_BINARY"))
	if binary == "" {
		binary = "rclone"
	}
	cmd := exec.CommandContext(runCtx, binary, "--config", configPath, "cat", target)

	body, err := cmd.Output()
	if err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return "", nil, fmt.Errorf("rclone read timed out for %q", target)
		}
		return "", nil, fmt.Errorf("rclone read failed for %q: %w", target, err)
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

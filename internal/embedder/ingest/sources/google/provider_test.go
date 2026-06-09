// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package google_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/ingest"
	"github.com/ultravioletrs/cube/internal/embedder/ingest/sources/google"
)

func TestGoogleSourceProvider_ListAndDownload_Smoke(t *testing.T) {
	const accessToken = "google-test-token"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := strings.TrimSpace(r.Header.Get("Authorization")); got != "Bearer "+accessToken {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/drive/v3/files":
			_, _ = io.WriteString(w, `{
				"files":[
					{
						"id":"file-1",
						"name":"notes.txt",
						"mimeType":"text/plain",
						"version":"3",
						"modifiedTime":"2026-05-12T10:00:00Z",
						"webViewLink":"https://drive.example.local/file-1",
						"parents":["folder-1"]
					}
				]
			}`)
			return
		case r.Method == http.MethodGet && r.URL.Path == "/drive/v3/files/file-1":
			if r.URL.Query().Get("alt") != "media" {
				http.Error(w, "missing alt=media", http.StatusBadRequest)
				return
			}
			_, _ = io.WriteString(w, "hello from google drive")
			return
		default:
			http.Error(w, "not found", http.StatusNotFound)
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			return
		}
	}))
	defer srv.Close()

	restore := ingest.SetDriveAPIEndpoints(
		srv.URL+"/drive/v3/files",
		srv.URL+"/drive/v3/files/%s/export",
		srv.URL+"/drive/v3/files/%s?alt=media",
	)
	defer restore()

	cfgRaw, err := json.Marshal(domain.GoogleDriveConfig{
		AccessToken: accessToken,
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	provider := google.NewSourceProvider()
	src := domain.Source{
		ID:     "src-g1",
		Type:   domain.SourceTypeGoogleDrive,
		Config: cfgRaw,
	}

	files, err := provider.ListFiles(context.Background(), "user-1", src)
	if err != nil {
		t.Fatalf("ListFiles returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d: %#v", len(files), files)
	}
	if files[0].ExternalID != "file-1" {
		t.Fatalf("unexpected external_id: %q", files[0].ExternalID)
	}
	if files[0].Name != "notes.txt" {
		t.Fatalf("unexpected file name: %q", files[0].Name)
	}

	text, pageCount, err := provider.DownloadRecord(context.Background(), domain.Record{
		ID:         "rec-g1",
		ExternalID: "file-1",
		Name:       "notes.txt",
		MimeType:   "text/plain",
	}, src)
	if err != nil {
		t.Fatalf("DownloadRecord returned error: %v", err)
	}
	if text != "hello from google drive" {
		t.Fatalf("unexpected download text: %q", text)
	}
	if pageCount != nil {
		t.Fatalf("expected nil page count, got %d", *pageCount)
	}
}

func TestGoogleSourceProvider_SelectedFileOutsideConfiguredFolder(t *testing.T) {
	const accessToken = "google-test-token"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := strings.TrimSpace(r.Header.Get("Authorization")); got != "Bearer "+accessToken {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/drive/v3/files":
			_, _ = io.WriteString(w, `{
				"files":[
					{
						"id":"local-file",
						"name":"local.txt",
						"mimeType":"text/plain",
						"version":"1",
						"modifiedTime":"2026-05-12T10:00:00Z",
						"webViewLink":"https://drive.example.local/local-file",
						"parents":["configured-folder"]
					}
				]
			}`)
			return
		case r.Method == http.MethodGet && r.URL.Path == "/drive/v3/files/nested-file":
			if r.URL.Query().Get("fields") == "" {
				http.Error(w, "missing fields", http.StatusBadRequest)
				return
			}
			_, _ = io.WriteString(w, `{
				"id":"nested-file",
				"name":"nested.md",
				"mimeType":"text/markdown",
				"version":"7",
				"modifiedTime":"2026-05-12T11:00:00Z",
				"webViewLink":"https://drive.example.local/nested-file",
				"parents":["other-folder"]
			}`)
			return
		default:
			http.Error(w, "not found", http.StatusNotFound)
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			return
		}
	}))
	defer srv.Close()

	restore := ingest.SetDriveAPIEndpoints(
		srv.URL+"/drive/v3/files",
		srv.URL+"/drive/v3/files/%s/export",
		srv.URL+"/drive/v3/files/%s?alt=media",
	)
	defer restore()

	cfgRaw, err := json.Marshal(domain.GoogleDriveConfig{
		AccessToken:     accessToken,
		FolderID:        "configured-folder",
		SelectedFileIDs: []string{"nested-file"},
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	files, err := google.NewSourceProvider().ListFiles(context.Background(), "user-1", domain.Source{
		ID:     "src-g2",
		Type:   domain.SourceTypeGoogleDrive,
		Config: cfgRaw,
	})
	if err != nil {
		t.Fatalf("ListFiles returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 selected file, got %d: %#v", len(files), files)
	}
	if files[0].ExternalID != "nested-file" {
		t.Fatalf("unexpected external_id: %q", files[0].ExternalID)
	}
	if files[0].FolderID != "other-folder" {
		t.Fatalf("unexpected folder id: %q", files[0].FolderID)
	}
}

// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/ingest/sources/microsoft"
)

func TestMicrosoftSourceProvider_ListAndDownload_Smoke(t *testing.T) {
	t.Helper()

	const (
		accessToken = "test-access-token"
		driveID     = "drv1"
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := strings.TrimSpace(r.Header.Get("Authorization")); got != "Bearer "+accessToken {
			http.Error(w, `{"error":{"code":"Unauthorized","message":"missing token"}}`, http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case "/v1.0/drives/" + driveID + "/root:/team/docs:":
			_, _ = io.WriteString(w, `{
				"id":"folder-root",
				"name":"docs",
				"folder":{"childCount":2}
			}`)
			return
		case "/v1.0/drives/" + driveID + "/items/folder-root/children":
			_, _ = io.WriteString(w, `{
				"value":[
					{
						"id":"file-root",
						"name":"a.txt",
						"webUrl":"https://example.local/a.txt",
						"lastModifiedDateTime":"2026-05-12T10:00:00Z",
						"eTag":"etag-a",
						"size":15,
						"file":{"mimeType":"text/plain"}
					},
					{
						"id":"folder-sub",
						"name":"sub",
						"folder":{"childCount":1}
					}
				]
			}`)
			return
		case "/v1.0/drives/" + driveID + "/items/folder-sub/children":
			_, _ = io.WriteString(w, `{
				"value":[
					{
						"id":"file-sub",
						"name":"b.txt",
						"webUrl":"https://example.local/b.txt",
						"lastModifiedDateTime":"2026-05-12T11:00:00Z",
						"eTag":"etag-b",
						"size":17,
						"file":{"mimeType":"text/plain"}
					}
				]
			}`)
			return
		case "/v1.0/drives/" + driveID + "/items/file-sub/content":
			_, _ = io.WriteString(w, "hello microsoft source")
			return
		default:
			http.Error(w, `{"error":{"code":"NotFound","message":"unexpected path"}}`, http.StatusNotFound)
			t.Errorf("unexpected Microsoft Graph request path: %s", r.URL.String())
			return
		}
	}))
	defer srv.Close()

	cfg := domain.MicrosoftConfig{
		AccessToken:   accessToken,
		DriveID:       driveID,
		RootPath:      "team/docs",
		ScopePaths:    []string{"team/docs"},
		SelectedPaths: []string{"team/docs/sub/b.txt"},
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal microsoft config: %v", err)
	}

	provider := microsoft.NewSourceProviderWithHTTPClient(newGraphRedirectHTTPClient(t, srv.URL))
	src := domain.Source{
		ID:     "src-1",
		Type:   domain.SourceTypeMicrosoft,
		Config: raw,
	}

	files, err := provider.ListFiles(context.Background(), "user-1", src)
	if err != nil {
		t.Fatalf("ListFiles returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 selected file, got %d: %#v", len(files), files)
	}
	if files[0].ExternalID != "file-sub" {
		t.Fatalf("unexpected external id: %q", files[0].ExternalID)
	}
	if files[0].ExternalRef != "team/docs/sub/b.txt" {
		t.Fatalf("unexpected external ref: %q", files[0].ExternalRef)
	}
	if files[0].MimeType != "text/plain" {
		t.Fatalf("unexpected mime type: %q", files[0].MimeType)
	}

	text, pageCount, err := provider.DownloadRecord(context.Background(), domain.Record{
		ID:         "rec-1",
		ExternalID: "file-sub",
		Name:       "b.txt",
		MimeType:   "text/plain",
	}, src)
	if err != nil {
		t.Fatalf("DownloadRecord returned error: %v", err)
	}
	if text != "hello microsoft source" {
		t.Fatalf("unexpected extracted text: %q", text)
	}
	if pageCount != nil {
		t.Fatalf("expected nil page count for text file, got %v", *pageCount)
	}
}

func newGraphRedirectHTTPClient(t *testing.T, serverURL string) *http.Client {
	t.Helper()

	target, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("parse test server url: %v", err)
	}

	return &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			clone := req.Clone(req.Context())
			copied := *clone.URL
			copied.Scheme = target.Scheme
			copied.Host = target.Host
			clone.URL = &copied
			return http.DefaultTransport.RoundTrip(clone)
		}),
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

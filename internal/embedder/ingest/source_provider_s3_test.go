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
	s3source "github.com/ultravioletrs/cube/internal/embedder/ingest/sources/s3"
)

func TestS3SourceProvider_ListAndDownload_Smoke(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "AWS4-HMAC-SHA256 ") {
			http.Error(w, "missing aws signature", http.StatusUnauthorized)
			return
		}

		switch {
		case r.Method == http.MethodGet &&
			(r.URL.Path == "/docs" || r.URL.Path == "/docs/") &&
			r.URL.Query().Get("list-type") == "2":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Name>docs</Name>
  <Prefix>team/docs/</Prefix>
  <KeyCount>2</KeyCount>
  <MaxKeys>1000</MaxKeys>
  <IsTruncated>false</IsTruncated>
  <Contents>
    <Key>team/docs/a.txt</Key>
    <LastModified>2026-05-12T10:00:00.000Z</LastModified>
    <ETag>"etag-a"</ETag>
    <Size>12</Size>
    <StorageClass>STANDARD</StorageClass>
  </Contents>
  <Contents>
    <Key>team/docs/sub/b.txt</Key>
    <LastModified>2026-05-12T11:00:00.000Z</LastModified>
    <ETag>"etag-b"</ETag>
    <Size>21</Size>
    <StorageClass>STANDARD</StorageClass>
  </Contents>
</ListBucketResult>`)
			return
		case r.Method == http.MethodGet && r.URL.Path == "/docs/team/docs/sub/b.txt":
			w.Header().Set("Last-Modified", "Tue, 12 May 2026 11:00:00 GMT")
			w.Header().Set("ETag", `"etag-b"`)
			_, _ = io.WriteString(w, "hello from s3")
			return
		default:
			http.Error(w, "not found", http.StatusNotFound)
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			return
		}
	}))
	defer srv.Close()

	srvURL, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse httptest server url: %v", err)
	}

	cfgRaw, err := json.Marshal(domain.S3Config{
		Endpoint:        srvURL.Host,
		Region:          "us-east-1",
		Bucket:          "docs",
		AccessKeyID:     "test-access",
		SecretAccessKey: "test-secret",
		UseSSL:          testBoolPtr(false),
		PathStyle:       testBoolPtr(true),
		RootPath:        "team/docs",
		ScopePaths:      []string{"team/docs"},
		SelectedPaths:   []string{"team/docs/sub/b.txt"},
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	provider := s3source.NewSourceProvider()
	src := domain.Source{
		ID:     "src-s3-1",
		Type:   domain.SourceTypeS3,
		Config: cfgRaw,
	}

	files, err := provider.ListFiles(context.Background(), "user-1", src)
	if err != nil {
		t.Fatalf("ListFiles returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 selected file, got %d: %#v", len(files), files)
	}
	if files[0].ExternalID != "team/docs/sub/b.txt" {
		t.Fatalf("unexpected external id: %q", files[0].ExternalID)
	}

	text, pageCount, err := provider.DownloadRecord(context.Background(), domain.Record{
		ID:         "rec-s3-1",
		ExternalID: "team/docs/sub/b.txt",
		Name:       "b.txt",
		MimeType:   "text/plain",
	}, src)
	if err != nil {
		t.Fatalf("DownloadRecord returned error: %v", err)
	}
	if text != "hello from s3" {
		t.Fatalf("unexpected extracted text: %q", text)
	}
	if pageCount != nil {
		t.Fatalf("expected nil page count, got %d", *pageCount)
	}
}

func testBoolPtr(v bool) *bool {
	return &v
}

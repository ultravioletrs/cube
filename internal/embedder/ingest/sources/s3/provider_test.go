// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package s3_test

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
		SelectedPaths:   []string{"team/docs/sub"},
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
	if files[0].MimeType != "text/plain" {
		t.Fatalf("expected inferred text/plain mime, got %q", files[0].MimeType)
	}

	text, pageCount, err := provider.DownloadRecord(context.Background(), domain.Record{
		ID:         "rec-s3-1",
		ExternalID: "team/docs/sub/b.txt",
		Name:       "b",
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

func TestS3SourceProvider_ListFilesInfersMIMEFromObjectKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet ||
			(r.URL.Path != "/docs" && r.URL.Path != "/docs/") ||
			r.URL.Query().Get("list-type") != "2" {
			http.Error(w, "not found", http.StatusNotFound)
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			return
		}

		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Name>docs</Name>
  <Prefix>team/docs/</Prefix>
  <KeyCount>4</KeyCount>
  <MaxKeys>1000</MaxKeys>
  <IsTruncated>false</IsTruncated>
  <Contents>
    <Key>team/docs/report.pdf</Key>
    <LastModified>2026-05-12T10:00:00.000Z</LastModified>
    <ETag>"etag-pdf"</ETag>
    <Size>1024</Size>
    <StorageClass>STANDARD</StorageClass>
  </Contents>
  <Contents>
    <Key>team/docs/notes.md</Key>
    <LastModified>2026-05-12T10:00:00.000Z</LastModified>
    <ETag>"etag-md"</ETag>
    <Size>12</Size>
    <StorageClass>STANDARD</StorageClass>
  </Contents>
  <Contents>
    <Key>team/docs/scan.png</Key>
    <LastModified>2026-05-12T10:00:00.000Z</LastModified>
    <ETag>"etag-png"</ETag>
    <Size>256</Size>
    <StorageClass>STANDARD</StorageClass>
  </Contents>
  <Contents>
    <Key>team/docs/archive.bin</Key>
    <LastModified>2026-05-12T10:00:00.000Z</LastModified>
    <ETag>"etag-bin"</ETag>
    <Size>64</Size>
    <StorageClass>STANDARD</StorageClass>
  </Contents>
</ListBucketResult>`)
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
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	provider := s3source.NewSourceProvider()
	files, err := provider.ListFiles(context.Background(), "user-1", domain.Source{
		ID:     "src-s3-1",
		Type:   domain.SourceTypeS3,
		Config: cfgRaw,
	})
	if err != nil {
		t.Fatalf("ListFiles returned error: %v", err)
	}

	got := make(map[string]string, len(files))
	for _, file := range files {
		got[file.Name] = file.MimeType
	}
	want := map[string]string{
		"archive.bin": "",
		"notes.md":    "text/markdown",
		"report.pdf":  "application/pdf",
		"scan.png":    "image/png",
	}
	for name, mimeType := range want {
		if got[name] != mimeType {
			t.Fatalf("expected %s mime %q, got %q in %#v", name, mimeType, got[name], got)
		}
	}
}

func TestBrowseS3PathPagePaginatesFlatListing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet ||
			(r.URL.Path != "/docs" && r.URL.Path != "/docs/") ||
			r.URL.Query().Get("list-type") != "2" {
			http.Error(w, "not found", http.StatusNotFound)
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			return
		}
		if r.URL.Query().Get("max-keys") != "3" {
			http.Error(w, "bad max-keys", http.StatusBadRequest)
			t.Errorf("expected max-keys=3, got %q", r.URL.Query().Get("max-keys"))
			return
		}

		w.Header().Set("Content-Type", "application/xml")
		switch r.URL.Query().Get("start-after") {
		case "":
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Name>docs</Name>
  <Prefix></Prefix>
  <KeyCount>3</KeyCount>
  <MaxKeys>3</MaxKeys>
  <Delimiter>/</Delimiter>
  <IsTruncated>false</IsTruncated>
  <Contents>
    <Key>a.txt</Key>
    <LastModified>2026-05-12T10:00:00.000Z</LastModified>
    <ETag>"etag-a"</ETag>
    <Size>12</Size>
    <StorageClass>STANDARD</StorageClass>
  </Contents>
  <Contents>
    <Key>b.txt</Key>
    <LastModified>2026-05-12T11:00:00.000Z</LastModified>
    <ETag>"etag-b"</ETag>
    <Size>21</Size>
    <StorageClass>STANDARD</StorageClass>
  </Contents>
  <CommonPrefixes>
    <Prefix>c/</Prefix>
  </CommonPrefixes>
</ListBucketResult>`)
		case "b.txt":
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Name>docs</Name>
  <Prefix></Prefix>
  <KeyCount>2</KeyCount>
  <MaxKeys>3</MaxKeys>
  <Delimiter>/</Delimiter>
  <IsTruncated>false</IsTruncated>
  <CommonPrefixes>
    <Prefix>c/</Prefix>
  </CommonPrefixes>
  <Contents>
    <Key>d.txt</Key>
    <LastModified>2026-05-12T12:00:00.000Z</LastModified>
    <ETag>"etag-d"</ETag>
    <Size>22</Size>
    <StorageClass>STANDARD</StorageClass>
  </Contents>
</ListBucketResult>`)
		default:
			http.Error(w, "bad start-after", http.StatusBadRequest)
			t.Errorf("unexpected start-after %q", r.URL.Query().Get("start-after"))
		}
	}))
	defer srv.Close()

	srvURL, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse httptest server url: %v", err)
	}

	cfg := domain.S3Config{
		Endpoint:        srvURL.Host,
		Region:          "us-east-1",
		Bucket:          "docs",
		AccessKeyID:     "test-access",
		SecretAccessKey: "test-secret",
		UseSSL:          testBoolPtr(false),
		PathStyle:       testBoolPtr(true),
	}

	first, err := s3source.BrowseS3PathPage(context.Background(), cfg, "", 2, "")
	if err != nil {
		t.Fatalf("BrowseS3PathPage first page returned error: %v", err)
	}
	if len(first.Entries) != 2 || first.Entries[0].Path != "a.txt" || first.Entries[1].Path != "b.txt" {
		t.Fatalf("unexpected first page: %#v", first.Entries)
	}
	if first.Entries[0].MimeType != "text/plain" || first.Entries[1].MimeType != "text/plain" {
		t.Fatalf("expected inferred text/plain mimes, got %#v", first.Entries)
	}
	if !first.HasMore || first.NextPageToken != "b.txt" {
		t.Fatalf("expected first page to continue after b.txt, got has_more=%v token=%q", first.HasMore, first.NextPageToken)
	}

	second, err := s3source.BrowseS3PathPage(context.Background(), cfg, "", 2, first.NextPageToken)
	if err != nil {
		t.Fatalf("BrowseS3PathPage second page returned error: %v", err)
	}
	if second.HasMore || second.NextPageToken != "" {
		t.Fatalf("expected second page to end, got has_more=%v token=%q", second.HasMore, second.NextPageToken)
	}
	if len(second.Entries) != 2 {
		t.Fatalf("expected 2 second-page entries, got %d: %#v", len(second.Entries), second.Entries)
	}
	if !second.Entries[0].IsDir || second.Entries[0].Path != "c" {
		t.Fatalf("expected directory c first on second page, got %#v", second.Entries[0])
	}
	if second.Entries[1].Path != "d.txt" {
		t.Fatalf("expected d.txt second on second page, got %#v", second.Entries[1])
	}
}

func testBoolPtr(v bool) *bool {
	return &v
}

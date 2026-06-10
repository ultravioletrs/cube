// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package localfs

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

type fakeStore struct {
	objects map[string][]byte
}

func (f *fakeStore) Put(context.Context, string, string, int64, io.Reader) error { return nil }
func (f *fakeStore) Delete(context.Context, string) error                        { return nil }
func (f *fakeStore) Get(_ context.Context, key string) ([]byte, error) {
	body, ok := f.objects[key]
	if !ok {
		return nil, os.ErrNotExist
	}
	return body, nil
}

func TestDownloadRecordContent_ObjectStoreKey(t *testing.T) {
	store := &fakeStore{objects: map[string][]byte{"user-1/file.txt": []byte("hello")}}
	p := &sourceProvider{store: store}

	body, err := p.DownloadRecordContent(context.Background(),
		domain.Record{ID: "r1", ExternalID: "user-1/file.txt"},
		domain.Source{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "hello" {
		t.Fatalf("got %q, want %q", body, "hello")
	}
}

func TestDownloadRecordContent_ObjectStoreKeyNoStore(t *testing.T) {
	p := &sourceProvider{store: nil}
	_, err := p.DownloadRecordContent(context.Background(),
		domain.Record{ID: "r1", ExternalID: "user-1/file.txt"},
		domain.Source{})
	if err == nil {
		t.Fatal("expected error when object storage is not configured")
	}
}

func TestDownloadRecordContent_LegacyUploadDir(t *testing.T) {
	dir := t.TempDir()
	userDir := filepath.Join(dir, "user-1")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "doc.txt"), []byte("legacy"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, _ := json.Marshal(uploadConfig{UploadDir: dir})
	p := &sourceProvider{}

	body, err := p.DownloadRecordContent(context.Background(),
		domain.Record{ID: "r1", UserID: "user-1", ExternalID: "doc.txt"},
		domain.Source{Config: cfg})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "legacy" {
		t.Fatalf("got %q, want %q", body, "legacy")
	}
}

func TestDownloadRecordContent_MissingExternalID(t *testing.T) {
	p := &sourceProvider{}
	_, err := p.DownloadRecordContent(context.Background(),
		domain.Record{ID: "r1"}, domain.Source{})
	if err == nil {
		t.Fatal("expected error for missing external_id")
	}
}

func TestDownloadRecordContent_InvalidLocalFileID(t *testing.T) {
	cfg, _ := json.Marshal(uploadConfig{UploadDir: t.TempDir()})
	p := &sourceProvider{}
	_, err := p.DownloadRecordContent(context.Background(),
		domain.Record{ID: "r1", UserID: "user-1", ExternalID: "sub.dir.name"}, // no slash -> legacy path
		domain.Source{Config: cfg})
	if err == nil {
		t.Fatal("expected read error for non-existent legacy file")
	}
}

func TestListFilesEmpty(t *testing.T) {
	p := &sourceProvider{}
	files, err := p.ListFiles(context.Background(), "user-1", domain.Source{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no files, got %d", len(files))
	}
}

func TestCapabilities(t *testing.T) {
	p := &sourceProvider{}
	if p.Type() != domain.SourceTypeLocalFS {
		t.Fatalf("got type %q", p.Type())
	}
	caps := p.Capabilities()
	if !caps.SupportsDownload || caps.SupportsList {
		t.Fatalf("unexpected capabilities: %+v", caps)
	}
	if p.PrunesStaleRecords() {
		t.Fatal("local_fs must not prune stale records")
	}
}

// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"testing"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/ingest"
)

func TestRecordFormatFromDriveFile(t *testing.T) {
	tests := []struct {
		name string
		file ingest.DriveFile
		want domain.RecordFormat
	}{
		{
			name: "markdown extension",
			file: ingest.DriveFile{Name: "notes.md", MimeType: "text/plain"},
			want: domain.RecordFormatMD,
		},
		{
			name: "plain text extension",
			file: ingest.DriveFile{Name: "notes.txt", MimeType: "text/plain"},
			want: domain.RecordFormatText,
		},
		{
			name: "pdf by mime",
			file: ingest.DriveFile{Name: "scan", MimeType: "application/pdf"},
			want: domain.RecordFormatPDF,
		},
		{
			name: "docx by mime",
			file: ingest.DriveFile{Name: "doc", MimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
			want: domain.RecordFormatDOCX,
		},
		{
			name: "google docs mime maps to text",
			file: ingest.DriveFile{Name: "doc", MimeType: "application/vnd.google-apps.document"},
			want: domain.RecordFormatText,
		},
		{
			name: "code extension",
			file: ingest.DriveFile{Name: "main.go", MimeType: "text/plain"},
			want: domain.RecordFormatCode,
		},
		{
			name: "json extension maps to code",
			file: ingest.DriveFile{Name: "config.json", MimeType: "text/plain"},
			want: domain.RecordFormatCode,
		},
		{
			name: "html extension remains text",
			file: ingest.DriveFile{Name: "index.html", MimeType: "text/html"},
			want: domain.RecordFormatText,
		},
		{
			name: "image by mime",
			file: ingest.DriveFile{Name: "photo", MimeType: "image/png"},
			want: domain.RecordFormatImage,
		},
		{
			name: "image by extension",
			file: ingest.DriveFile{Name: "photo.jpeg", MimeType: "application/octet-stream"},
			want: domain.RecordFormatImage,
		},
		{
			name: "fallback link",
			file: ingest.DriveFile{Name: "blob.bin", MimeType: "application/octet-stream"},
			want: domain.RecordFormatLink,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := recordFormatFromDriveFile(tc.file)
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestFilterDriveFilesBySelection(t *testing.T) {
	files := []ingest.DriveFile{
		{ID: "a", Name: "a.txt"},
		{ID: "b", Name: "b.txt"},
		{ID: "c", Name: "c.txt"},
	}

	filtered := filterDriveFilesBySelection(files, []string{"b", "c"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 files, got %d", len(filtered))
	}
	if filtered[0].ID != "b" || filtered[1].ID != "c" {
		t.Fatalf("unexpected filtered order/content: %+v", filtered)
	}
}

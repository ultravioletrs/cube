// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"testing"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

func TestDetectRecordFormat(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		mimeType string
		want     domain.RecordFormat
	}{
		{
			name:     "markdown extension",
			fileName: "notes.md",
			mimeType: "application/octet-stream",
			want:     domain.RecordFormatMD,
		},
		{
			name:     "plain text extension",
			fileName: "notes.txt",
			mimeType: "text/plain",
			want:     domain.RecordFormatText,
		},
		{
			name:     "pdf mime",
			fileName: "scan.bin",
			mimeType: "application/pdf",
			want:     domain.RecordFormatPDF,
		},
		{
			name:     "docx mime",
			fileName: "doc.bin",
			mimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			want:     domain.RecordFormatDOCX,
		},
		{
			name:     "code extension",
			fileName: "main.go",
			mimeType: "text/plain",
			want:     domain.RecordFormatCode,
		},
		{
			name:     "html extension remains text",
			fileName: "index.html",
			mimeType: "text/html",
			want:     domain.RecordFormatText,
		},
		{
			name:     "image mime",
			fileName: "img.bin",
			mimeType: "image/png",
			want:     domain.RecordFormatImage,
		},
		{
			name:     "google docs mime maps to text",
			fileName: "doc",
			mimeType: "application/vnd.google-apps.document",
			want:     domain.RecordFormatText,
		},
		{
			name:     "json extension maps to code",
			fileName: "config.json",
			mimeType: "text/plain",
			want:     domain.RecordFormatCode,
		},
		{
			name:     "image by extension",
			fileName: "photo.jpeg",
			mimeType: "application/octet-stream",
			want:     domain.RecordFormatImage,
		},
		{
			name:     "fallback link",
			fileName: "blob.bin",
			mimeType: "application/octet-stream",
			want:     domain.RecordFormatLink,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DetectRecordFormat(tc.fileName, tc.mimeType)
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

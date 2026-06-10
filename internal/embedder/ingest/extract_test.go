// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestNormalizeFileMIMETypeInfersFromName(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		mimeType string
		want     string
	}{
		{
			name:     "blank pdf",
			fileName: "report.pdf",
			want:     "application/pdf",
		},
		{
			name:     "generic binary pdf",
			fileName: "report.pdf",
			mimeType: "application/octet-stream",
			want:     "application/pdf",
		},
		{
			name:     "markdown",
			fileName: "notes.md",
			mimeType: "application/octet-stream",
			want:     "text/markdown",
		},
		{
			name:     "docx",
			fileName: "proposal.docx",
			want:     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		},
		{
			name:     "image",
			fileName: "scan.png",
			want:     "image/png",
		},
		{
			name:     "media type parameters",
			fileName: "notes.txt",
			mimeType: "text/plain; charset=utf-8",
			want:     "text/plain",
		},
		{
			name:     "unknown blank",
			fileName: "archive.bin",
			want:     "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeFileMIMEType(tc.fileName, tc.mimeType)
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestExtractTextInfersImageMIMEFromName(t *testing.T) {
	doc, err := ExtractText(FileMeta{Name: "photo.png"}, []byte("not-a-real-png"))
	if err != nil {
		t.Fatalf("ExtractText returned error: %v", err)
	}
	if doc.ImageMode != ImageIngestModeImage {
		t.Fatalf("expected image mode, got %q", doc.ImageMode)
	}
	if doc.MimeType != "image/png" {
		t.Fatalf("expected normalized mime image/png, got %q", doc.MimeType)
	}
	if !strings.Contains(doc.Text, "mime_type: image/png") {
		t.Fatalf("expected inferred image mime in descriptor, got %q", doc.Text)
	}
}

func TestExtractTextInfersPlainTextFromContent(t *testing.T) {
	doc, err := ExtractText(FileMeta{}, []byte("hello from extensionless s3 object"))
	if err != nil {
		t.Fatalf("ExtractText returned error: %v", err)
	}
	if doc.Text != "hello from extensionless s3 object" {
		t.Fatalf("unexpected text: %q", doc.Text)
	}
	if doc.MimeType != "text/plain" {
		t.Fatalf("expected sniffed text/plain mime, got %q", doc.MimeType)
	}
}

func TestExtractTextInfersMIMEFromExternalID(t *testing.T) {
	doc, err := ExtractText(FileMeta{
		ID:   "team/docs/notes.md",
		Name: "notes",
	}, []byte("# Notes"))
	if err != nil {
		t.Fatalf("ExtractText returned error: %v", err)
	}
	if doc.Text != "# Notes" {
		t.Fatalf("unexpected text: %q", doc.Text)
	}
}

func TestMIMETypeFromContentSniffsSupportedTypes(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    string
	}{
		{
			name:    "pdf",
			content: []byte("%PDF-1.7\n1 0 obj\n"),
			want:    "application/pdf",
		},
		{
			name:    "plain text",
			content: []byte("hello from s3\n"),
			want:    "text/plain",
		},
		{
			name:    "docx",
			content: minimalDOCX(t),
			want:    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := mimeTypeFromContent(tc.content); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func minimalDOCX(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	archive := zip.NewWriter(&buf)
	file, err := archive.Create("word/document.xml")
	if err != nil {
		t.Fatalf("create docx entry: %v", err)
	}
	if _, err := file.Write([]byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"/>`)); err != nil {
		t.Fatalf("write docx entry: %v", err)
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("close docx archive: %v", err)
	}
	return buf.Bytes()
}

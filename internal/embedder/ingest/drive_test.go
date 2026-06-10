// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"strings"
	"testing"
)

func TestExtractTextImageUsesDescriptor(t *testing.T) {
	doc, err := ExtractText(FileMeta{
		Name:     "poster.png",
		MimeType: "image/png",
	}, []byte{0x89, 0x50, 0x4e, 0x47})
	if err != nil {
		t.Fatalf("ExtractText returned error: %v", err)
	}

	if !strings.Contains(doc.Text, "poster.png") {
		t.Fatalf("expected descriptor to include file name, got %q", doc.Text)
	}
	if !strings.Contains(doc.Text, "image/png") {
		t.Fatalf("expected descriptor to include MIME type, got %q", doc.Text)
	}
}

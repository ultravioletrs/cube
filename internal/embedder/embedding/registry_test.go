// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package embedding

import (
	"testing"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

func TestRegistryForRecordByFormat(t *testing.T) {
	reg, err := NewRegistry(Config{
		Profiles: map[string]ProfileConfig{
			"text": {
				Provider:   "ollama",
				BaseURL:    "http://localhost:11434",
				Model:      "text-model",
				Dimensions: 111,
			},
			"code": {
				Provider:   "ollama",
				BaseURL:    "http://localhost:11434",
				Model:      "code-model",
				Dimensions: 222,
			},
			"image": {
				Provider:   "ollama",
				BaseURL:    "http://localhost:11434",
				Model:      "image-model",
				Dimensions: 333,
			},
		},
		Selection: SelectionConfig{
			DefaultProfile: "text",
			ByRecordFormat: map[domain.RecordFormat]string{
				domain.RecordFormatText:  "text",
				domain.RecordFormatMD:    "text",
				domain.RecordFormatPDF:   "text",
				domain.RecordFormatDOCX:  "text",
				domain.RecordFormatLink:  "text",
				domain.RecordFormatCode:  "code",
				domain.RecordFormatImage: "image",
			},
		},
	})
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}

	tests := []struct {
		name     string
		format   domain.RecordFormat
		wantDims int
	}{
		{name: "text format", format: domain.RecordFormatText, wantDims: 111},
		{name: "markdown format", format: domain.RecordFormatMD, wantDims: 111},
		{name: "pdf format", format: domain.RecordFormatPDF, wantDims: 111},
		{name: "docx format", format: domain.RecordFormatDOCX, wantDims: 111},
		{name: "link format", format: domain.RecordFormatLink, wantDims: 111},
		{name: "code format", format: domain.RecordFormatCode, wantDims: 222},
		{name: "image format", format: domain.RecordFormatImage, wantDims: 333},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, err := reg.ForRecord(domain.Record{Format: tc.format})
			if err != nil {
				t.Fatalf("for record: %v", err)
			}
			if got := client.Dimensions(); got != tc.wantDims {
				t.Fatalf("expected dimensions %d, got %d", tc.wantDims, got)
			}
		})
	}
}

// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"testing"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

func TestSourceRecordNeedsRequeue(t *testing.T) {
	base := domain.Record{
		Status:        domain.RecordStatusIndexed,
		Format:        domain.RecordFormatLink,
		MimeType:      "text/html",
		SourceVersion: "etag-1",
	}

	tests := []struct {
		name     string
		existing domain.Record
		next     domain.Record
		want     bool
	}{
		{
			name:     "unchanged indexed record",
			existing: base,
			next:     base,
			want:     false,
		},
		{
			name:     "source version changed",
			existing: base,
			next:     withSourceVersion(base, "etag-2"),
			want:     true,
		},
		{
			name:     "not indexed",
			existing: withStatus(base, domain.RecordStatusFailed),
			next:     base,
			want:     true,
		},
		{
			name:     "format changed after source metadata improved",
			existing: base,
			next:     withFormat(base, domain.RecordFormatImage),
			want:     true,
		},
		{
			name:     "mime changed after source metadata improved",
			existing: base,
			next:     withMIME(base, "image/png"),
			want:     true,
		},
		{
			name:     "mime whitespace ignored",
			existing: withMIME(base, " image/png "),
			next:     withMIME(base, "image/png"),
			want:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sourceRecordNeedsRequeue(tc.existing, tc.next)
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func withSourceVersion(rec domain.Record, sourceVersion string) domain.Record {
	rec.SourceVersion = sourceVersion
	return rec
}

func withStatus(rec domain.Record, status domain.RecordStatus) domain.Record {
	rec.Status = status
	return rec
}

func withFormat(rec domain.Record, format domain.RecordFormat) domain.Record {
	rec.Format = format
	return rec
}

func withMIME(rec domain.Record, mimeType string) domain.Record {
	rec.MimeType = mimeType
	return rec
}

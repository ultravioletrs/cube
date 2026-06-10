// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"testing"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

func TestWorkerShouldProbeRawImage(t *testing.T) {
	worker := &Worker{}
	base := domain.Record{
		Format:   domain.RecordFormatLink,
		Name:     "f55cb722-9c38-4b16-a1dd-66a1b8bd9b20",
		MimeType: "application/octet-stream",
	}

	tests := []struct {
		name string
		rec  domain.Record
		want bool
	}{
		{
			name: "extensionless link with generic mime",
			rec:  base,
			want: true,
		},
		{
			name: "extensionless link with empty mime",
			rec:  withWorkerProbeMIME(base, ""),
			want: true,
		},
		{
			name: "already image",
			rec:  withWorkerProbeFormat(base, domain.RecordFormatImage),
			want: false,
		},
		{
			name: "known extension",
			rec:  withWorkerProbeName(base, "photo.png"),
			want: false,
		},
		{
			name: "known external id extension",
			rec:  withWorkerProbeExternalID(base, "team/docs/photo.png"),
			want: false,
		},
		{
			name: "specific non-image mime",
			rec:  withWorkerProbeMIME(base, "text/html"),
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := worker.shouldProbeRawImage(tc.rec)
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func withWorkerProbeFormat(rec domain.Record, format domain.RecordFormat) domain.Record {
	rec.Format = format
	return rec
}

func withWorkerProbeName(rec domain.Record, name string) domain.Record {
	rec.Name = name
	return rec
}

func withWorkerProbeExternalID(rec domain.Record, externalID string) domain.Record {
	rec.ExternalID = externalID
	return rec
}

func withWorkerProbeMIME(rec domain.Record, mimeType string) domain.Record {
	rec.MimeType = mimeType
	return rec
}

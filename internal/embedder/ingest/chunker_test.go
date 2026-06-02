// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest

import "testing"

func TestAdaptiveChunkSize(t *testing.T) {
	cases := []struct {
		name     string
		words    int
		baseSize int
		want     int
	}{
		{
			name:     "keeps default for small documents",
			words:    10_000,
			baseSize: 512,
			want:     512,
		},
		{
			name:     "uses larger chunks for large documents",
			words:    largeChunkWords,
			baseSize: 512,
			want:     largeChunkSize,
		},
		{
			name:     "uses largest chunks for very large documents",
			words:    veryLargeChunkWords,
			baseSize: 512,
			want:     veryLargeChunkSize,
		},
		{
			name:     "does not reduce explicit larger chunk size",
			words:    veryLargeChunkWords,
			baseSize: 8192,
			want:     8192,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := adaptiveChunkSize(tc.words, tc.baseSize); got != tc.want {
				t.Fatalf("expected chunk size %d, got %d", tc.want, got)
			}
		})
	}
}

func TestAdaptiveOverlap(t *testing.T) {
	if got := adaptiveOverlap(10_000, 64); got != 64 {
		t.Fatalf("expected overlap to remain 64 for small document, got %d", got)
	}
	if got := adaptiveOverlap(largeChunkWords, 64); got != 0 {
		t.Fatalf("expected overlap to be disabled for large document, got %d", got)
	}
}

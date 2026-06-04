// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"testing"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

func TestMatchedRecordsBlockDeduplicatesRecordNames(t *testing.T) {
	got := matchedRecordsBlock([]domain.VectorChunk{
		{RecordName: "odrzavanje_mart_2026.pdf"},
		{RecordName: "odrzavanje_mart_2026.pdf"},
		{RecordName: "DusanEUCNC.pdf"},
		{RecordName: " "},
	})

	want := "- odrzavanje_mart_2026.pdf\n- DusanEUCNC.pdf\n"
	if got != want {
		t.Fatalf("unexpected records block:\nwant %q\ngot  %q", want, got)
	}
}

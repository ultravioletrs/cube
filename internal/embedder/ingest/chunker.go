// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest

import "strings"

type chunkPlan struct {
	Size    int
	Overlap int
	Words   int
}

const (
	defaultChunkSize    = 512
	largeChunkWords     = 250_000
	veryLargeChunkWords = 1_000_000
	largeChunkSize      = 512
	veryLargeChunkSize  = 512
)

// chunk splits text into overlapping word windows.
// chunkSize and overlap are in words, matching the Python worker defaults.
func chunk(text string, chunkSize, overlap int) []string {
	words := strings.Fields(text)
	chunks, _ := chunkWords(words, chunkSize, overlap)
	return chunks
}

func adaptiveChunk(text string, chunkSize, overlap int) ([]string, chunkPlan) {
	words := strings.Fields(text)
	wordCount := len(words)
	return chunkWords(words, adaptiveChunkSize(wordCount, chunkSize), adaptiveOverlap(wordCount, overlap))
}

func adaptiveChunkSize(wordCount, baseSize int) int {
	if baseSize <= 0 {
		baseSize = defaultChunkSize
	}
	switch {
	case wordCount >= veryLargeChunkWords && baseSize < veryLargeChunkSize:
		return veryLargeChunkSize
	case wordCount >= largeChunkWords && baseSize < largeChunkSize:
		return largeChunkSize
	default:
		return baseSize
	}
}

func adaptiveOverlap(wordCount, overlap int) int {
	if wordCount >= largeChunkWords {
		return 0
	}
	return overlap
}

func chunkWords(words []string, chunkSize, overlap int) ([]string, chunkPlan) {
	plan := chunkPlan{Size: chunkSize, Overlap: overlap, Words: len(words)}
	if len(words) == 0 {
		return nil, plan
	}
	plan.Size, plan.Overlap = normalizeChunkOptions(chunkSize, overlap)
	var chunks []string
	step := plan.Size - plan.Overlap
	for start := 0; start < len(words); start += step {
		end := start + plan.Size
		if end > len(words) {
			end = len(words)
		}
		chunks = append(chunks, strings.Join(words[start:end], " "))
		if end == len(words) {
			break
		}
	}
	return chunks, plan
}

func normalizeChunkOptions(chunkSize, overlap int) (int, int) {
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= chunkSize {
		overlap = chunkSize / 2
	}
	return chunkSize, overlap
}

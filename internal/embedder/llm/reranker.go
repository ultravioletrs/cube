// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package llm

import "context"

// Reranker scores (query, document) pairs and returns relevance scores in the
// same order as the supplied documents slice.  Higher score = more relevant.
type Reranker interface {
	Rerank(ctx context.Context, query string, documents []string) ([]float64, error)
}

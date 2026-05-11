package domain

import "context"

// VectorChunk is a chunk returned by vector similarity search.
type VectorChunk struct {
	RecordID    string `json:"record_id"`
	RecordName  string `json:"record_name"`
	ExternalURL string `json:"external_url,omitempty"`
	ChunkIndex  int    `json:"chunk_index"`
	Content     string `json:"content"`
}

// VectorRetrieveService retrieves relevant chunks via embedding similarity.
type VectorRetrieveService interface {
	Retrieve(ctx context.Context, userID, query string, recordIDs []string, topK int) ([]VectorChunk, error)
}

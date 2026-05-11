package transport

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ultravioletrs/cube/internal/embedder/auth"
	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

// MountRetrieve registers the vector retrieve endpoint.
func MountRetrieve(r chi.Router, svc domain.VectorRetrieveService) {
	r.Post("/api/v1/retrieve", retrieveHandler(svc))
}

type retrieveRequest struct {
	Query     string   `json:"query"`
	RecordIDs []string `json:"record_ids,omitempty"`
	TopK      int      `json:"top_k,omitempty"`
}

type retrieveResponse struct {
	Chunks []domain.VectorChunk `json:"chunks"`
}

func retrieveHandler(svc domain.VectorRetrieveService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserID(r.Context())

		var req retrieveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}
		if req.Query == "" {
			writeJSON(w, http.StatusBadRequest, errBody("query is required"))
			return
		}

		chunks, err := svc.Retrieve(r.Context(), userID, req.Query, req.RecordIDs, req.TopK)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("retrieve failed"))
			return
		}

		if chunks == nil {
			chunks = []domain.VectorChunk{}
		}
		writeJSON(w, http.StatusOK, retrieveResponse{Chunks: chunks})
	}
}

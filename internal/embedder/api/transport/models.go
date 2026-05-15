// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// MountModels registers model-listing endpoints.
func MountModels(r chi.Router, ollamaBaseURL string) {
	r.Get("/api/v1/models/ollama", listOllamaModelsHandler(ollamaBaseURL))
}

type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

func listOllamaModelsHandler(baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := http.Get(baseURL + "/api/tags") //nolint:noctx
		if err != nil {
			writeJSON(w, http.StatusBadGateway, errBody("ollama unreachable: "+err.Error()))
			return
		}
		defer resp.Body.Close()

		var tags ollamaTagsResponse
		if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
			writeJSON(w, http.StatusBadGateway, errBody("ollama response invalid: "+err.Error()))
			return
		}

		names := make([]string, 0, len(tags.Models))
		for _, m := range tags.Models {
			// Skip models that don't support chat (embeddings, code-completion).
			name := strings.ToLower(m.Name)
			if strings.Contains(name, "embed") || strings.Contains(name, "starcoder") {
				continue
			}
			names = append(names, m.Name)
		}

		writeJSON(w, http.StatusOK, map[string]any{"models": names})
	}
}

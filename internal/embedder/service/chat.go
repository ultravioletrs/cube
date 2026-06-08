// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/llm"
)

type chatService struct {
	retrieve      domain.VectorRetrieveService
	llm           llm.Client
	reranker      llm.Reranker // nil = disabled
	topK          int
	defaultCfg    llm.Config
	ollamaBaseURL string
	factory       llm.ClientFactory // builds a client from a per-request config
}

// NewChatService returns a ChatService that retrieves context chunks then
// streams the LLM response.  reranker may be nil to skip re-ranking.
// factory is called to build a temporary client when the request overrides
// the server-default model; it may be nil if per-request overrides are not needed.
func NewChatService(retrieve domain.VectorRetrieveService, llmClient llm.Client, reranker llm.Reranker, topK int, defaultCfg llm.Config, ollamaBaseURL string, factory llm.ClientFactory) domain.ChatService {
	if topK <= 0 {
		topK = 15
	}
	return &chatService{retrieve: retrieve, llm: llmClient, reranker: reranker, topK: topK, defaultCfg: defaultCfg, ollamaBaseURL: ollamaBaseURL, factory: factory}
}

func (s *chatService) Chat(ctx context.Context, domainID string, messages []domain.ChatMessage, recordIDs []string, modelCfg *domain.ModelConfig) (<-chan domain.ChatEvent, error) {
	// Build a per-request client if the caller overrides the model.
	llmClient := s.llm
	if modelCfg != nil && modelCfg.Model != "" && s.factory != nil {
		baseURL := modelCfg.BaseURL
		if baseURL == "" {
			if strings.EqualFold(modelCfg.Provider, "ollama") || strings.EqualFold(modelCfg.Provider, "local") {
				baseURL = s.ollamaBaseURL
			} else {
				baseURL = s.defaultCfg.BaseURL
			}
		}
		provider := modelCfg.Provider
		if provider == "" {
			provider = s.defaultCfg.Provider
		}
		llmClient = s.factory(llm.Config{
			Provider:    provider,
			BaseURL:     baseURL,
			Model:       modelCfg.Model,
			APIKey:      modelCfg.APIKey,
			Temperature: modelCfg.Temperature,
			MaxTokens:   modelCfg.MaxTokens,
		})
		slog.Info("chat: using per-request model", "provider", modelCfg.Provider, "model", modelCfg.Model)
	} else {
		slog.Info("chat: using default model", "provider", s.defaultCfg.Provider, "model", s.defaultCfg.Model)
	}
	query := ""
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			query = messages[i].Content
			break
		}
	}

	var citations []domain.Citation
	var chunks []domain.VectorChunk
	var retrievalWarning string

	if query != "" {
		var err error
		chunks, err = s.retrieve.Retrieve(ctx, domainID, query, recordIDs, s.topK)
		if err != nil {
			retrievalWarning = "Retrieval failed: " + err.Error() + " — answering without document context."
			chunks = nil
		} else if len(chunks) == 0 {
			retrievalWarning = "No relevant chunks found in indexed records — answering without document context."
		}
	}

	// Re-rank if we have a reranker and more than one chunk to order.
	if s.reranker != nil && len(chunks) > 1 {
		docs := make([]string, len(chunks))
		for i, c := range chunks {
			docs[i] = c.Content
		}
		scores, err := s.reranker.Rerank(ctx, query, docs)
		if err == nil && len(scores) == len(chunks) {
			// Sort descending by reranker score.
			type scored struct {
				chunk domain.VectorChunk
				score float64
			}
			pairs := make([]scored, len(chunks))
			for i, c := range chunks {
				pairs[i] = scored{chunk: c, score: scores[i]}
			}
			sort.Slice(pairs, func(i, j int) bool { return pairs[i].score > pairs[j].score })
			for i, p := range pairs {
				chunks[i] = p.chunk
			}
		}
		// On reranker error, keep the RRF order — non-fatal.
	}

	// Build citation list.
	var contextBlock strings.Builder
	for i, c := range chunks {
		contextBlock.WriteString(fmt.Sprintf("[%d] %s:\n%s\n\n", i+1, c.RecordName, c.Content))
		citations = append(citations, domain.Citation{
			RecordID:    c.RecordID,
			RecordName:  c.RecordName,
			ExternalURL: c.ExternalURL,
			ChunkIndex:  c.ChunkIndex,
			Excerpt:     truncate(c.Content, 200),
		})
	}

	// Assemble messages for the LLM.
	// When context chunks are available, inject them directly into the last user
	// message and drop stale history so small models (e.g. llama3.2:3b) are not
	// confused by prior wrong answers or an oversized context window.
	llmMessages := make([]llm.Message, 0, len(messages)+1)
	llmMessages = append(llmMessages, llm.Message{
		Role:    "system",
		Content: "You are a helpful, knowledgeable assistant. When document excerpts are provided, use them as your primary source and cite them, but also use your general knowledge to give complete, thorough answers.",
	})
	if contextBlock.Len() > 0 {
		// RAG mode: only send the last user message augmented with context.
		// Older turns are dropped to prevent accumulated incorrect answers from
		// biasing the model.
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				llmMessages = append(llmMessages, llm.Message{
					Role: "user",
					Content: "Use the following document excerpts to inform your answer. " +
						"Cite relevant excerpts, then elaborate with a complete and thorough explanation.\n\n" +
						"EXCERPTS:\n" + contextBlock.String() +
						"QUESTION: " + messages[i].Content,
				})
				break
			}
		}
	} else {
		for _, m := range messages {
			llmMessages = append(llmMessages, llm.Message{Role: m.Role, Content: m.Content})
		}
	}

	out := make(chan domain.ChatEvent, 64)

	go func() {
		defer close(out)

		if retrievalWarning != "" {
			select {
			case out <- domain.ChatEvent{Type: domain.ChatEventWarning, Error: retrievalWarning}:
			case <-ctx.Done():
				return
			}
		}

		if len(citations) > 0 {
			select {
			case out <- domain.ChatEvent{Type: domain.ChatEventCitations, Citations: citations}:
			case <-ctx.Done():
				return
			}
		}

		tokenCh := make(chan string, 64)
		errCh := make(chan error, 1)

		go func() {
			errCh <- llmClient.StreamChat(ctx, llmMessages, tokenCh)
		}()

		for {
			select {
			case tok, ok := <-tokenCh:
				if !ok {
					if err := <-errCh; err != nil && ctx.Err() == nil {
						select {
						case out <- domain.ChatEvent{Type: domain.ChatEventError, Error: err.Error()}:
						case <-ctx.Done():
						}
					} else {
						select {
						case out <- domain.ChatEvent{Type: domain.ChatEventDone}:
						case <-ctx.Done():
						}
					}
					return
				}
				select {
				case out <- domain.ChatEvent{Type: domain.ChatEventToken, Content: tok}:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

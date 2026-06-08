// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"unicode"

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

const ragSystemPrompt = `You are a retrieval-grounded assistant.
When document excerpts are provided, answer from those excerpts first and avoid inventing information.
When the user's intent is unclear, consider both the retrieved document context and the possible general meaning of the query. Clearly distinguish what the records show from general knowledge, and ask a concise follow-up question when needed.
For short keywords, filename fragments, dates, or codes, mention relevant record names naturally when they help answer the question.
Use direct phrasing such as "I found this in <record name>" or "The relevant record is <record name>". Do not say "the matched record name for the query".
If the excerpts do not contain enough information to answer the user's intent, say what was found and ask a concise follow-up question.
Cite relevant excerpt numbers when making factual claims.`

const conversationSystemPrompt = `You are a helpful assistant.
Respond naturally to conversational messages.
Do not claim to have searched, found, or used records when no document excerpts are provided.`

const noRelevantContextResponse = "I couldn't find relevant information in the indexed records for that question. Please narrow the question or select a specific record or source."

var conversationalMessages = map[string]struct{}{
	"good afternoon":  {},
	"good evening":    {},
	"good morning":    {},
	"goodbye":         {},
	"got it":          {},
	"hello":           {},
	"hi":              {},
	"hi there":        {},
	"how are you":     {},
	"hey":             {},
	"ok":              {},
	"okay":            {},
	"see you":         {},
	"thank you":       {},
	"thanks":          {},
	"thanks a lot":    {},
	"understood":      {},
	"you are welcome": {},
	"you're welcome":  {},
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

func (s *chatService) Chat(ctx context.Context, domainID string, messages []domain.ChatMessage, recordIDs []string, modelCfg *domain.ModelConfig, debug bool) (<-chan domain.ChatEvent, error) {
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
	noRelevantContext := false
	weakContext := false
	retrieveForQuery := shouldRetrieve(query)
	skippedReason := ""

	if retrieveForQuery {
		var err error
		chunks, err = s.retrieve.Retrieve(ctx, domainID, query, recordIDs, s.topK)
		if err != nil {
			retrievalWarning = "Retrieval failed: " + err.Error() + " — answering without document context."
			chunks = nil
		} else if len(chunks) == 0 {
			retrievalWarning = "No relevant chunks found in indexed records."
			noRelevantContext = true
		} else if len(recordIDs) > 0 {
			// User explicitly scoped retrieval to specific records; trust those
			// chunks rather than second-guessing them with lexical grounding.
		} else if terms := meaningfulQueryTerms(query); len(terms) > 0 {
			// Only filter when the query carries a lexical signal to filter on.
			// A query of pure stopwords/short tokens yields no terms; in that
			// case keep the retrieved chunks instead of discarding everything.
			if grounded := groundedChunks(terms, chunks); len(grounded) == 0 {
				retrievalWarning = "Retrieved chunks did not appear relevant enough to answer from indexed records."
				weakContext = true
			} else {
				chunks = grounded
			}
		}
	}
	if !retrieveForQuery {
		skippedReason = "conversational message"
		slog.Info("chat: skipped retrieval for conversational message")
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

	slog.Info("chat: prepared rag context",
		"retrieval_query_chars", len(query),
		"retrieved_chunks", len(chunks),
		"prompt_chunks", len(chunks),
	)

	// Build citation list.
	var contextBlock strings.Builder
	for i, c := range chunks {
		contextBlock.WriteString(fmt.Sprintf("[%d] Record: %s\nExcerpt:\n%s\n\n", i+1, c.RecordName, c.Content))
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
	systemPrompt := ragSystemPrompt
	if !retrieveForQuery {
		systemPrompt = conversationSystemPrompt
	}
	llmMessages = append(llmMessages, llm.Message{
		Role:    "system",
		Content: systemPrompt,
	})
	if contextBlock.Len() > 0 {
		// RAG mode: only send the last user message augmented with context.
		// Older turns are dropped to prevent accumulated incorrect answers from
		// biasing the model.
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				llmMessages = append(llmMessages, llm.Message{
					Role: "user",
					Content: "Use the following retrieved records and excerpts to answer the question. " +
						"Mention relevant record names naturally when they help answer the question. " +
						"Prefer direct phrasing like \"I found this in <record name>\". " +
						"Clearly distinguish information found in the records from general knowledge.\n\n" +
						"MATCHED RECORDS:\n" + matchedRecordsBlock(chunks) + "\n" +
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

		if debug {
			debugData := buildChatDebug(query, s.topK, retrieveForQuery, skippedReason, recordIDs, chunks)
			select {
			case out <- domain.ChatEvent{Type: domain.ChatEventDebug, Debug: &debugData}:
			case <-ctx.Done():
				return
			}
		}

		if noRelevantContext || weakContext {
			select {
			case out <- domain.ChatEvent{Type: domain.ChatEventToken, Content: noRelevantContextResponse}:
			case <-ctx.Done():
				return
			}
			select {
			case out <- domain.ChatEvent{Type: domain.ChatEventDone}:
			case <-ctx.Done():
			}
			return
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

func shouldRetrieve(query string) bool {
	normalized := strings.ToLower(strings.TrimSpace(query))
	normalized = strings.TrimFunc(normalized, func(r rune) bool {
		return unicode.IsPunct(r) || unicode.IsSpace(r)
	})
	normalized = strings.Join(strings.Fields(normalized), " ")
	if normalized == "" {
		return false
	}
	_, conversational := conversationalMessages[normalized]
	return !conversational
}

func matchedRecordsBlock(chunks []domain.VectorChunk) string {
	if len(chunks) == 0 {
		return ""
	}
	seen := make(map[string]struct{}, len(chunks))
	var b strings.Builder
	for _, c := range chunks {
		name := strings.TrimSpace(c.RecordName)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		b.WriteString("- ")
		b.WriteString(name)
		b.WriteByte('\n')
	}
	return b.String()
}

func groundedChunks(terms []string, chunks []domain.VectorChunk) []domain.VectorChunk {
	if len(terms) == 0 {
		return nil
	}
	grounded := make([]domain.VectorChunk, 0, len(chunks))
	for _, c := range chunks {
		haystack := strings.ToLower(c.RecordName + " " + c.Content)
		for _, term := range terms {
			if strings.Contains(haystack, term) {
				grounded = append(grounded, c)
				break
			}
		}
	}
	return grounded
}

func meaningfulQueryTerms(query string) []string {
	stopwords := map[string]struct{}{
		"about": {}, "after": {}, "again": {}, "also": {}, "and": {}, "any": {}, "are": {}, "can": {}, "could": {}, "details": {}, "did": {}, "document": {}, "documents": {}, "does": {}, "file": {}, "files": {}, "for": {}, "from": {}, "give": {}, "has": {}, "have": {}, "how": {}, "into": {}, "its": {}, "more": {}, "please": {}, "record": {}, "records": {}, "show": {}, "some": {}, "tell": {}, "that": {}, "the": {}, "their": {}, "them": {}, "there": {}, "this": {}, "was": {}, "what": {}, "when": {}, "where": {}, "which": {}, "with": {}, "would": {}, "you": {},
	}
	seen := make(map[string]struct{})
	terms := make([]string, 0, 8)
	for _, field := range strings.Fields(strings.ToLower(query)) {
		term := strings.TrimFunc(field, func(r rune) bool {
			return unicode.IsPunct(r) || unicode.IsSpace(r)
		})
		if len(term) < 3 {
			continue
		}
		if _, skip := stopwords[term]; skip {
			continue
		}
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
		if len(terms) >= 8 {
			break
		}
	}
	return terms
}

func buildChatDebug(query string, topK int, retrievalEnabled bool, skippedReason string, recordIDs []string, chunks []domain.VectorChunk) domain.ChatDebug {
	if topK <= 0 {
		topK = 5
	}
	debug := domain.ChatDebug{
		Query:            query,
		TopK:             topK,
		RetrievalEnabled: retrievalEnabled,
		SkippedReason:    skippedReason,
		RecordIDs:        append([]string(nil), recordIDs...),
		PromptChunks:     make([]domain.ChatDebugChunk, 0, len(chunks)),
	}
	for i, c := range chunks {
		debug.PromptChunks = append(debug.PromptChunks, domain.ChatDebugChunk{
			Rank:        i + 1,
			RecordID:    c.RecordID,
			RecordName:  c.RecordName,
			ExternalURL: c.ExternalURL,
			ChunkIndex:  c.ChunkIndex,
			Score:       c.Score,
			Preview:     truncate(c.Content, 280),
		})
	}
	return debug
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

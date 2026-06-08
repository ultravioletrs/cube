// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"strings"
	"testing"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/llm"
)

type chatRetrieveStub struct {
	calls  int
	chunks []domain.VectorChunk
}

func (s *chatRetrieveStub) Retrieve(context.Context, string, string, []string, int) ([]domain.VectorChunk, error) {
	s.calls++
	return s.chunks, nil
}

type chatLLMStub struct {
	calls    int
	messages []llm.Message
}

func (s *chatLLMStub) StreamChat(_ context.Context, messages []llm.Message, out chan<- string) error {
	s.calls++
	s.messages = messages
	close(out)
	return nil
}

func TestShouldRetrieveOnlySkipsClearConversationalMessages(t *testing.T) {
	tests := []struct {
		query string
		want  bool
	}{
		{query: "hello", want: false},
		{query: "  THANK YOU! ", want: false},
		{query: "how are you?", want: false},
		{query: "", want: false},
		{query: "example retrieval query", want: true},
		{query: "record details", want: true},
		{query: "hello record details", want: true},
		{query: "thanks from record details", want: true},
	}

	for _, test := range tests {
		t.Run(test.query, func(t *testing.T) {
			if got := shouldRetrieve(test.query); got != test.want {
				t.Fatalf("shouldRetrieve(%q) = %v, want %v", test.query, got, test.want)
			}
		})
	}
}

func TestChatSkipsRetrievalForConversationalMessage(t *testing.T) {
	retrieve := &chatRetrieveStub{}
	client := &chatLLMStub{}
	service := NewChatService(retrieve, client, nil, 8, llm.Config{}, "", nil)

	events, err := service.Chat(context.Background(), "domain", []domain.ChatMessage{
		{Role: "user", Content: "hello!"},
	}, nil, nil, false)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
	for range events {
	}

	if retrieve.calls != 0 {
		t.Fatalf("expected retrieval to be skipped, got %d calls", retrieve.calls)
	}
	if len(client.messages) == 0 || client.messages[0].Content != conversationSystemPrompt {
		t.Fatal("expected conversational system prompt")
	}
}

func TestChatDoesNotCallLLMWhenRetrievalFindsNoChunks(t *testing.T) {
	retrieve := &chatRetrieveStub{}
	client := &chatLLMStub{}
	service := NewChatService(retrieve, client, nil, 8, llm.Config{}, "", nil)

	events, err := service.Chat(context.Background(), "domain", []domain.ChatMessage{
		{Role: "user", Content: "record details"},
	}, nil, nil, true)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	var gotToken string
	var gotDone bool
	var gotDebug bool
	for ev := range events {
		switch ev.Type {
		case domain.ChatEventToken:
			gotToken += ev.Content
		case domain.ChatEventDone:
			gotDone = true
		case domain.ChatEventDebug:
			gotDebug = true
		}
	}

	if retrieve.calls != 1 {
		t.Fatalf("expected retrieval to run once, got %d calls", retrieve.calls)
	}
	if client.calls != 0 {
		t.Fatalf("expected LLM call to be skipped, got %d calls", client.calls)
	}
	if gotToken != noRelevantContextResponse {
		t.Fatalf("unexpected fallback response: %q", gotToken)
	}
	if !gotDone {
		t.Fatal("expected done event")
	}
	if !gotDebug {
		t.Fatal("expected debug event")
	}
}

func TestChatDoesNotCallLLMWhenRetrievedChunksAreWeak(t *testing.T) {
	retrieve := &chatRetrieveStub{chunks: []domain.VectorChunk{{
		RecordID:   "record-1",
		RecordName: "record_a.pdf",
		ChunkIndex: 1,
		Content:    "Unrelated retrieved content",
	}}}
	client := &chatLLMStub{}
	service := NewChatService(retrieve, client, nil, 8, llm.Config{}, "", nil)

	events, err := service.Chat(context.Background(), "domain", []domain.ChatMessage{
		{Role: "user", Content: "specific external topic"},
	}, nil, nil, true)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	var gotToken string
	var gotWarning bool
	var gotDebug bool
	for ev := range events {
		switch ev.Type {
		case domain.ChatEventToken:
			gotToken += ev.Content
		case domain.ChatEventWarning:
			gotWarning = true
		case domain.ChatEventDebug:
			gotDebug = true
		}
	}

	if client.calls != 0 {
		t.Fatalf("expected LLM call to be skipped, got %d calls", client.calls)
	}
	if gotToken != noRelevantContextResponse {
		t.Fatalf("unexpected fallback response: %q", gotToken)
	}
	if !gotWarning {
		t.Fatal("expected warning event")
	}
	if !gotDebug {
		t.Fatal("expected debug event")
	}
}

func TestChatFiltersWeakChunksBeforeCallingLLM(t *testing.T) {
	retrieve := &chatRetrieveStub{chunks: []domain.VectorChunk{
		{
			RecordID:   "record-1",
			RecordName: "record_a.pdf",
			ChunkIndex: 1,
			Content:    "Relevant alpha details",
		},
		{
			RecordID:   "record-2",
			RecordName: "record_b.pdf",
			ChunkIndex: 1,
			Content:    "Unrelated retrieved content",
		},
	}}
	client := &chatLLMStub{}
	service := NewChatService(retrieve, client, nil, 8, llm.Config{}, "", nil)

	events, err := service.Chat(context.Background(), "domain", []domain.ChatMessage{
		{Role: "user", Content: "alpha topic"},
	}, nil, nil, true)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
	for range events {
	}

	if client.calls != 1 {
		t.Fatalf("expected one LLM call, got %d", client.calls)
	}
	if len(client.messages) < 2 {
		t.Fatalf("expected system and user messages, got %d", len(client.messages))
	}
	prompt := client.messages[1].Content
	if !strings.Contains(prompt, "record_a.pdf") {
		t.Fatalf("expected grounded chunk in prompt: %q", prompt)
	}
	if strings.Contains(prompt, "record_b.pdf") || strings.Contains(prompt, "Unrelated retrieved content") {
		t.Fatalf("expected weak chunk to be filtered from prompt: %q", prompt)
	}
}

func TestChatEmitsRetrievalDebug(t *testing.T) {
	score := 0.42
	retrieve := &chatRetrieveStub{chunks: []domain.VectorChunk{{
		RecordID:   "record-1",
		RecordName: "record_a.pdf",
		ChunkIndex: 2,
		Content:    "Example alpha retrieved chunk",
		Score:      &score,
	}}}
	client := &chatLLMStub{}
	service := NewChatService(retrieve, client, nil, 5, llm.Config{}, "", nil)

	events, err := service.Chat(context.Background(), "domain", []domain.ChatMessage{
		{Role: "user", Content: "alpha details"},
	}, []string{"record-1"}, nil, true)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	var debug *domain.ChatDebug
	for ev := range events {
		if ev.Type == domain.ChatEventDebug {
			debug = ev.Debug
		}
	}

	if debug == nil {
		t.Fatal("expected retrieval debug event")
	}
	if debug.Query != "alpha details" || !debug.RetrievalEnabled || debug.TopK != 5 {
		t.Fatalf("unexpected debug metadata: %+v", *debug)
	}
	if len(debug.RecordIDs) != 1 || debug.RecordIDs[0] != "record-1" {
		t.Fatalf("unexpected debug record scope: %+v", debug.RecordIDs)
	}
	if len(debug.PromptChunks) != 1 {
		t.Fatalf("expected one debug chunk, got %d", len(debug.PromptChunks))
	}
	chunk := debug.PromptChunks[0]
	if chunk.Rank != 1 || chunk.RecordName != "record_a.pdf" || chunk.ChunkIndex != 2 {
		t.Fatalf("unexpected debug chunk: %+v", chunk)
	}
	if chunk.Score == nil || *chunk.Score != score {
		t.Fatalf("expected score %v, got %v", score, chunk.Score)
	}
}

func TestMatchedRecordsBlockDeduplicatesRecordNames(t *testing.T) {
	got := matchedRecordsBlock([]domain.VectorChunk{
		{RecordName: "record_a.pdf"},
		{RecordName: "record_a.pdf"},
		{RecordName: "record_b.pdf"},
		{RecordName: " "},
	})

	want := "- record_a.pdf\n- record_b.pdf\n"
	if got != want {
		t.Fatalf("unexpected records block:\nwant %q\ngot  %q", want, got)
	}
}

func TestGroundedChunksMatchesRecordNameOrContent(t *testing.T) {
	chunks := []domain.VectorChunk{{
		RecordName: "record_a.pdf",
		Content:    "Example alpha chunk",
	}}

	if len(groundedChunks(meaningfulQueryTerms("alpha request"), chunks)) == 0 {
		t.Fatal("expected content to ground query")
	}
	if len(groundedChunks(meaningfulQueryTerms("alpha details"), chunks)) == 0 {
		t.Fatal("expected content to ground query")
	}
	if len(groundedChunks(meaningfulQueryTerms("external topic"), chunks)) != 0 {
		t.Fatal("expected unrelated query to be weak")
	}
}

func TestChatKeepsChunksWhenQueryHasNoMeaningfulTerms(t *testing.T) {
	retrieve := &chatRetrieveStub{chunks: []domain.VectorChunk{{
		RecordID:   "record-1",
		RecordName: "record_a.pdf",
		ChunkIndex: 1,
		Content:    "Unrelated retrieved content",
	}}}
	client := &chatLLMStub{}
	service := NewChatService(retrieve, client, nil, 8, llm.Config{}, "", nil)

	// Query is entirely stopwords/short tokens: no lexical signal to filter on.
	events, err := service.Chat(context.Background(), "domain", []domain.ChatMessage{
		{Role: "user", Content: "what is this about"},
	}, nil, nil, true)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
	for range events {
	}

	if client.calls != 1 {
		t.Fatalf("expected LLM to be called, got %d calls", client.calls)
	}
}

func TestChatSkipsGroundingForExplicitlyScopedRecords(t *testing.T) {
	retrieve := &chatRetrieveStub{chunks: []domain.VectorChunk{{
		RecordID:   "record-1",
		RecordName: "record_a.pdf",
		ChunkIndex: 1,
		Content:    "Unrelated retrieved content",
	}}}
	client := &chatLLMStub{}
	service := NewChatService(retrieve, client, nil, 8, llm.Config{}, "", nil)

	// User scoped to record-1; grounding must not block the answer even when
	// the query terms do not literally appear in the chunk.
	events, err := service.Chat(context.Background(), "domain", []domain.ChatMessage{
		{Role: "user", Content: "specific external topic"},
	}, []string{"record-1"}, nil, true)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
	for range events {
	}

	if client.calls != 1 {
		t.Fatalf("expected LLM to be called for scoped records, got %d calls", client.calls)
	}
}

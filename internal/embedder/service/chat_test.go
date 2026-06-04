// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"testing"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"github.com/ultravioletrs/cube/internal/embedder/llm"
)

type chatRetrieveStub struct {
	calls int
}

func (s *chatRetrieveStub) Retrieve(context.Context, string, string, []string, int) ([]domain.VectorChunk, error) {
	s.calls++
	return nil, nil
}

type chatLLMStub struct {
	messages []llm.Message
}

func (s *chatLLMStub) StreamChat(_ context.Context, messages []llm.Message, out chan<- string) error {
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
		{query: "mart", want: true},
		{query: "Dusan", want: true},
		{query: "hello Dusan", want: true},
		{query: "thanks from Dusan", want: true},
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
	service := NewChatService(retrieve, client, nil, 8, llm.Config{}, nil)

	events, err := service.Chat(context.Background(), "domain", []domain.ChatMessage{
		{Role: "user", Content: "hello!"},
	}, nil, nil)
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

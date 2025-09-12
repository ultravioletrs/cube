// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"time"

	"github.com/ultraviolet/cube/guardrails"
)

type messageResponse struct {
	Message string `json:"message"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Chat completion responses
type chatCompletionResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []chatChoice `json:"choices"`
	Usage   usage        `json:"usage"`
}

type chatChoice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NeMo configuration responses
type nemoConfigResponse struct {
	Config []byte `json:"config"`
}

// Flow management responses
type createFlowResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type getFlowResponse struct {
	Flow guardrails.Flow `json:"flow"`
}

type getFlowsResponse struct {
	Flows []guardrails.Flow `json:"flows"`
}

type updateFlowResponse struct {
	Flow    guardrails.Flow `json:"flow"`
	Message string          `json:"message"`
}

type deleteFlowResponse struct {
	Message string `json:"message"`
}

// Knowledge Base file management responses
type createKBFileResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type getKBFileResponse struct {
	File guardrails.KBFile `json:"file"`
}

type getKBFilesResponse struct {
	Files []guardrails.KBFile `json:"files"`
}

type listKBFilesResponse struct {
	PageMetadata

	Files []guardrails.KBFile `json:"files"`
}

type updateKBFileResponse struct {
	File    guardrails.KBFile `json:"file"`
	Message string            `json:"message"`
}

type deleteKBFileResponse struct {
	Message string `json:"message"`
}

type searchKBFilesResponse struct {
	Files []guardrails.KBFile `json:"files"`
}

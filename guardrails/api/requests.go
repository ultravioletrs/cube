// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"

	"github.com/absmach/supermq/pkg/errors"
	"github.com/ultraviolet/cube/guardrails"
)

type importConfigRequest struct {
	Data []byte
}

func (r importConfigRequest) validate() error {
	if len(r.Data) == 0 {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("configuration data cannot be empty"))
	}
	return nil
}

// Chat completion requests
type chatCompletionRequest struct {
	Model       string        `json:"model,omitempty"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	UserID      string        `json:"-"` // Set from headers
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (r chatCompletionRequest) validate() error {
	if len(r.Messages) == 0 {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("messages cannot be empty"))
	}

	for i, msg := range r.Messages {
		if msg.Role == "" {
			return errors.Wrap(errors.ErrMalformedEntity, errors.New("message role cannot be empty"))
		}
		if msg.Content == "" {
			return errors.Wrap(errors.ErrMalformedEntity, errors.New("message content cannot be empty"))
		}

		validRoles := []string{"system", "user", "assistant", "function"}
		roleValid := false
		for _, validRole := range validRoles {
			if msg.Role == validRole {
				roleValid = true
				break
			}
		}
		if !roleValid {
			return errors.Wrap(errors.ErrMalformedEntity,
				errors.New(fmt.Sprintf("invalid role '%s' at message %d", msg.Role, i)))
		}
	}

	if r.Temperature < 0 || r.Temperature > 2 {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("temperature must be between 0 and 2"))
	}

	if r.MaxTokens < 0 {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("max_tokens cannot be negative"))
	}

	return nil
}

// Flow management requests

type createFlowRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Type        string `json:"type"`
	Active      bool   `json:"active"`
}

func (r createFlowRequest) validate() error {
	if r.Name == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("flow name cannot be empty"))
	}
	if r.Content == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("flow content cannot be empty"))
	}
	if r.Type == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("flow type cannot be empty"))
	}
	validTypes := []string{"input", "output", "dialog", "retrieval", "execution"}
	typeValid := false
	for _, validType := range validTypes {
		if r.Type == validType {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return errors.Wrap(errors.ErrMalformedEntity,
			errors.New(fmt.Sprintf("invalid flow type '%s'", r.Type)))
	}
	return nil
}

type getFlowRequest struct {
	ID string
}

func (r getFlowRequest) validate() error {
	if r.ID == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("flow ID cannot be empty"))
	}
	return nil
}

type updateFlowRequest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Type        string `json:"type"`
	Active      bool   `json:"active"`
}

func (r updateFlowRequest) validate() error {
	if r.ID == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("flow ID cannot be empty"))
	}

	if r.Name == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("flow name cannot be empty"))
	}

	if r.Content == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("flow content cannot be empty"))
	}

	if r.Type == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("flow type cannot be empty"))
	}
	validTypes := []string{"input", "output", "dialog", "retrieval", "execution"}
	typeValid := false

	for _, validType := range validTypes {
		if r.Type == validType {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return errors.Wrap(errors.ErrMalformedEntity,
			errors.New(fmt.Sprintf("invalid flow type '%s'", r.Type)))
	}
	return nil
}

type deleteFlowRequest struct {
	ID string
}

func (r deleteFlowRequest) validate() error {
	if r.ID == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("flow ID cannot be empty"))
	}
	return nil
}

// Knowledge Base file management requests

type createKBFileRequest struct {
	Name     string                 `json:"name"`
	Content  string                 `json:"content"`
	Type     string                 `json:"type"`
	Category string                 `json:"category"`
	Tags     []string               `json:"tags"`
	Metadata map[string]interface{} `json:"metadata"`
	Active   bool                   `json:"active"`
}

func (r createKBFileRequest) validate() error {
	if r.Name == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("KB file name cannot be empty"))
	}
	if r.Content == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("KB file content cannot be empty"))
	}
	if r.Type == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("KB file type cannot be empty"))
	}
	if r.Category == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("KB file category cannot be empty"))
	}
	validTypes := []string{"markdown", "text", "json", "yaml"}
	typeValid := false
	for _, validType := range validTypes {
		if r.Type == validType {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return errors.Wrap(errors.ErrMalformedEntity,
			errors.New(fmt.Sprintf("invalid KB file type '%s'", r.Type)))
	}
	return nil
}

type getKBFileRequest struct {
	ID string
}

func (r getKBFileRequest) validate() error {
	if r.ID == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("KB file ID cannot be empty"))
	}
	return nil
}

type listKBFilesRequest struct {
	PageMetadata

	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

func (r listKBFilesRequest) validate() error {
	if r.Limit < 0 {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("limit cannot be negative"))
	}
	if r.Offset < 0 {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("offset cannot be negative"))
	}
	if r.Limit == 0 {
		r.Limit = 10
	}
	if r.Limit > 100 {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("limit cannot exceed 100"))
	}
	return nil
}

type updateKBFileRequest struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Content  string                 `json:"content"`
	Type     string                 `json:"type"`
	Category string                 `json:"category"`
	Tags     []string               `json:"tags"`
	Metadata map[string]interface{} `json:"metadata"`
	Active   bool                   `json:"active"`
}

func (r updateKBFileRequest) validate() error {
	if r.ID == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("KB file ID cannot be empty"))
	}
	if r.Name == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("KB file name cannot be empty"))
	}
	if r.Content == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("KB file content cannot be empty"))
	}
	if r.Type == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("KB file type cannot be empty"))
	}
	if r.Category == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("KB file category cannot be empty"))
	}
	validTypes := []string{"markdown", "text", "json", "yaml"}
	typeValid := false
	for _, validType := range validTypes {
		if r.Type == validType {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return errors.Wrap(errors.ErrMalformedEntity,
			errors.New(fmt.Sprintf("invalid KB file type '%s'", r.Type)))
	}
	return nil
}

type deleteKBFileRequest struct {
	ID string
}

func (r deleteKBFileRequest) validate() error {
	if r.ID == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("KB file ID cannot be empty"))
	}
	return nil
}

type searchKBFilesRequest struct {
	Query      string   `json:"query"`
	Categories []string `json:"categories"`
	Tags       []string `json:"tags"`
	Limit      uint64   `json:"limit"`
}

func (r searchKBFilesRequest) validate() error {
	if r.Query == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("search query cannot be empty"))
	}

	if r.Limit == 0 {
		r.Limit = 10
	}

	if r.Limit > 100 {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("limit cannot exceed 100"))
	}
	return nil
}

// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/absmach/supermq/pkg/errors"
	"github.com/ultraviolet/cube/guardrails"
)

type updateTopicsRequest struct {
	Topics []string `json:"topics"`
}

func (r updateTopicsRequest) validate() error {
	if r.Topics == nil {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("topics list cannot be nil"))
	}
	// Validate each topic
	for _, topic := range r.Topics {
		if topic == "" {
			return errors.Wrap(errors.ErrMalformedEntity, errors.New("topic cannot be empty"))
		}
	}
	return nil
}

type addTopicRequest struct {
	Topic string `json:"topic"`
}

func (r addTopicRequest) validate() error {
	if r.Topic == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("topic cannot be empty"))
	}
	return nil
}

type removeTopicRequest struct {
	Topic string
}

func (r removeTopicRequest) validate() error {
	if r.Topic == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("topic cannot be empty"))
	}
	return nil
}

type updateBiasPatternsRequest struct {
	Patterns map[string][]guardrails.BiasPattern `json:"patterns"`
}

func (r updateBiasPatternsRequest) validate() error {
	if r.Patterns == nil {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("patterns cannot be nil"))
	}
	for category, patterns := range r.Patterns {
		if category == "" {
			return errors.Wrap(errors.ErrMalformedEntity, errors.New("category name cannot be empty"))
		}
		for _, pattern := range patterns {
			if pattern.Pattern == "" {
				return errors.Wrap(errors.ErrMalformedEntity, errors.New("pattern cannot be empty"))
			}
		}
	}
	return nil
}

type updateFactualityConfigRequest struct {
	Config guardrails.FactualityConfig `json:"config"`
}

func (r updateFactualityConfigRequest) validate() error {
	if r.Config.ConfidenceThreshold < 0 || r.Config.ConfidenceThreshold > 1 {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("confidence threshold must be between 0 and 1"))
	}
	return nil
}

type getAuditLogRequest struct {
	Limit int
}

func (r getAuditLogRequest) validate() error {
	if r.Limit < 0 {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("limit cannot be negative"))
	}
	if r.Limit == 0 {
		r.Limit = 100 // Default limit
	}
	if r.Limit > 1000 {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("limit cannot exceed 1000"))
	}
	return nil
}

type importConfigRequest struct {
	Data []byte
}

func (r importConfigRequest) validate() error {
	if len(r.Data) == 0 {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("configuration data cannot be empty"))
	}
	return nil
}

// Policy management requests

type createPolicyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Rules       string `json:"rules"`
}

func (r createPolicyRequest) validate() error {
	if r.Name == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("policy name cannot be empty"))
	}
	if r.Rules == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("policy rules cannot be empty"))
	}
	return nil
}

type updatePolicyRequest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Rules       string `json:"rules"`
}

func (r updatePolicyRequest) validate() error {
	if r.ID == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("policy ID cannot be empty"))
	}
	if r.Name == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("policy name cannot be empty"))
	}
	if r.Rules == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("policy rules cannot be empty"))
	}
	return nil
}

type getPolicyRequest struct {
	ID string
}

func (r getPolicyRequest) validate() error {
	if r.ID == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("policy ID cannot be empty"))
	}
	return nil
}

type listPoliciesRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func (r listPoliciesRequest) validate() error {
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

type deletePolicyRequest struct {
	ID string
}

func (r deletePolicyRequest) validate() error {
	if r.ID == "" {
		return errors.Wrap(errors.ErrMalformedEntity, errors.New("policy ID cannot be empty"))
	}
	return nil
}

// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import "github.com/ultraviolet/cube/guardrails"

type getRestrictedTopicsResponse struct {
	Topics []string `json:"topics"`
}

type getBiasPatternsResponse struct {
	Patterns map[string][]guardrails.BiasPattern `json:"patterns"`
}

type getFactualityConfigResponse struct {
	Config guardrails.FactualityConfig `json:"config"`
}

type getAuditLogResponse struct {
	Entries []guardrails.AuditLog `json:"entries"`
	Total   int                   `json:"total"`
}

type messageResponse struct {
	Message string `json:"message"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type createPolicyResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type getPolicyResponse struct {
	Policy guardrails.Policy `json:"policy"`
}

type listPoliciesResponse struct {
	Policies []guardrails.Policy `json:"policies"`
	Total    int                 `json:"total"`
	Limit    int                 `json:"limit"`
	Offset   int                 `json:"offset"`
}

type updatePolicyResponse struct {
	Policy  guardrails.Policy `json:"policy"`
	Message string            `json:"message"`
}

type deletePolicyResponse struct {
	Message string `json:"message"`
}

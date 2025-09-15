// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/google/uuid"
	"github.com/ultraviolet/cube/guardrails"
)

// createPolicyEndpoint creates endpoint for creating a policy
func createPolicyEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createPolicyRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}

		policy := guardrails.Policy{
			ID:          uuid.New().String(),
			Name:        req.Name,
			Description: req.Description,
			Enabled:     req.Enabled,
			Rules:       req.Rules,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
			UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		}

		if err := svc.CreatePolicy(ctx, policy); err != nil {
			return nil, err
		}

		return createPolicyResponse{
			ID:      policy.ID,
			Message: "Policy created successfully",
		}, nil
	}
}

// getPolicyEndpoint creates endpoint for getting a policy by ID
func getPolicyEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getPolicyRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}

		policy, err := svc.GetPolicy(ctx, req.ID)
		if err != nil {
			return nil, err
		}

		return getPolicyResponse{Policy: policy}, nil
	}
}

// listPoliciesEndpoint creates endpoint for listing policies
func listPoliciesEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listPoliciesRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}

		policies, err := svc.ListPolicies(ctx, req.Limit, req.Offset)
		if err != nil {
			return nil, err
		}

		return listPoliciesResponse{
			Policies: policies,
			Total:    len(policies),
			Limit:    req.Limit,
			Offset:   req.Offset,
		}, nil
	}
}

// updatePolicyEndpoint creates endpoint for updating a policy
func updatePolicyEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updatePolicyRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}

		policy := guardrails.Policy{
			ID:          req.ID,
			Name:        req.Name,
			Description: req.Description,
			Enabled:     req.Enabled,
			Rules:       req.Rules,
			UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		}

		if err := svc.UpdatePolicy(ctx, policy); err != nil {
			return nil, err
		}

		// Get updated policy to return
		updatedPolicy, err := svc.GetPolicy(ctx, req.ID)
		if err != nil {
			return nil, err
		}

		return updatePolicyResponse{
			Policy:  updatedPolicy,
			Message: "Policy updated successfully",
		}, nil
	}
}

// deletePolicyEndpoint creates endpoint for deleting a policy
func deletePolicyEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deletePolicyRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.DeletePolicy(ctx, req.ID); err != nil {
			return nil, err
		}

		return deletePolicyResponse{
			Message: "Policy deleted successfully",
		}, nil
	}
}

// getRestrictedTopicsEndpoint creates endpoint for getting restricted topics
func getRestrictedTopicsEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// Use repository through service
		topics, err := svc.GetRestrictedTopics(ctx)
		if err != nil {
			return nil, err
		}
		return getRestrictedTopicsResponse{Topics: topics}, nil
	}
}

// updateRestrictedTopicsEndpoint creates endpoint for updating restricted topics
func updateRestrictedTopicsEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateTopicsRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}
		if err := svc.UpdateRestrictedTopics(ctx, req.Topics); err != nil {
			return nil, err
		}
		return messageResponse{Message: "Topics updated successfully"}, nil
	}
}

// addRestrictedTopicEndpoint creates endpoint for adding a restricted topic
func addRestrictedTopicEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addTopicRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}
		if err := svc.AddRestrictedTopic(ctx, req.Topic); err != nil {
			return nil, err
		}
		return messageResponse{Message: "Topic added successfully"}, nil
	}
}

// removeRestrictedTopicEndpoint creates endpoint for removing a restricted topic
func removeRestrictedTopicEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeTopicRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}
		if err := svc.RemoveRestrictedTopic(ctx, req.Topic); err != nil {
			return nil, err
		}
		return messageResponse{Message: "Topic removed successfully"}, nil
	}
}

// getBiasPatternsEndpoint creates endpoint for getting bias patterns
func getBiasPatternsEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		patterns, err := svc.GetBiasPatterns(ctx)
		if err != nil {
			return nil, err
		}
		return getBiasPatternsResponse{Patterns: patterns}, nil
	}
}

// updateBiasPatternsEndpoint creates endpoint for updating bias patterns
func updateBiasPatternsEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateBiasPatternsRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}
		if err := svc.UpdateBiasPatterns(ctx, req.Patterns); err != nil {
			return nil, err
		}
		return messageResponse{Message: "Bias patterns updated successfully"}, nil
	}
}

// getFactualityConfigEndpoint creates endpoint for getting factuality configuration
func getFactualityConfigEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		config, err := svc.GetFactualityConfig(ctx)
		if err != nil {
			return nil, err
		}
		return getFactualityConfigResponse{Config: config}, nil
	}
}

func updateFactualityConfigEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateFactualityConfigRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}
		if err := svc.UpdateFactualityConfig(ctx, req.Config); err != nil {
			return nil, err
		}
		return messageResponse{Message: "Factuality config updated successfully"}, nil
	}
}

// getAuditLogEndpoint creates endpoint for getting audit log
func getAuditLogEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getAuditLogRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}
		entries, err := svc.GetAuditLogs(ctx, req.Limit)
		if err != nil {
			return nil, err
		}
		return getAuditLogResponse{
			Entries: entries,
			Total:   len(entries),
		}, nil
	}
}

// exportConfigEndpoint creates endpoint for exporting configuration
func exportConfigEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		data, err := svc.ExportConfig(ctx)
		if err != nil {
			return nil, err
		}
		// Return raw bytes for YAML download
		return data, nil
	}
}

// importConfigEndpoint creates endpoint for importing configuration
func importConfigEndpoint(svc guardrails.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(importConfigRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}
		if err := svc.ImportConfig(ctx, req.Data); err != nil {
			return nil, err
		}
		return messageResponse{Message: "Config imported successfully"}, nil
	}
}

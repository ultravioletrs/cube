// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"

	"github.com/absmach/supermq/pkg/postgres"
	"github.com/ultravioletrs/cube/proxy"
	"github.com/ultravioletrs/cube/proxy/router"
)

var _ proxy.Repository = (*repository)(nil)

type repository struct {
	db postgres.Database
}

func NewRepository(db postgres.Database) proxy.Repository {
	return &repository{db: db}
}

// GetAttestationPolicy implements proxy.Repository.
func (r *repository) GetAttestationPolicy(ctx context.Context) ([]byte, error) {
	q := "SELECT policy FROM attestation_policy ORDER BY id DESC LIMIT 1"

	var policy []byte

	row := r.db.QueryRowxContext(ctx, q)
	if err := row.Scan(&policy); err != nil {
		return nil, err
	}

	return policy, nil
}

// UpdateAttestationPolicy implements proxy.Repository.
func (r *repository) UpdateAttestationPolicy(ctx context.Context, policy []byte) error {
	q := "INSERT INTO attestation_policy (policy) VALUES ($1)"
	_, err := r.db.ExecContext(ctx, q, policy)

	return err
}

// CreateRoute implements proxy.Repository.
func (r *repository) CreateRoute(ctx context.Context, route *router.RouteRule) (*router.RouteRule, error) {
	q := `INSERT INTO routes (name, target_url, matchers, priority, default_rule, strip_prefix, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (name) DO UPDATE SET
			target_url = EXCLUDED.target_url,
			matchers = EXCLUDED.matchers,
			priority = EXCLUDED.priority,
			default_rule = EXCLUDED.default_rule,
			strip_prefix = EXCLUDED.strip_prefix,
			enabled = EXCLUDED.enabled,			
			updated_at = CURRENT_TIMESTAMP
		RETURNING name, target_url, matchers, priority, default_rule, strip_prefix, enabled`

	matchersJSON, err := json.Marshal(route.Matchers)
	if err != nil {
		return nil, err
	}

	var (
		returnedMatchersJSON []byte
		createdRoute         router.RouteRule
	)

	enabled := route.Enabled == nil || *route.Enabled

	row := r.db.QueryRowxContext(
		ctx, q, route.Name, route.TargetURL, matchersJSON, route.Priority, route.DefaultRule, route.StripPrefix, enabled)

	if err := row.Scan(
		&createdRoute.Name, &createdRoute.TargetURL, &returnedMatchersJSON,
		&createdRoute.Priority, &createdRoute.DefaultRule, &createdRoute.StripPrefix, &enabled); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(returnedMatchersJSON, &createdRoute.Matchers); err != nil {
		return nil, err
	}

	return &createdRoute, nil
}

// GetRoute implements proxy.Repository.
func (r *repository) GetRoute(ctx context.Context, name string) (*router.RouteRule, error) {
	q := `SELECT id, name, target_url, matchers, priority, default_rule, strip_prefix, enabled, created_at, updated_at
		FROM routes WHERE name = $1`

	var (
		id           int
		matchersJSON []byte
		route        router.RouteRule
		enabled      bool
	)

	row := r.db.QueryRowxContext(ctx, q, name)

	err := row.Scan(
		&id, &route.Name, &route.TargetURL, &matchersJSON, &route.Priority,
		&route.DefaultRule, &route.StripPrefix, &enabled, nil, nil)
	if err != nil {
		return nil, err
	}

	route.Enabled = &enabled

	if err := json.Unmarshal(matchersJSON, &route.Matchers); err != nil {
		return nil, err
	}

	return &route, nil
}

// UpdateRoute implements proxy.Repository.
func (r *repository) UpdateRoute(ctx context.Context, route *router.RouteRule) (*router.RouteRule, error) {
	q := `UPDATE routes SET
		target_url = $1,
		matchers = $2,
		priority = $3,
		default_rule = $4,
		strip_prefix = $5,
		enabled = $6,
		updated_at = CURRENT_TIMESTAMP
		WHERE name = $7
		RETURNING name, target_url, matchers, priority, default_rule, strip_prefix, enabled`

	matchersJSON, err := json.Marshal(route.Matchers)
	if err != nil {
		return nil, err
	}

	var (
		returnedMatchersJSON []byte
		updatedRoute         router.RouteRule
	)

	enabled := route.Enabled == nil || *route.Enabled

	row := r.db.QueryRowxContext(
		ctx, q, route.TargetURL, matchersJSON, route.Priority, route.DefaultRule, route.StripPrefix, enabled, route.Name)

	if err := row.Scan(
		&updatedRoute.Name, &updatedRoute.TargetURL, &returnedMatchersJSON,
		&updatedRoute.Priority, &updatedRoute.DefaultRule, &updatedRoute.StripPrefix, &enabled); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(returnedMatchersJSON, &updatedRoute.Matchers); err != nil {
		return nil, err
	}

	return &updatedRoute, nil
}

// DeleteRoute implements proxy.Repository.
func (r *repository) DeleteRoute(ctx context.Context, name string) error {
	q := "DELETE FROM routes WHERE name = $1"
	_, err := r.db.ExecContext(ctx, q, name)

	return err
}

// ListRoutes implements proxy.Repository.
func (r *repository) ListRoutes(ctx context.Context, offset, limit uint64) ([]router.RouteRule, uint64, error) {
	q := `SELECT name, target_url, matchers, priority, default_rule, strip_prefix, enabled
		FROM routes ORDER BY priority DESC, name ASC OFFSET $1 LIMIT $2`

	rows, err := r.db.QueryxContext(ctx, q, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var routes []router.RouteRule

	for rows.Next() {
		var (
			matchersJSON []byte
			route        router.RouteRule
			enabled      bool
		)

		if err := rows.Scan(
			&route.Name, &route.TargetURL, &matchersJSON, &route.Priority, &route.DefaultRule,
			&route.StripPrefix, &enabled); err != nil {
			return nil, 0, err
		}

		route.Enabled = &enabled

		if err := json.Unmarshal(matchersJSON, &route.Matchers); err != nil {
			return nil, 0, err
		}

		routes = append(routes, route)
	}

	cq := "SELECT COUNT(*) FROM routes"

	var total uint64
	if err := r.db.QueryRowxContext(ctx, cq).Scan(&total); err != nil {
		return nil, 0, err
	}

	return routes, total, rows.Err()
}

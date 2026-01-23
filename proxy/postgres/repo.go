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
func (r *repository) CreateRoute(ctx context.Context, route *router.RouteRule) error {
	q := `INSERT INTO routes (name, target_url, matchers, priority, default_rule, strip_prefix)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (name) DO UPDATE SET
			target_url = EXCLUDED.target_url,
			matchers = EXCLUDED.matchers,
			priority = EXCLUDED.priority,
			default_rule = EXCLUDED.default_rule,
			strip_prefix = EXCLUDED.strip_prefix,
			updated_at = CURRENT_TIMESTAMP`

	matchersJSON, err := json.Marshal(route.Matchers)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(
		ctx, q, route.Name, route.TargetURL, matchersJSON, route.Priority, route.DefaultRule, route.StripPrefix)

	return err
}

// GetRoute implements proxy.Repository.
func (r *repository) GetRoute(ctx context.Context, name string) (*router.RouteRule, error) {
	q := `SELECT id, name, target_url, matchers, priority, default_rule, strip_prefix, created_at, updated_at
		FROM routes WHERE name = $1`

	var (
		id           int
		matchersJSON []byte
		route        router.RouteRule
	)

	row := r.db.QueryRowxContext(ctx, q, name)

	err := row.Scan(
		&id, &route.Name, &route.TargetURL, &matchersJSON, &route.Priority, &route.DefaultRule, &route.StripPrefix, nil, nil)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(matchersJSON, &route.Matchers); err != nil {
		return nil, err
	}

	return &route, nil
}

// UpdateRoute implements proxy.Repository.
func (r *repository) UpdateRoute(ctx context.Context, route *router.RouteRule) error {
	q := `UPDATE routes SET
		target_url = $1,
		matchers = $2,
		priority = $3,
		default_rule = $4,
		strip_prefix = $5,
		updated_at = CURRENT_TIMESTAMP
		WHERE name = $6`

	matchersJSON, err := json.Marshal(route.Matchers)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(
		ctx, q, route.TargetURL, matchersJSON, route.Priority, route.DefaultRule, route.StripPrefix, route.Name)

	return err
}

// DeleteRoute implements proxy.Repository.
func (r *repository) DeleteRoute(ctx context.Context, name string) error {
	q := "DELETE FROM routes WHERE name = $1"
	_, err := r.db.ExecContext(ctx, q, name)

	return err
}

// ListRoutes implements proxy.Repository.
func (r *repository) ListRoutes(ctx context.Context) ([]router.RouteRule, error) {
	q := `SELECT name, target_url, matchers, priority, default_rule, strip_prefix
		FROM routes ORDER BY priority DESC, name ASC`

	rows, err := r.db.QueryxContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []router.RouteRule

	for rows.Next() {
		var (
			matchersJSON []byte
			route        router.RouteRule
		)

		if err := rows.Scan(
			&route.Name, &route.TargetURL, &matchersJSON, &route.Priority, &route.DefaultRule, &route.StripPrefix); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(matchersJSON, &route.Matchers); err != nil {
			return nil, err
		}

		routes = append(routes, route)
	}

	return routes, rows.Err()
}

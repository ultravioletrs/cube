// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"

	"github.com/absmach/supermq/pkg/postgres"
	"github.com/ultraviolet/cube/proxy"
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

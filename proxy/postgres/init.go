// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package postgres

import migrate "github.com/rubenv/sql-migrate"

func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "20250101000001_create_attestation_policy_table",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS attestation_policy (
						id SERIAL PRIMARY KEY,
						policy JSONB NOT NULL
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS attestation_policy`,
				},
			},
			{
				Id: "20250101000002_create_routes_table",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS routes (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL UNIQUE,
						target_url VARCHAR(1024) NOT NULL,
						matchers JSONB NOT NULL DEFAULT '[]',
						priority INTEGER NOT NULL DEFAULT 0,
						default_rule BOOLEAN NOT NULL DEFAULT false,
						created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
						updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS routes`,
				},
			},
			{
				Id: "20260101000003_add_strip_prefix_to_routes",
				Up: []string{
					`ALTER TABLE routes ADD COLUMN IF NOT EXISTS strip_prefix VARCHAR(255) NOT NULL DEFAULT ''`,
				},
				Down: []string{
					`ALTER TABLE routes DROP COLUMN IF EXISTS strip_prefix`,
				},
			},
		},
	}
}

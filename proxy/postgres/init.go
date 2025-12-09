// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package postgres

import migrate "github.com/rubenv/sql-migrate"

func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "create_attestation_policy_table",
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
		},
	}
}

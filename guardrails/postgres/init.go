// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package postgres

import migrate "github.com/rubenv/sql-migrate"

func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "guardrails_initial_setup",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS flows (
						id VARCHAR(36) PRIMARY KEY,
						name VARCHAR(255) NOT NULL UNIQUE,
						description TEXT,
						content TEXT NOT NULL,
						type VARCHAR(50) NOT NULL CHECK (type IN ('input', 'output', 'dialog', 'subflow')),
						active BOOLEAN NOT NULL,
						version INTEGER NOT NULL,
						created_at TIMESTAMP NOT NULL,
						updated_at TIMESTAMP NOT NULL
					)`,
					`CREATE TABLE IF NOT EXISTS kb_files (
						id VARCHAR(36) PRIMARY KEY,
						name VARCHAR(255) NOT NULL,
						content TEXT NOT NULL,
						type VARCHAR(50) NOT NULL,
						category VARCHAR(100) NOT NULL,
						tags TEXT[],
						metadata JSONB,
						active BOOLEAN NOT NULL,
						version INTEGER NOT NULL,
						created_at TIMESTAMP NOT NULL,
						updated_at TIMESTAMP NOT NULL,
						UNIQUE(name, category)
					)`,
				},
				Down: []string{
					"DROP TABLE IF EXISTS kb_files",
					"DROP TABLE IF EXISTS flows",
				},
			},
		},
	}
}

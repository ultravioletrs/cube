# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

import asyncio
import logging
import os
from pathlib import Path

import asyncpg

logger = logging.getLogger(__name__)


async def run_migrations(
    host: str,
    port: int,
    user: str,
    password: str,
    database: str,
) -> None:
    """
    Run database migrations.

    Args:
        host: Database host
        port: Database port
        user: Database user
        password: Database password
        database: Database name
    """
    dsn = f"postgresql://{user}:{password}@{host}:{port}/{database}"

    logger.info(f"Connecting to database {host}:{port}/{database}")
    conn = await asyncpg.connect(dsn)

    try:
        # Create migrations tracking table
        await conn.execute(
            """
            CREATE TABLE IF NOT EXISTS schema_migrations (
                version VARCHAR(255) PRIMARY KEY,
                applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
            )
            """
        )

        # Get applied migrations
        applied = set()
        rows = await conn.fetch("SELECT version FROM schema_migrations")
        for row in rows:
            applied.add(row["version"])

        # Find migration files
        migrations_dir = Path(__file__).parent
        migration_files = sorted(migrations_dir.glob("*.sql"))

        for migration_file in migration_files:
            version = migration_file.stem
            if version in applied:
                logger.debug(f"Migration {version} already applied")
                continue

            logger.info(f"Applying migration {version}")

            # Read and execute migration
            sql = migration_file.read_text()
            await conn.execute(sql)

            # Record migration
            await conn.execute(
                "INSERT INTO schema_migrations (version) VALUES ($1)",
                version,
            )

            logger.info(f"Migration {version} applied successfully")

        logger.info("All migrations complete")

    finally:
        await conn.close()


async def main():
    """Run migrations from command line."""
    logging.basicConfig(level=logging.INFO)

    host = os.environ.get("UV_GUARDRAILS_DB_HOST", "localhost")
    port = int(os.environ.get("UV_GUARDRAILS_DB_PORT", "5432"))
    user = os.environ.get("UV_GUARDRAILS_DB_USER", "guardrails")
    password = os.environ.get("UV_GUARDRAILS_DB_PASS", "guardrails")
    database = os.environ.get("UV_GUARDRAILS_DB_NAME", "guardrails")

    await run_migrations(host, port, user, password, database)


if __name__ == "__main__":
    asyncio.run(main())

# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

import logging
from datetime import datetime, timezone
from typing import List, Optional
from uuid import UUID

import asyncpg

from src.adapters.exceptions import RepositoryError
from src.domain.entities import GuardrailConfig, GuardrailVersion, MaterializedGuardrail
from src.ports.repositories import GuardrailRepository

logger = logging.getLogger(__name__)


class PostgresGuardrailRepository(GuardrailRepository):
    """
    PostgreSQL implementation of the GuardrailRepository port.
    Uses asyncpg for async database operations.
    """

    def __init__(self, pool: asyncpg.Pool):
        self.pool = pool

    # ==================== Config CRUD ====================

    async def create_config(self, config: GuardrailConfig) -> GuardrailConfig:
        """Create a new guardrail configuration."""
        try:
            async with self.pool.acquire() as conn:
                await conn.execute(
                    """
                    INSERT INTO guardrail_configs
                    (id, name, description, config_yaml, prompts_yaml, colang, created_at, updated_at)
                    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
                    """,
                    config.id,
                    config.name,
                    config.description,
                    config.config_yaml,
                    config.prompts_yaml,
                    config.colang,
                    config.created_at,
                    config.updated_at,
                )
                return config
        except asyncpg.UniqueViolationError as e:
            raise RepositoryError(f"Config already exists: {e}")
        except Exception as e:
            logger.error(f"Failed to create config: {e}")
            raise RepositoryError(f"Failed to create config: {e}")

    async def get_config(self, config_id: UUID) -> Optional[GuardrailConfig]:
        """Get a guardrail configuration by ID."""
        try:
            async with self.pool.acquire() as conn:
                row = await conn.fetchrow(
                    """
                    SELECT id, name, description, config_yaml, prompts_yaml, colang,
                           created_at, updated_at
                    FROM guardrail_configs
                    WHERE id = $1
                    """,
                    config_id,
                )
                if not row:
                    return None
                return self._row_to_config(row)
        except Exception as e:
            logger.error(f"Failed to get config: {e}")
            raise RepositoryError(f"Failed to get config: {e}")

    async def get_config_by_name(self, name: str) -> Optional[GuardrailConfig]:
        """Get a guardrail configuration by name."""
        try:
            async with self.pool.acquire() as conn:
                row = await conn.fetchrow(
                    """
                    SELECT id, name, description, config_yaml, prompts_yaml, colang,
                           created_at, updated_at
                    FROM guardrail_configs
                    WHERE name = $1
                    """,
                    name,
                )
                if not row:
                    return None
                return self._row_to_config(row)
        except Exception as e:
            logger.error(f"Failed to get config by name: {e}")
            raise RepositoryError(f"Failed to get config by name: {e}")

    async def list_configs(
        self, offset: int = 0, limit: int = 100
    ) -> List[GuardrailConfig]:
        """List all guardrail configurations with pagination."""
        try:
            async with self.pool.acquire() as conn:
                rows = await conn.fetch(
                    """
                    SELECT id, name, description, config_yaml, prompts_yaml, colang,
                           created_at, updated_at
                    FROM guardrail_configs
                    ORDER BY created_at DESC
                    OFFSET $1 LIMIT $2
                    """,
                    offset,
                    limit,
                )
                return [self._row_to_config(row) for row in rows]
        except Exception as e:
            logger.error(f"Failed to list configs: {e}")
            raise RepositoryError(f"Failed to list configs: {e}")

    async def count_configs(self) -> int:
        """Count total number of guardrail configurations."""
        try:
            async with self.pool.acquire() as conn:
                result = await conn.fetchval(
                    "SELECT COUNT(*) FROM guardrail_configs"
                )
                return result or 0
        except Exception as e:
            logger.error(f"Failed to count configs: {e}")
            raise RepositoryError(f"Failed to count configs: {e}")

    async def update_config(self, config: GuardrailConfig) -> GuardrailConfig:
        """Update an existing guardrail configuration."""
        try:
            async with self.pool.acquire() as conn:
                result = await conn.execute(
                    """
                    UPDATE guardrail_configs
                    SET name = $2, description = $3, config_yaml = $4,
                        prompts_yaml = $5, colang = $6, updated_at = $7
                    WHERE id = $1
                    """,
                    config.id,
                    config.name,
                    config.description,
                    config.config_yaml,
                    config.prompts_yaml,
                    config.colang,
                    config.updated_at,
                )
                if result == "UPDATE 0":
                    raise RepositoryError(f"Config not found: {config.id}")
                return config
        except asyncpg.UniqueViolationError as e:
            raise RepositoryError(f"Config name already exists: {e}")
        except RepositoryError:
            raise
        except Exception as e:
            logger.error(f"Failed to update config: {e}")
            raise RepositoryError(f"Failed to update config: {e}")

    async def delete_config(self, config_id: UUID) -> bool:
        """Delete a guardrail configuration by ID."""
        try:
            async with self.pool.acquire() as conn:
                result = await conn.execute(
                    "DELETE FROM guardrail_configs WHERE id = $1",
                    config_id,
                )
                return result != "DELETE 0"
        except Exception as e:
            logger.error(f"Failed to delete config: {e}")
            raise RepositoryError(f"Failed to delete config: {e}")

    # ==================== Version Management ====================

    async def create_version(self, version: GuardrailVersion) -> GuardrailVersion:
        """Create a new version for a configuration."""
        try:
            async with self.pool.acquire() as conn:
                async with conn.transaction():
                    await conn.execute(
                        """
                        INSERT INTO guardrail_versions
                        (id, config_id, name, revision, is_active, created_at, description)
                        VALUES ($1, $2, $3, $4, $5, $6, $7)
                        """,
                        version.id,
                        version.config_id,
                        version.name,
                        version.revision,
                        version.is_active,
                        version.created_at,
                        version.description,
                    )

                    # Materialize the version
                    await self._materialize(conn, version.id)

                return version
        except Exception as e:
            logger.error(f"Failed to create version: {e}")
            raise RepositoryError(f"Failed to create version: {e}")

    async def get_version(self, version_id: UUID) -> Optional[GuardrailVersion]:
        """Get a version by ID."""
        try:
            async with self.pool.acquire() as conn:
                row = await conn.fetchrow(
                    """
                    SELECT id, config_id, name, revision, is_active, created_at, description
                    FROM guardrail_versions
                    WHERE id = $1
                    """,
                    version_id,
                )
                if not row:
                    return None
                return self._row_to_version(row)
        except Exception as e:
            logger.error(f"Failed to get version: {e}")
            raise RepositoryError(f"Failed to get version: {e}")

    async def list_versions(
        self, config_id: UUID, offset: int = 0, limit: int = 100
    ) -> List[GuardrailVersion]:
        """List all versions for a configuration."""
        try:
            async with self.pool.acquire() as conn:
                rows = await conn.fetch(
                    """
                    SELECT id, config_id, name, revision, is_active, created_at, description
                    FROM guardrail_versions
                    WHERE config_id = $1
                    ORDER BY revision DESC
                    OFFSET $2 LIMIT $3
                    """,
                    config_id,
                    offset,
                    limit,
                )
                return [self._row_to_version(row) for row in rows]
        except Exception as e:
            logger.error(f"Failed to list versions: {e}")
            raise RepositoryError(f"Failed to list versions: {e}")

    async def count_versions(self, config_id: UUID) -> int:
        """Count total number of versions for a configuration."""
        try:
            async with self.pool.acquire() as conn:
                result = await conn.fetchval(
                    "SELECT COUNT(*) FROM guardrail_versions WHERE config_id = $1",
                    config_id,
                )
                return result or 0
        except Exception as e:
            logger.error(f"Failed to count versions: {e}")
            raise RepositoryError(f"Failed to count versions: {e}")

    async def get_latest_version(self, config_id: UUID) -> Optional[GuardrailVersion]:
        """Get the latest version for a configuration."""
        try:
            async with self.pool.acquire() as conn:
                row = await conn.fetchrow(
                    """
                    SELECT id, config_id, name, revision, is_active, created_at, description
                    FROM guardrail_versions
                    WHERE config_id = $1
                    ORDER BY revision DESC
                    LIMIT 1
                    """,
                    config_id,
                )
                if not row:
                    return None
                return self._row_to_version(row)
        except Exception as e:
            logger.error(f"Failed to get latest version: {e}")
            raise RepositoryError(f"Failed to get latest version: {e}")

    async def get_active_version(self) -> Optional[GuardrailVersion]:
        """Get the currently active version."""
        try:
            async with self.pool.acquire() as conn:
                row = await conn.fetchrow(
                    """
                    SELECT id, config_id, name, revision, is_active, created_at, description
                    FROM guardrail_versions
                    WHERE is_active = true
                    LIMIT 1
                    """
                )
                if not row:
                    return None
                return self._row_to_version(row)
        except Exception as e:
            logger.error(f"Failed to get active version: {e}")
            raise RepositoryError(f"Failed to get active version: {e}")

    # ==================== Activation ====================

    async def deactivate_all(self) -> None:
        """Deactivate all versions."""
        try:
            async with self.pool.acquire() as conn:
                await conn.execute(
                    "UPDATE guardrail_versions SET is_active = false WHERE is_active = true"
                )
        except Exception as e:
            logger.error(f"Failed to deactivate all versions: {e}")
            raise RepositoryError(f"Failed to deactivate all versions: {e}")

    async def activate(self, version_id: UUID) -> None:
        """Activate a specific version."""
        try:
            async with self.pool.acquire() as conn:
                async with conn.transaction():
                    # Deactivate all first
                    await conn.execute(
                        "UPDATE guardrail_versions SET is_active = false WHERE is_active = true"
                    )
                    # Activate the specified version
                    result = await conn.execute(
                        "UPDATE guardrail_versions SET is_active = true WHERE id = $1",
                        version_id,
                    )
                    if result == "UPDATE 0":
                        raise RepositoryError(f"Version not found: {version_id}")
        except RepositoryError:
            raise
        except Exception as e:
            logger.error(f"Failed to activate version: {e}")
            raise RepositoryError(f"Failed to activate version: {e}")

    # ==================== Materialization ====================

    async def fetch_active_materialized(self) -> Optional[MaterializedGuardrail]:
        """Fetch the materialized active guardrail."""
        try:
            async with self.pool.acquire() as conn:
                row = await conn.fetchrow(
                    """
                    SELECT m.version_id, m.config_yaml, m.prompts_yaml, m.colang, v.revision
                    FROM guardrail_materialized m
                    JOIN guardrail_versions v ON v.id = m.version_id
                    WHERE v.is_active = true
                    """
                )
                if not row:
                    return None
                return MaterializedGuardrail(
                    version_id=row["version_id"],
                    config_yaml=row["config_yaml"],
                    prompts_yaml=row["prompts_yaml"],
                    colang=row["colang"],
                    revision=row["revision"],
                )
        except Exception as e:
            logger.error(f"Failed to fetch active materialized: {e}")
            raise RepositoryError(f"Failed to fetch active materialized: {e}")

    async def materialize_version(self, version_id: UUID) -> MaterializedGuardrail:
        """Materialize a version for runtime use."""
        try:
            async with self.pool.acquire() as conn:
                await self._materialize(conn, version_id)

                row = await conn.fetchrow(
                    """
                    SELECT m.version_id, m.config_yaml, m.prompts_yaml, m.colang, v.revision
                    FROM guardrail_materialized m
                    JOIN guardrail_versions v ON v.id = m.version_id
                    WHERE m.version_id = $1
                    """,
                    version_id,
                )
                if not row:
                    raise RepositoryError(f"Failed to materialize version: {version_id}")

                return MaterializedGuardrail(
                    version_id=row["version_id"],
                    config_yaml=row["config_yaml"],
                    prompts_yaml=row["prompts_yaml"],
                    colang=row["colang"],
                    revision=row["revision"],
                )
        except RepositoryError:
            raise
        except Exception as e:
            logger.error(f"Failed to materialize version: {e}")
            raise RepositoryError(f"Failed to materialize version: {e}")

    async def _materialize(self, conn: asyncpg.Connection, version_id: UUID) -> None:
        """Internal method to materialize a version."""
        # Get version with config
        row = await conn.fetchrow(
            """
            SELECT v.id, c.config_yaml, c.prompts_yaml, c.colang
            FROM guardrail_versions v
            JOIN guardrail_configs c ON c.id = v.config_id
            WHERE v.id = $1
            """,
            version_id,
        )
        if not row:
            raise RepositoryError(f"Version not found: {version_id}")

        # Insert or update materialized view
        await conn.execute(
            """
            INSERT INTO guardrail_materialized (version_id, config_yaml, prompts_yaml, colang, updated_at)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (version_id)
            DO UPDATE SET config_yaml = $2, prompts_yaml = $3, colang = $4, updated_at = $5
            """,
            version_id,
            row["config_yaml"],
            row["prompts_yaml"],
            row["colang"],
            datetime.now(timezone.utc),
        )

    # ==================== Helpers ====================

    def _row_to_config(self, row: asyncpg.Record) -> GuardrailConfig:
        """Convert a database row to GuardrailConfig entity."""
        return GuardrailConfig(
            id=row["id"],
            name=row["name"],
            description=row["description"],
            config_yaml=row["config_yaml"],
            prompts_yaml=row["prompts_yaml"],
            colang=row["colang"],
            created_at=row["created_at"],
            updated_at=row["updated_at"],
        )

    def _row_to_version(self, row: asyncpg.Record) -> GuardrailVersion:
        """Convert a database row to GuardrailVersion entity."""
        return GuardrailVersion(
            id=row["id"],
            config_id=row["config_id"],
            name=row["name"],
            revision=row["revision"],
            is_active=row["is_active"],
            created_at=row["created_at"],
            description=row["description"],
        )

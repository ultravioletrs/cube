# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

import logging
from typing import Optional

import asyncpg

logger = logging.getLogger(__name__)

_pool: Optional[asyncpg.Pool] = None


async def create_pool(
    host: str,
    port: int,
    user: str,
    password: str,
    database: str,
    min_size: int = 5,
    max_size: int = 20,
) -> asyncpg.Pool:
    """
    Create a connection pool to PostgreSQL.

    Args:
        host: Database host
        port: Database port
        user: Database user
        password: Database password
        database: Database name
        min_size: Minimum pool size
        max_size: Maximum pool size

    Returns:
        asyncpg.Pool connection pool
    """
    global _pool

    if _pool is not None:
        return _pool

    dsn = f"postgresql://{user}:{password}@{host}:{port}/{database}"

    logger.info(f"Creating database pool to {host}:{port}/{database}")

    _pool = await asyncpg.create_pool(
        dsn,
        min_size=min_size,
        max_size=max_size,
    )

    logger.info("Database pool created successfully")
    return _pool


async def close_pool() -> None:
    """Close the connection pool."""
    global _pool

    if _pool is not None:
        logger.info("Closing database pool")
        await _pool.close()
        _pool = None


def get_pool() -> Optional[asyncpg.Pool]:
    """Get the current connection pool."""
    return _pool

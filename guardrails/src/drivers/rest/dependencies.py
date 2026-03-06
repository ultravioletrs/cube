# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

import glob
import hashlib
import logging
import os
from typing import Optional, Tuple

import asyncpg

from src.adapters.repositories import PostgresGuardrailRepository
from src.adapters.runtime import NemoRuntime
from src.infrastructure.db import create_pool, close_pool
from src.ports.repositories import GuardrailRepository
from src.ports.runtime import GuardrailRuntime
from src.use_cases import (
    ActivateVersion,
    CreateConfig,
    CreateVersion,
    DeleteConfig,
    GetConfig,
    ListConfigs,
    ListVersions,
    LoadActiveGuardrail,
    UpdateConfig,
)

logger = logging.getLogger(__name__)

# Global instances
_pool: Optional[asyncpg.Pool] = None
_repository: Optional[GuardrailRepository] = None
_runtime: Optional[NemoRuntime] = None


async def init_dependencies() -> None:
    """Initialize all dependencies (database pool, runtime, etc.)."""
    global _pool, _repository, _runtime

    # Get database configuration from environment
    db_host = os.environ.get("UV_GUARDRAILS_DB_HOST", "guardrails-db")
    db_port = int(os.environ.get("UV_GUARDRAILS_DB_PORT", "5432"))
    db_user = os.environ.get("UV_GUARDRAILS_DB_USER", "guardrails")
    db_password = os.environ.get("UV_GUARDRAILS_DB_PASS", "guardrails")
    db_name = os.environ.get("UV_GUARDRAILS_DB_NAME", "guardrails")

    logger.info("Initializing dependencies...")

    # Create database pool
    _pool = await create_pool(
        host=db_host,
        port=db_port,
        user=db_user,
        password=db_password,
        database=db_name,
    )

    # Create repository
    _repository = PostgresGuardrailRepository(_pool)

    # Create runtime
    _runtime = NemoRuntime()

    # Sync default configuration
    try:
        await _sync_default_config()
    except Exception as e:
        logger.error(f"Failed to sync default configuration: {e}")

    # Load active configuration into runtime
    try:
        loader = LoadActiveGuardrail(_repository)
        materialized = await loader.execute()
        await _runtime.swap(materialized)
        logger.info(f"Loaded active configuration revision {materialized.revision}")
    except Exception as e:
        logger.error(f"No active configuration available: {e}")

    logger.info("Dependencies initialized successfully")


async def shutdown_dependencies() -> None:
    """Shutdown all dependencies."""
    global _pool, _repository, _runtime

    logger.info("Shutting down dependencies...")

    await close_pool()
    _pool = None
    _repository = None
    _runtime = None

    logger.info("Dependencies shut down successfully")


def get_repository() -> GuardrailRepository:
    """Get the guardrail repository instance."""
    if _repository is None:
        raise RuntimeError("Dependencies not initialized")
    return _repository


def get_runtime() -> GuardrailRuntime:
    """Get the guardrail runtime instance."""
    if _runtime is None:
        raise RuntimeError("Dependencies not initialized")
    return _runtime


# Use case factories
def get_create_config() -> CreateConfig:
    """Get CreateConfig use case."""
    return CreateConfig(get_repository())


def get_get_config() -> GetConfig:
    """Get GetConfig use case."""
    return GetConfig(get_repository())


def get_list_configs() -> ListConfigs:
    """Get ListConfigs use case."""
    return ListConfigs(get_repository())


def get_update_config() -> UpdateConfig:
    """Get UpdateConfig use case."""
    return UpdateConfig(get_repository())


def get_delete_config() -> DeleteConfig:
    """Get DeleteConfig use case."""
    return DeleteConfig(get_repository())


def get_create_version() -> CreateVersion:
    """Get CreateVersion use case."""
    return CreateVersion(get_repository())


def get_activate_version() -> ActivateVersion:
    """Get ActivateVersion use case."""
    return ActivateVersion(get_repository())


def get_list_versions() -> ListVersions:
    """Get ListVersions use case."""
    return ListVersions(get_repository())


def get_load_active_guardrail() -> LoadActiveGuardrail:
    """Get LoadActiveGuardrail use case."""
    return LoadActiveGuardrail(get_repository())


def _read_rails_files(rails_dir: str = "./rails") -> Tuple[str, str, str]:
    if not os.path.exists(rails_dir):
        raise FileNotFoundError(f"Rails directory {rails_dir} not found")

    # Read config.yml (required)
    config_path = os.path.join(rails_dir, "config.yml")
    if not os.path.exists(config_path):
        raise FileNotFoundError("config.yml not found in rails directory")

    with open(config_path, "r") as f:
        config_yaml = f.read()

    prompts_yaml = ""
    prompts_path = os.path.join(rails_dir, "prompts.yml")
    if os.path.exists(prompts_path):
        with open(prompts_path, "r") as f:
            prompts_yaml = f.read()

    colang = ""
    co_files = sorted(glob.glob(os.path.join(rails_dir, "*.co")))
    for file in co_files:
        with open(file, "r") as f:
            colang += f.read() + "\n"

    return config_yaml, prompts_yaml, colang


def _compute_content_hash(config_yaml: str, prompts_yaml: str, colang: str) -> str:
    combined = f"{config_yaml}\n---\n{prompts_yaml}\n---\n{colang}"
    return hashlib.sha256(combined.encode()).hexdigest()


def _content_matches(
    config_yaml: str, prompts_yaml: str, colang: str,
    db_config_yaml: str, db_prompts_yaml: str, db_colang: str
) -> bool:
    file_hash = _compute_content_hash(config_yaml, prompts_yaml, colang)
    db_hash = _compute_content_hash(db_config_yaml, db_prompts_yaml, db_colang)
    return file_hash == db_hash


async def _sync_default_config() -> None:
    logger.info("Synchronizing default configuration from ./rails...")

    rails_dir = "./rails"

    try:
        config_yaml, prompts_yaml, colang = _read_rails_files(rails_dir)
    except FileNotFoundError as e:
        logger.warning(f"Cannot sync config: {e}")
        return

    repository = get_repository()

    existing_config = await repository.get_config_by_name("default-config")

    if existing_config is None:
        logger.info("No default config found, creating new configuration...")
        await _create_new_default_config(config_yaml, prompts_yaml, colang)
        return

    if _content_matches(
        config_yaml, prompts_yaml, colang,
        existing_config.config_yaml, existing_config.prompts_yaml, existing_config.colang
    ):
        logger.info("Default config is up to date, no changes detected")
        return

    logger.info("File changes detected, updating default configuration...")

    update_config_uc = get_update_config()
    await update_config_uc.execute(
        config_id=existing_config.id,
        config_yaml=config_yaml,
        prompts_yaml=prompts_yaml,
        colang=colang,
    )
    logger.info(f"Updated default config: {existing_config.id}")

    # Create new version with semantic versioning
    next_version = await _get_next_semantic_version(existing_config.id)
    create_version_uc = get_create_version()
    version = await create_version_uc.execute(
        config_id=existing_config.id,
        name=next_version,
        description="Auto-updated from file system changes",
    )
    logger.info(f"Created new version: {version.id} (revision {version.revision})")

    activate_version_uc = get_activate_version()
    await activate_version_uc.execute(version.id)
    logger.info(f"Activated new version: {version.id}")


async def _get_next_semantic_version(config_id) -> str:
    """Get next semantic version based on existing versions."""
    import re

    list_versions_uc = get_list_versions()
    result = await list_versions_uc.execute(config_id, offset=0, limit=1000)

    if not result.versions:
        return "v1.0.0"

    # Parse existing versions and find the highest
    max_major, max_minor, max_patch = 0, 0, 0
    semver_pattern = re.compile(r"^v?(\d+)\.(\d+)\.(\d+)$")

    for ver in result.versions:
        match = semver_pattern.match(ver.name)
        if match:
            major, minor, patch = int(match.group(1)), int(match.group(2)), int(match.group(3))
            if (major, minor, patch) > (max_major, max_minor, max_patch):
                max_major, max_minor, max_patch = major, minor, patch

    # Increment patch version for auto-updates
    return f"v{max_major}.{max_minor}.{max_patch + 1}"


async def _create_new_default_config(config_yaml: str, prompts_yaml: str, colang: str) -> None:
    """Create a new default configuration."""
    # Create config
    create_config_uc = get_create_config()
    config = await create_config_uc.execute(
        name="default-config",
        description="Default configuration initialized from file system",
        config_yaml=config_yaml,
        prompts_yaml=prompts_yaml,
        colang=colang,
    )
    logger.info(f"Created default config: {config.id}")

    # Create version
    create_version_uc = get_create_version()
    version = await create_version_uc.execute(
        config_id=config.id,
        name="v1.0.0",
        description="Initial default version",
    )
    logger.info(f"Created default version: {version.id}")

    # Activate version
    activate_version_uc = get_activate_version()
    await activate_version_uc.execute(version.id)
    logger.info(f"Activated default version: {version.id}")


async def _create_default_config() -> None:
    await _sync_default_config()

# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

import logging
import os
from typing import Optional

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

    # Try to load active configuration
    try:
        loader = LoadActiveGuardrail(_repository)
        materialized = await loader.execute()
        await _runtime.swap(materialized)
        logger.info(f"Loaded active configuration revision {materialized.revision}")
    except Exception as e:
        logger.warning(f"No active configuration found: {e}")
        logger.info("Initializing default configuration from ./rails...")
        
        try:
            await _create_default_config()
            # Try loading the newly created config
            loader = LoadActiveGuardrail(_repository)
            materialized = await loader.execute()
            await _runtime.swap(materialized)
            logger.info(f"Loaded default configuration revision {materialized.revision}")
        except Exception as ex:
            logger.error(f"Failed to initialize default configuration: {ex}")

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



async def _create_default_config() -> None:
    """Create a default configuration if none exists."""
    import glob
    
    logger.info("Creating default configuration from ./rails...")
    
    # Read config files
    rails_dir = "./rails"
    if not os.path.exists(rails_dir):
        logger.warning(f"Rails directory {rails_dir} not found. Skipping default config creation.")
        return

    try:
        with open(os.path.join(rails_dir, "config.yml"), "r") as f:
            config_yaml = f.read()
    except FileNotFoundError:
        logger.warning("config.yml not found in ./rails")
        return

    prompts_yaml = ""
    try:
        with open(os.path.join(rails_dir, "prompts.yml"), "r") as f:
            prompts_yaml = f.read()
    except FileNotFoundError:
        pass

    colang = ""
    for file in glob.glob(os.path.join(rails_dir, "*.co")):
        with open(file, "r") as f:
            colang += f.read() + "\n"

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
        name="v1",
        description="Initial default version",
    )
    logger.info(f"Created default version: {version.id}")

    # Activate version
    activate_version_uc = get_activate_version()
    await activate_version_uc.execute(version.id)
    logger.info(f"Activated default version: {version.id}")

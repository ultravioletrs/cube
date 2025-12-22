# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

import logging
from uuid import UUID

from fastapi import APIRouter, Depends, Query

from src.drivers.rest.dependencies import (
    get_activate_version,
    get_create_config,
    get_create_version,
    get_delete_config,
    get_get_config,
    get_list_configs,
    get_list_versions,
    get_load_active_guardrail,
    get_runtime,
    get_update_config,
)
from src.drivers.rest.routers.schemas import (
    ActivateResponse,
    ConfigCreate,
    ConfigListResponse,
    ConfigResponse,
    ConfigUpdate,
    ReloadResponse,
    VersionCreate,
    VersionListResponse,
    VersionResponse,
)
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

router = APIRouter(prefix="/guardrails", tags=["guardrails"])


# ==================== Config CRUD ====================


@router.post("/configs", response_model=ConfigResponse, tags=["configs"])
async def create_config(
    data: ConfigCreate,
    uc: CreateConfig = Depends(get_create_config),
) -> ConfigResponse:
    """Create a new guardrail configuration."""
    config = await uc.execute(
        name=data.name,
        config_yaml=data.config_yaml,
        prompts_yaml=data.prompts_yaml,
        colang=data.colang,
        description=data.description,
    )
    return ConfigResponse(
        id=config.id,
        name=config.name,
        description=config.description,
        config_yaml=config.config_yaml,
        prompts_yaml=config.prompts_yaml,
        colang=config.colang,
        created_at=config.created_at,
        updated_at=config.updated_at,
    )


@router.get("/configs", response_model=ConfigListResponse, tags=["configs"])
async def list_configs(
    offset: int = Query(0, ge=0),
    limit: int = Query(100, ge=1, le=1000),
    uc: ListConfigs = Depends(get_list_configs),
) -> ConfigListResponse:
    """List all guardrail configurations."""
    result = await uc.execute(offset=offset, limit=limit)
    return ConfigListResponse(
        configs=[
            ConfigResponse(
                id=c.id,
                name=c.name,
                description=c.description,
                config_yaml=c.config_yaml,
                prompts_yaml=c.prompts_yaml,
                colang=c.colang,
                created_at=c.created_at,
                updated_at=c.updated_at,
            )
            for c in result.configs
        ],
        total=result.total,
        offset=result.offset,
        limit=result.limit,
    )


@router.get("/configs/{config_id}", response_model=ConfigResponse, tags=["configs"])
async def get_config(
    config_id: UUID,
    uc: GetConfig = Depends(get_get_config),
) -> ConfigResponse:
    """Get a guardrail configuration by ID."""
    config = await uc.execute(config_id)
    return ConfigResponse(
        id=config.id,
        name=config.name,
        description=config.description,
        config_yaml=config.config_yaml,
        prompts_yaml=config.prompts_yaml,
        colang=config.colang,
        created_at=config.created_at,
        updated_at=config.updated_at,
    )


@router.put("/configs/{config_id}", response_model=ConfigResponse, tags=["configs"])
async def update_config(
    config_id: UUID,
    data: ConfigUpdate,
    uc: UpdateConfig = Depends(get_update_config),
) -> ConfigResponse:
    """Update a guardrail configuration."""
    config = await uc.execute(
        config_id=config_id,
        name=data.name,
        config_yaml=data.config_yaml,
        prompts_yaml=data.prompts_yaml,
        colang=data.colang,
        description=data.description,
    )
    return ConfigResponse(
        id=config.id,
        name=config.name,
        description=config.description,
        config_yaml=config.config_yaml,
        prompts_yaml=config.prompts_yaml,
        colang=config.colang,
        created_at=config.created_at,
        updated_at=config.updated_at,
    )


@router.delete("/configs/{config_id}", tags=["configs"])
async def delete_config(
    config_id: UUID,
    uc: DeleteConfig = Depends(get_delete_config),
) -> dict:
    """Delete a guardrail configuration."""
    await uc.execute(config_id)
    return {"status": "deleted", "config_id": str(config_id)}


# ==================== Version Management ====================


@router.post(
    "/configs/{config_id}/versions",
    response_model=VersionResponse,
    tags=["versions"],
)
async def create_version(
    config_id: UUID,
    data: VersionCreate,
    uc: CreateVersion = Depends(get_create_version),
) -> VersionResponse:
    """Create a new version for a configuration."""
    version = await uc.execute(
        config_id=config_id,
        name=data.name,
        description=data.description,
    )
    return VersionResponse(
        id=version.id,
        config_id=version.config_id,
        name=version.name,
        revision=version.revision,
        is_active=version.is_active,
        description=version.description,
        created_at=version.created_at,
    )


@router.get(
    "/configs/{config_id}/versions",
    response_model=VersionListResponse,
    tags=["versions"],
)
async def list_versions(
    config_id: UUID,
    offset: int = Query(0, ge=0),
    limit: int = Query(100, ge=1, le=1000),
    uc: ListVersions = Depends(get_list_versions),
) -> VersionListResponse:
    """List all versions for a configuration."""
    result = await uc.execute(config_id, offset=offset, limit=limit)
    return VersionListResponse(
        versions=[
            VersionResponse(
                id=v.id,
                config_id=v.config_id,
                name=v.name,
                revision=v.revision,
                is_active=v.is_active,
                description=v.description,
                created_at=v.created_at,
            )
            for v in result.versions
        ],
        total=result.total,
        offset=result.offset,
        limit=result.limit,
    )


# ==================== Activation ====================


@router.post("/versions/{version_id}/activate", response_model=ActivateResponse)
async def activate_version(
    version_id: UUID,
    uc: ActivateVersion = Depends(get_activate_version),
) -> ActivateResponse:
    """Activate a guardrail version."""
    version = await uc.execute(version_id)

    # Trigger runtime reload
    runtime = get_runtime()
    loader = get_load_active_guardrail()
    materialized = await loader.execute()
    await runtime.swap(materialized)

    logger.info(f"Activated version {version_id} with revision {version.revision}")

    return ActivateResponse(
        status="activated",
        version_id=version.id,
        revision=version.revision,
    )


@router.post("/reload", response_model=ReloadResponse)
async def reload_runtime() -> ReloadResponse:
    """Reload the runtime with the active configuration."""
    runtime = get_runtime()
    loader = get_load_active_guardrail()

    try:
        materialized = await loader.execute()
        await runtime.swap(materialized)

        return ReloadResponse(
            status="success",
            revision=materialized.revision,
            message=f"Reloaded configuration revision {materialized.revision}",
        )
    except Exception as e:
        logger.error(f"Reload failed: {e}")
        return ReloadResponse(
            status="failed",
            revision=runtime.get_current_revision(),
            message=str(e),
        )

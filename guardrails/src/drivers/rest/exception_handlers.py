# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

import logging

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

from src.adapters.exceptions import AdapterError, RepositoryError
from src.use_cases.exceptions import (
    ConfigAlreadyExistsError,
    ConfigNotFoundError,
    InvalidConfigError,
    NoActiveVersionError,
    UseCaseError,
    VersionNotFoundError,
)

logger = logging.getLogger(__name__)


def register_exception_handlers(app: FastAPI) -> None:
    """Register exception handlers for the application."""

    @app.exception_handler(ConfigNotFoundError)
    async def config_not_found_handler(
        request: Request, exc: ConfigNotFoundError
    ) -> JSONResponse:
        return JSONResponse(
            status_code=404,
            content={"detail": str(exc), "error_code": "CONFIG_NOT_FOUND"},
        )

    @app.exception_handler(VersionNotFoundError)
    async def version_not_found_handler(
        request: Request, exc: VersionNotFoundError
    ) -> JSONResponse:
        return JSONResponse(
            status_code=404,
            content={"detail": str(exc), "error_code": "VERSION_NOT_FOUND"},
        )

    @app.exception_handler(ConfigAlreadyExistsError)
    async def config_already_exists_handler(
        request: Request, exc: ConfigAlreadyExistsError
    ) -> JSONResponse:
        return JSONResponse(
            status_code=409,
            content={"detail": str(exc), "error_code": "CONFIG_ALREADY_EXISTS"},
        )

    @app.exception_handler(InvalidConfigError)
    async def invalid_config_handler(
        request: Request, exc: InvalidConfigError
    ) -> JSONResponse:
        return JSONResponse(
            status_code=400,
            content={"detail": str(exc), "error_code": "INVALID_CONFIG"},
        )

    @app.exception_handler(NoActiveVersionError)
    async def no_active_version_handler(
        request: Request, exc: NoActiveVersionError
    ) -> JSONResponse:
        return JSONResponse(
            status_code=404,
            content={"detail": str(exc), "error_code": "NO_ACTIVE_VERSION"},
        )

    @app.exception_handler(UseCaseError)
    async def use_case_error_handler(
        request: Request, exc: UseCaseError
    ) -> JSONResponse:
        logger.error(f"Use case error: {exc}")
        return JSONResponse(
            status_code=400,
            content={"detail": str(exc), "error_code": "USE_CASE_ERROR"},
        )

    @app.exception_handler(RepositoryError)
    async def repository_error_handler(
        request: Request, exc: RepositoryError
    ) -> JSONResponse:
        logger.error(f"Repository error: {exc}")
        return JSONResponse(
            status_code=500,
            content={"detail": "Internal server error", "error_code": "REPOSITORY_ERROR"},
        )

    @app.exception_handler(AdapterError)
    async def adapter_error_handler(
        request: Request, exc: AdapterError
    ) -> JSONResponse:
        logger.error(f"Adapter error: {exc}")
        return JSONResponse(
            status_code=500,
            content={"detail": "Internal server error", "error_code": "ADAPTER_ERROR"},
        )

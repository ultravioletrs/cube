# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from fastapi import FastAPI

from src.drivers.rest.routers import admin, guardrails
from src.drivers.rest.exception_handlers import register_exception_handlers


def create_app() -> FastAPI:
    """
    Create and configure the FastAPI application.

    Returns:
        Configured FastAPI application
    """
    app = FastAPI(
        title="Nemo Guardrails Service",
        description="AI Safety Guardrails API for input validation and output sanitization",
        version="1.0.0",
        openapi_tags=[
            {
                "name": "validation",
                "description": "Input validation and output sanitization endpoints",
            },
            {
                "name": "health",
                "description": "Health check endpoints",
            },
            {
                "name": "admin",
                "description": "Configuration management endpoints",
            },
            {
                "name": "configs",
                "description": "Guardrail configuration CRUD endpoints",
            },
            {
                "name": "versions",
                "description": "Version management endpoints",
            },
        ],
    )

    # Register exception handlers
    register_exception_handlers(app)

    # Include routers
    app.include_router(admin.router)
    app.include_router(guardrails.router)

    return app

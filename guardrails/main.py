# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

import logging
import os
import sys
from contextlib import asynccontextmanager
from typing import Any, Dict, Optional

from fastapi import FastAPI, HTTPException
from nemoguardrails import LLMRails, RailsConfig

# Add src to path for imports
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Import clean architecture components
from src.drivers.rest.app import create_app
from src.drivers.rest.dependencies import init_dependencies, shutdown_dependencies
from src.migrations.migrate import run_migrations

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan handler for database mode."""
    # Run migrations on startup
    try:
        db_host = os.environ.get("UV_GUARDRAILS_DB_HOST", "guardrails-db")
        db_port = int(os.environ.get("UV_GUARDRAILS_DB_PORT", "5432"))
        db_user = os.environ.get("UV_GUARDRAILS_DB_USER", "guardrails")
        db_password = os.environ.get("UV_GUARDRAILS_DB_PASS", "guardrails")
        db_name = os.environ.get("UV_GUARDRAILS_DB_NAME", "guardrails")

        await run_migrations(db_host, db_port, db_user, db_password, db_name)
    except Exception as e:
        logger.warning(f"Migration failed (may already be applied): {e}")

    # Initialize dependencies
    await init_dependencies()

    yield

    # Shutdown dependencies
    await shutdown_dependencies()

app = create_app()
app.router.lifespan_context = lifespan


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

# Check if database mode is enabled
USE_DATABASE = os.environ.get("UV_GUARDRAILS_USE_DATABASE", "false").lower() == "true"

if USE_DATABASE:
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

else:
    from src.drivers.rest.routers.schemas import ChatRequest, HealthResponse

    # Legacy mode: Load from file system
    logger.info("Running in legacy mode (file-based configuration)")

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
        ],
    )

    # Initialize Rails configurations from file system
    rails: Optional[LLMRails] = None
    try:
        logger.info("Loading guardrails configuration from ./rails...")
        config = RailsConfig.from_path("./rails")
        rails = LLMRails(config)
        logger.info("Guardrails configurations loaded successfully")
    except Exception as e:
        logger.error(f"Failed to load guardrails configurations: {e}")
        raise

    @app.post("/guardrails/messages", tags=["chat"])
    async def chat_completion(req: ChatRequest):
        try:
            logger.info(f"Processing chat request with {len(req.messages)} messages")

            # Convert Pydantic models to dicts for nemoguardrails
            messages = [{"role": m.role, "content": m.content} for m in req.messages]

            # Generate response using the chat rail
            res = await rails.generate_async(
                messages=messages,
                options={
                    "log": {
                        "llm_calls": True,
                        "internal_events": True,
                        "colang_history": True,
                        "activated_rails": True,
                        "llm_prompts": True,
                        "print_llm_calls_outputs": True,
                    },
                    "llm": {
                        "model": req.model,
                        "temperature": req.temperature,
                        "max_tokens": req.max_tokens,
                    },
                    "llm_params": {
                        "model": req.model,
                        "temperature": req.temperature,
                        "max_tokens": req.max_tokens,
                    },
                    "output_vars": ["relevant_chunks", "triggered_input_rail", "triggered_output_rail"],
                    "return_context": True,
                    "llm_output": True
                },
            )
            res.log.print_summary()
            response_content = res.response if res.response else ""

            # Construct OpenAI-compatible response
            return {
                "id": "chatcmpl-guardrails",
                "object": "chat.completion",
                "created": 0,
                "model": req.model,
                "choices": [
                    {
                        "index": 0,
                        "message": {"role": "assistant", "content": response_content},
                        "finish_reason": "stop",
                    }
                ],
                "usage": {
                    "prompt_tokens": 0,
                    "completion_tokens": 0,
                    "total_tokens": 0,
                },
            }

        except Exception as e:
            logger.error(f"Chat completion error: {str(e)}")
            raise HTTPException(status_code=500, detail=str(e))

    @app.get("/guardrails/health", response_model=HealthResponse, tags=["health"])
    async def health_check():
        """Health check endpoint for container monitoring."""
        return HealthResponse(status="healthy", runtime_ready=True, version="1.0.0", current_revision=1)

    @app.get("/guardrails", response_model=Dict[str, Any])
    async def root():
        """Root endpoint with service information."""
        return {
            "service": "Nemo Guardrails API",
            "version": "1.0.0",
            "status": "running",
            "mode": "legacy",
            "endpoints": ["/guardrails/messages", "/guardrails/health", "/guardrails/docs"],
        }

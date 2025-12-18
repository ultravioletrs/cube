# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

import logging
from typing import Any, Dict

from fastapi import APIRouter, HTTPException

from src.drivers.rest.dependencies import get_runtime
from src.drivers.rest.routers.schemas import (
    ChatRequest,
    HealthResponse,
)

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/guardrails", tags=["guardrails"])


@router.post("/messages", tags=["chat"])
async def chat_completion(req: ChatRequest) -> Dict[str, Any]:
    runtime = get_runtime()

    if not runtime.is_ready():
        raise HTTPException(
            status_code=503,
            detail="Guardrails runtime not ready. No active configuration loaded.",
        )

    try:
        logger.info(f"Processing chat request with {len(req.messages)} messages")

        # Convert Pydantic models to dicts
        messages = [{"role": m.role, "content": m.content} for m in req.messages]

        # Generate response using runtime
        res = await runtime.generate(
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


@router.get("/health", response_model=HealthResponse, tags=["health"])
async def health_check() -> HealthResponse:
    """Health check endpoint for container monitoring."""
    runtime = get_runtime()
    return HealthResponse(
        status="healthy",
        version="1.0.0",
        runtime_ready=runtime.is_ready(),
        current_revision=runtime.get_current_revision(),
    )


@router.get("/", tags=["health"])
async def root() -> Dict[str, Any]:
    """Root endpoint with service information."""
    runtime = get_runtime()
    return {
        "service": "Nemo Guardrails API",
        "version": "1.0.0",
        "status": "running",
        "runtime_ready": runtime.is_ready(),
        "current_revision": runtime.get_current_revision(),
        "endpoints": [
            "/guardrails/messages",
            "/guardrails/health",
            "/guardrails",
            "/guardrails/configs",
            "/guardrails/versions/{version_id}/activate",
            "/guardrails/reload",
        ],
    }

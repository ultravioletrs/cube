# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

import logging
import re
from typing import Any, Dict

from fastapi import APIRouter, Header, HTTPException, Request

from src.drivers.rest.dependencies import get_runtime
from src.drivers.rest.routers.schemas import (
    ChatRequest,
    HealthResponse,
)

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/guardrails", tags=["guardrails"])


def clean_response(response) -> str:
    if not response:
        return ""

    if isinstance(response, list):
        parts = []
        for item in response:
            if isinstance(item, dict):
                parts.append(item.get("content", str(item)))
            else:
                parts.append(str(item))
        response = " ".join(parts)

    if not isinstance(response, str):
        response = str(response)

    cleaned = response.strip()

    cleaned = re.sub(r'^(bot|I)\s+\w+(\s+\w+)*\s*\n', '', cleaned, flags=re.IGNORECASE)

    if cleaned.startswith('"') and cleaned.endswith('"'):
        cleaned = cleaned[1:-1]

    if cleaned.lower().startswith("bot "):
        cleaned = "I" + cleaned[3:]

    return cleaned.strip()


@router.post("/messages", tags=["chat"])
async def chat_completion(request: Request, req: ChatRequest, authorization: str = Header(None)) -> Dict[str, Any]:
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

        llm_params = {
            "model": req.model,
            "temperature": req.temperature,
            "max_tokens": req.max_tokens,
            "headers": {
                "Authorization": authorization,
                "X-Guardrails-Request": "true"
            }
        }

        logger.debug(f"llm_params prepared for model: {req.model}, auth_present={authorization is not None}")
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
                "llm_params": llm_params,
                "llm": llm_params,
                "output_vars": ["relevant_chunks", "triggered_input_rail", "triggered_output_rail"],
                "return_context": True,
                "llm_output": True
            },
        )

        response_content = res.response if res.response else ""
        response_content = clean_response(response_content)

        return {
            "model": req.model,
            "message": {
                "role": "assistant",
                "content": response_content,
            },
            "done": True,
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

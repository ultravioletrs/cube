# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

import logging
import time
from typing import Any, Dict, List, Optional, Tuple

from fastapi import APIRouter, HTTPException

from src.drivers.rest.dependencies import get_runtime
from src.drivers.rest.routers.schemas import (
    ChatRequest,
    EvaluationResponse,
    HealthResponse,
)

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/guardrails", tags=["guardrails"])

# Keywords that indicate a refusal response
REFUSAL_KEYWORDS = [
    "cannot", "can't", "won't", "will not", "unable to",
    "sorry", "apologize", "not able", "don't engage",
    "not provide", "not assist", "not help", "i'm not able",
    "i cannot", "i can't", "i won't", "i will not",
    "not allowed", "not permitted", "against my guidelines",
    "inappropriate", "harmful", "unethical", "illegal",
]

# Patterns in colang history that indicate refusal/blocking
COLANG_REFUSAL_PATTERNS = [
    "refuse", "decline", "inform invalid", "bot refuse", "bot decline",
    "stop", "blocked", "reject",
]

def clean_guardrails_response(response: Optional[str]) -> Optional[str]:
    if not response:
        return response

    cleaned = response.strip()

    if cleaned.lower().startswith("bot "):
        cleaned = "I" + cleaned[3:]

    return cleaned


def extract_evaluation_result(res, rail_type: str = "input") -> Tuple[str, Optional[str], List[str]]:
    triggered_rails = []

    if res.output_data:
        input_rail = res.output_data.get("triggered_input_rail")
        output_rail = res.output_data.get("triggered_output_rail")
        if input_rail:
            triggered_rails.append(input_rail)
        if output_rail:
            triggered_rails.append(output_rail)

    guardrails_response = None
    if res.response:
        if isinstance(res.response, list) and len(res.response) > 0:
            guardrails_response = res.response[0].get("content")
        elif isinstance(res.response, str):
            guardrails_response = res.response

    guardrails_response = clean_guardrails_response(guardrails_response)

    decision = "ALLOW"
    block_reason = None

    # Signal 1: Check activated rails for stop=True (most reliable)
    if res.log and res.log.activated_rails:
        for rail in res.log.activated_rails:
            if rail.stop:
                decision = "BLOCK"
                block_reason = f"activated_rail_stop: {rail.type if hasattr(rail, 'type') else 'unknown'}"
                logger.debug(f"Block detected via activated rail stop: {rail}")
                break

    if decision == "ALLOW" and res.log and res.log.colang_history:
        colang_lower = res.log.colang_history.lower()
        for pattern in COLANG_REFUSAL_PATTERNS:
            if pattern in colang_lower:
                decision = "BLOCK"
                block_reason = f"colang_pattern: {pattern}"
                logger.debug(f"Block detected via colang pattern: {pattern}")
                break

    if decision == "ALLOW" and triggered_rails:
        decision = "BLOCK"
        block_reason = f"triggered_rail: {triggered_rails[0]}"
        logger.debug(f"Block detected via triggered rail: {triggered_rails}")

    if decision == "ALLOW" and guardrails_response:
        response_lower = guardrails_response.lower()
        for keyword in REFUSAL_KEYWORDS:
            if keyword in response_lower:
                decision = "BLOCK"
                block_reason = f"response_keyword: {keyword}"
                logger.debug(f"Block detected via response keyword: {keyword}")
                break

    if block_reason:
        logger.info(f"Evaluation decision: {decision} (reason: {block_reason})")
    else:
        logger.debug(f"Evaluation decision: {decision}")

    return decision, guardrails_response, triggered_rails



@router.post("/evaluate/input", tags=["evaluation"])
async def evaluate_input(req: ChatRequest) -> EvaluationResponse:
    """Evaluate user input against guardrails."""
    runtime = get_runtime()
    if not runtime.is_ready():
        raise HTTPException(status_code=503, detail="Runtime not ready")

    start_time = time.time()
    logger.info("Starting input evaluation with LLM disabled")

    try:
        messages = [{"role": m.role, "content": m.content} for m in req.messages]

        res = await runtime.generate(
            messages=messages,
            options={
                "rails": {
                    "input": True,
                    "output": False,
                    "dialog": True,
                    "retrieval": False,
                },
                "log": {
                    "llm_calls": True,
                    "internal_events": True,
                    "colang_history": True,
                    "activated_rails": True,
                    "llm_prompts": True,
                    "print_llm_calls_outputs": True,
                },
                "output_vars": ["relevant_chunks", "triggered_input_rail", "triggered_output_rail"],
                "return_context": True,
                "llm_output": True
            },
        )

        duration = (time.time() - start_time) * 1000

        decision, guardrails_response, triggered_rails = extract_evaluation_result(res, rail_type="input")

        return EvaluationResponse(
            decision=decision,
            guardrails_response=guardrails_response,
            evaluation_time_ms=duration,
            triggered_rails=triggered_rails
        )

    except Exception as e:
        logger.error(f"Input evaluation error: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/evaluate/output", tags=["evaluation"])
async def evaluate_output(req: ChatRequest) -> EvaluationResponse:
    runtime = get_runtime()
    if not runtime.is_ready():
        raise HTTPException(status_code=503, detail="Runtime not ready")

    start_time = time.time()

    try:
        messages = [{"role": m.role, "content": m.content} for m in req.messages]

        res = await runtime.generate(
            messages=messages,
            options={
                "rails": {
                    "input": False,
                    "output": True,
                    "dialog": True,
                    "retrieval": False,
                },
                "log": {
                    "llm_calls": True,
                    "internal_events": True,
                    "colang_history": True,
                    "activated_rails": True,
                    "llm_prompts": True,
                    "print_llm_calls_outputs": True,
                },
                "output_vars": ["relevant_chunks", "triggered_input_rail", "triggered_output_rail"],
                "return_context": True,
                "llm_output": True
            },
        )

        duration = (time.time() - start_time) * 1000

        decision, guardrails_response, triggered_rails = extract_evaluation_result(res, rail_type="output")

        return EvaluationResponse(
            decision=decision,
            guardrails_response=guardrails_response,
            evaluation_time_ms=duration,
            triggered_rails=triggered_rails
        )

    except Exception as e:
        logger.error(f"Output evaluation error: {str(e)}")
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

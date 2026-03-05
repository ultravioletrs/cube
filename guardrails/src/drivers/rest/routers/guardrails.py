# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

import logging
import re
import time
from typing import Any, Dict, Optional

from fastapi import APIRouter, Header, HTTPException, Request
from langchain_core.messages import HumanMessage
from langchain_openai import ChatOpenAI

from src.drivers.rest.dependencies import get_runtime
from src.drivers.rest.routers.schemas import (
    ChatRequest,
    HealthResponse,
)

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/guardrails", tags=["guardrails"])

# ---------------------------------------------------------------------------
# Canned responses handled in Python – bypasses NeMo entirely for speed
# and reliability (Colang's `user said` is case-sensitive exact matching).
# ---------------------------------------------------------------------------

_GREETING_PATTERNS = re.compile(
    r"^(hi|hello|hey|howdy|good\s*(morning|afternoon|evening)|"
    r"greetings|what'?s\s*up|sup|yo|hiya|heya)[\s!?.,:;]*$",
    re.IGNORECASE,
)
_GOODBYE_PATTERNS = re.compile(
    r"^(bye|goodbye|good\s*bye|see\s*you(\s*later)?|farewell|take\s*care|"
    r"talk\s*to\s*you\s*later|later|cheers|cya|ttyl|adios|"
    r"good\s*night|have\s*a\s*good\s*(one|day|night))[\s!?.,:;]*$",
    re.IGNORECASE,
)
_CAPABILITIES_PATTERNS = re.compile(
    r"^(what\s*can\s*you\s*do|what\s*are\s*your\s*capabilities|"
    r"how\s*can\s*you\s*help|what\s*can\s*you\s*help\s*with|"
    r"tell\s*me\s*about\s*your\s*features|what\s*do\s*you\s*do|"
    r"who\s*are\s*you|what\s*are\s*you)[\s?!.]*$",
    re.IGNORECASE,
)

_CANNED_RESPONSES = {
    "greeting": "Hello! How can I assist you today?",
    "goodbye": "Goodbye! Feel free to return if you need any assistance.",
    "capabilities": (
        "I can help with a variety of tasks including answering questions, "
        "providing information, and assisting with analysis. I have safety "
        "guardrails in place to ensure our conversation remains helpful "
        "and appropriate."
    ),
}


def _match_canned(text: str) -> Optional[str]:
    """Return a canned response key if *text* matches a known pattern, else None."""
    if _GREETING_PATTERNS.match(text):
        return "greeting"
    if _GOODBYE_PATTERNS.match(text):
        return "goodbye"
    if _CAPABILITIES_PATTERNS.match(text):
        return "capabilities"
    return None


# ---------------------------------------------------------------------------
# Direct LLM fallback – used when NeMo returns an empty response
# ---------------------------------------------------------------------------

async def _direct_llm_fallback(
    user_text: str, model: str, authorization: Optional[str]
) -> str:
    """Call the LLM directly, bypassing NeMo, as a last-resort fallback."""
    try:
        headers = {"X-Guardrails-Request": "true"}
        if authorization:
            headers["Authorization"] = authorization

        llm = ChatOpenAI(
            model=model,
            base_url="http://cube-proxy:8900/v1",
            api_key="EMPTY",
            default_headers=headers,
            temperature=0.7,
            max_tokens=1024,
            timeout=60,
        )
        result = await llm.ainvoke([HumanMessage(content=user_text)])
        return result.content.strip() if result and result.content else ""
    except Exception as e:
        logger.error(f"Direct LLM fallback failed: {e}")
        return ""


def clean_response(response) -> str:
    """Extract a clean assistant message string from a NeMo GenerationResponse.

    Handles the multiple shapes .response can take in Colang 2.x:
      - str
      - list[dict] with 'content' keys  (most common from single_call mode)
      - list[str]
      - None / empty
    """
    if not response:
        return ""

    # --- list of message dicts (Colang 2.x single-call mode) ---------------
    if isinstance(response, list):
        parts: list[str] = []
        for item in response:
            if isinstance(item, dict):
                content = item.get("content", "")
                if content:
                    parts.append(str(content))
            elif isinstance(item, str) and item.strip():
                parts.append(item)
        response = " ".join(parts) if parts else ""

    if not isinstance(response, str):
        response = str(response)

    cleaned = response.strip()

    # Remove leading "bot say" / "bot inform" prefixes that Colang can leak
    cleaned = re.sub(
        r'^bot\s+(say|inform|respond|express|clarify|suggest)\s+',
        '',
        cleaned,
        flags=re.IGNORECASE,
    )

    # Strip surrounding quotes
    if len(cleaned) >= 2 and cleaned[0] == '"' and cleaned[-1] == '"':
        cleaned = cleaned[1:-1]

    return cleaned.strip()


def extract_guardrails_detections(res: Any, response_content: str, original_messages: list) -> Dict[str, Any]:
    """
    Extract guardrails detection information from NeMo Guardrails response.
    
    This function analyzes the response context, triggered rails, and response content
    to identify what guardrails were activated and what violations were detected.
    
    Returns a dictionary with detection information for audit logging.
    """
    detections = {
        "processed": True,
        "decision": "ALLOW",
        "triggered_input_rails": [],
        "triggered_output_rails": [],
        "violations": [],
    }
    
    # Try to extract context from the response
    context = {}
    if hasattr(res, 'output_data') and res.output_data:
        context = res.output_data
    elif hasattr(res, 'context') and res.context:
        context = res.context
    
    # Extract triggered rails from context
    if "triggered_input_rail" in context:
        triggered_input = context["triggered_input_rail"]
        if triggered_input:
            if isinstance(triggered_input, list):
                detections["triggered_input_rails"] = triggered_input
            else:
                detections["triggered_input_rails"] = [str(triggered_input)]
    
    if "triggered_output_rail" in context:
        triggered_output = context["triggered_output_rail"]
        if triggered_output:
            if isinstance(triggered_output, list):
                detections["triggered_output_rails"] = triggered_output
            else:
                detections["triggered_output_rails"] = [str(triggered_output)]
    
    # Analyze response content for known guardrail patterns
    response_lower = response_content.lower() if response_content else ""
    
    # Check for jailbreak/prompt injection detection
    jailbreak_patterns = [
        "bypass my guidelines",
        "can't help with requests that try to bypass",
        "detected an attempt to manipulate",
        "refuse jailbreak",
        "refuse prompt_injection",
    ]
    for pattern in jailbreak_patterns:
        if pattern in response_lower:
            detections["violations"].append({
                "type": "jailbreak_attempt",
                "category": "input_validation",
                "severity": "high",
                "description": "Detected attempt to bypass safety guidelines",
                "action": "blocked",
            })
            detections["decision"] = "BLOCK"
            break
    
    # Check for prompt injection detection
    injection_patterns = [
        "prompt injection",
        "manipulate my responses",
        "ignore previous instructions",
    ]
    for pattern in injection_patterns:
        if pattern in response_lower:
            detections["violations"].append({
                "type": "prompt_injection",
                "category": "input_validation", 
                "severity": "high",
                "description": "Detected prompt injection attempt",
                "action": "blocked",
            })
            detections["decision"] = "BLOCK"
            break
    
    # Check for toxic content detection
    toxic_patterns = [
        "cannot engage with that type of language",
        "refuse toxic_content",
        "let's have a respectful conversation",
    ]
    for pattern in toxic_patterns:
        if pattern in response_lower:
            detections["violations"].append({
                "type": "toxic_content",
                "category": "input_validation",
                "severity": "medium",
                "description": "Detected toxic or inappropriate language",
                "action": "blocked",
            })
            detections["decision"] = "BLOCK"
            break
    
    # Check for off-topic detection
    offtopic_patterns = [
        "cannot provide information about that topic",
        "restricted content guidelines",
        "refuse restricted_topic",
    ]
    for pattern in offtopic_patterns:
        if pattern in response_lower:
            detections["violations"].append({
                "type": "off_topic",
                "category": "input_validation",
                "severity": "low",
                "description": "Request falls outside acceptable topic range",
                "action": "blocked",
            })
            detections["decision"] = "BLOCK"
            break
    
    # Check for PII/sensitive data masking
    # This requires comparing input messages with output to detect masking
    pii_patterns = [
        r'\[MASKED\]',
        r'\*{4,}',  # Four or more asterisks (masking pattern)
        r'\[REDACTED\]',
        r'\[PII\]',
    ]
    for pattern in pii_patterns:
        if re.search(pattern, response_content, re.IGNORECASE):
            detections["violations"].append({
                "type": "pii_detected",
                "category": "output_validation",
                "severity": "medium",
                "description": "Sensitive data was detected and masked",
                "action": "masked",
            })
            # PII masking is MODIFY, not BLOCK
            if detections["decision"] == "ALLOW":
                detections["decision"] = "MODIFY"
            break
    
    # Check for hallucination warnings
    hallucination_patterns = [
        "revise my response",
        "more careful and accurate response",
        "cautious about the accuracy",
        "potential_hallucination",
        "factuality_concern",
    ]
    for pattern in hallucination_patterns:
        if pattern in response_lower:
            detections["violations"].append({
                "type": "hallucination_risk",
                "category": "output_validation",
                "severity": "low",
                "description": "Response may contain inaccurate information",
                "action": "warning",
            })
            break
    
    # Check for invalid/empty message handling
    invalid_patterns = [
        "didn't receive a valid message",
        "invalid_message",
    ]
    for pattern in invalid_patterns:
        if pattern in response_lower:
            detections["violations"].append({
                "type": "invalid_input",
                "category": "input_validation",
                "severity": "low",
                "description": "Invalid or empty message received",
                "action": "rejected",
            })
            detections["decision"] = "BLOCK"
            break
    
    # Also check triggered rails for detection info
    all_rails = detections["triggered_input_rails"] + detections["triggered_output_rails"]
    for rail in all_rails:
        rail_lower = rail.lower() if isinstance(rail, str) else ""
        
        if "jailbreak" in rail_lower and not any(v["type"] == "jailbreak_attempt" for v in detections["violations"]):
            detections["violations"].append({
                "type": "jailbreak_attempt",
                "category": "input_validation",
                "severity": "high",
                "description": f"Triggered by rail: {rail}",
                "rail": rail,
                "action": "blocked",
            })
            detections["decision"] = "BLOCK"
        
        if "injection" in rail_lower and not any(v["type"] == "prompt_injection" for v in detections["violations"]):
            detections["violations"].append({
                "type": "prompt_injection",
                "category": "input_validation",
                "severity": "high",
                "description": f"Triggered by rail: {rail}",
                "rail": rail,
                "action": "blocked",
            })
            detections["decision"] = "BLOCK"
        
        if "toxic" in rail_lower and not any(v["type"] == "toxic_content" for v in detections["violations"]):
            detections["violations"].append({
                "type": "toxic_content",
                "category": "input_validation",
                "severity": "medium",
                "description": f"Triggered by rail: {rail}",
                "rail": rail,
                "action": "blocked",
            })
            detections["decision"] = "BLOCK"
        
        if ("pii" in rail_lower or "sensitive" in rail_lower or "mask" in rail_lower) and \
           not any(v["type"] == "pii_detected" for v in detections["violations"]):
            detections["violations"].append({
                "type": "pii_detected", 
                "category": "output_validation",
                "severity": "medium",
                "description": f"Triggered by rail: {rail}",
                "rail": rail,
                "action": "masked",
            })
            if detections["decision"] == "ALLOW":
                detections["decision"] = "MODIFY"
    
    logger.info(f"Guardrails detections: decision={detections['decision']}, "
                f"violations={len(detections['violations'])}, "
                f"input_rails={detections['triggered_input_rails']}, "
                f"output_rails={detections['triggered_output_rails']}")
    
    return detections


@router.post("/messages", tags=["chat"])
async def chat_completion(request: Request, req: ChatRequest, authorization: str = Header(None)) -> Dict[str, Any]:
    start_time = time.time()

    runtime = get_runtime()

    if not runtime.is_ready():
        raise HTTPException(
            status_code=503,
            detail="Guardrails runtime not ready. No active configuration loaded.",
        )

    try:
        logger.info(f"Processing chat request with {len(req.messages)} messages")

        # Convert Pydantic models to dicts
        all_messages = [{"role": m.role, "content": m.content} for m in req.messages]

        # Find the last user message — this is the current turn.
        last_user_msg = None
        for m in reversed(all_messages):
            if m["role"] == "user":
                last_user_msg = m
                break

        if not last_user_msg:
            last_user_msg = all_messages[-1] if all_messages else {"role": "user", "content": ""}

        # Handle empty / whitespace-only messages in Python before NeMo.
        user_text = (last_user_msg.get("content") or "").strip()
        if not user_text or user_text in ("...", "???", "null", "None", "undefined",
                                           "[null]", "[undefined]", "<empty>", "<null>"):
            logger.info("Empty or invalid user message — returning canned response")
            return {
                "model": req.model,
                "message": {
                    "role": "assistant",
                    "content": "I didn't receive a valid message. Please try again with a clear question or request.",
                },
                "done": True,
                "usage": {"prompt_tokens": 0, "completion_tokens": 0, "total_tokens": 0},
                "guardrails": {
                    "processed": True,
                    "decision": "BLOCK",
                    "triggered_input_rails": ["empty_message"],
                    "triggered_output_rails": [],
                    "violations": [{"type": "invalid_input", "category": "input_validation",
                                    "severity": "low", "description": "Empty or invalid message",
                                    "action": "rejected"}],
                    "latency_ms": 0,
                },
            }

        # ---- Canned responses (greetings, goodbye, capabilities) -----------
        # Handled in Python for reliability — Colang's `user said` is case-
        # sensitive exact matching and fails on "Hello" vs "hello".
        canned_key = _match_canned(user_text)
        if canned_key:
            latency = (time.time() - start_time) * 1000
            logger.info(f"Canned response for '{canned_key}' in {latency:.0f}ms")
            return {
                "model": req.model,
                "message": {
                    "role": "assistant",
                    "content": _CANNED_RESPONSES[canned_key],
                },
                "done": True,
                "usage": {"prompt_tokens": max(1, len(user_text) // 4),
                          "completion_tokens": max(1, len(_CANNED_RESPONSES[canned_key]) // 4),
                          "total_tokens": max(2, (len(user_text) + len(_CANNED_RESPONSES[canned_key])) // 4)},
                "guardrails": {
                    "processed": True,
                    "decision": "ALLOW",
                    "triggered_input_rails": [],
                    "triggered_output_rails": [],
                    "violations": [],
                    "latency_ms": latency,
                },
            }

        # ---- NeMo Guardrails pipeline ------------------------------------
        # Lowercase the user message so Colang's case-sensitive `user said`
        # patterns (in guard flows) match regardless of the original casing.
        lowered_msg = {"role": "user", "content": user_text.lower()}
        messages = [lowered_msg]

        llm_headers = {"X-Guardrails-Request": "true"}
        if authorization:
            llm_headers["Authorization"] = authorization

        llm_params = {
            "model": req.model,
            "temperature": req.temperature,
            "max_tokens": req.max_tokens,
            "headers": llm_headers,
        }

        logger.debug(f"llm_params prepared for model: {req.model}, auth_present={authorization is not None}")

        response_content = ""
        res = None  # NeMo response object for detection extraction
        nemo_failed = False

        try:
            res = await runtime.generate(
                messages=messages,
                options={
                    "llm_params": llm_params,
                    "llm": llm_params,
                },
            )

            # --- robust response extraction --------------------------------
            raw = res.response if hasattr(res, "response") else None

            if not raw and hasattr(res, "response") and isinstance(res.response, list):
                for msg in reversed(res.response):
                    if isinstance(msg, dict) and msg.get("role") == "assistant" and msg.get("content"):
                        raw = msg["content"]
                        break

            response_content = clean_response(raw)
        except Exception as nemo_err:
            logger.warning(f"NeMo pipeline failed: {nemo_err} — will try direct LLM fallback")
            nemo_failed = True

        # ---- Fallback: if NeMo returned empty or threw, call LLM directly --
        if not response_content:
            logger.warning("NeMo returned empty response — invoking direct LLM fallback")
            response_content = await _direct_llm_fallback(user_text, req.model, authorization)
            if not response_content:
                response_content = "I'm sorry, I wasn't able to generate a response. Please try again."

        # Calculate guardrails processing latency
        guardrails_latency_ms = (time.time() - start_time) * 1000

        # Extract guardrails detection information from response context
        if res is not None:
            guardrails_info = extract_guardrails_detections(res, response_content, all_messages)
        else:
            guardrails_info = {
                "processed": nemo_failed is False,
                "decision": "ALLOW",
                "triggered_input_rails": [],
                "triggered_output_rails": [],
                "violations": [],
            }
        guardrails_info["latency_ms"] = guardrails_latency_ms

        # Calculate token usage (rough estimate: ~4 chars per token)
        input_chars = sum(len(m.content) for m in req.messages)
        output_chars = len(response_content)
        prompt_tokens = max(1, input_chars // 4)
        completion_tokens = max(1, output_chars // 4)

        return {
            "model": req.model,
            "message": {
                "role": "assistant",
                "content": response_content,
            },
            "done": True,
            "usage": {
                "prompt_tokens": prompt_tokens,
                "completion_tokens": completion_tokens,
                "total_tokens": prompt_tokens + completion_tokens,
            },
            "guardrails": guardrails_info,
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

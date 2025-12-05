# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from nemoguardrails import LLMRails, RailsConfig
from typing import Dict, Any, Optional
import os
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# FastAPI app with metadata
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
        }
    ]
)

# Initialize Rails configurations
try:
    logger.info("Loading guardrails configuration from ./rails...")
    config = RailsConfig.from_path("./rails")
    rails = LLMRails(config)
    
    logger.info("Guardrails configurations loaded successfully")
except Exception as e:
    logger.error(f"Failed to load guardrails configurations: {e}")
    raise


class ChatMessage(BaseModel):
    role: str
    content: str

class ChatRequest(BaseModel):
    messages: list[ChatMessage]
    model: Optional[str] = "tinyllama"
    temperature: Optional[float] = 0.1
    max_tokens: Optional[int] = 150



class HealthResponse(BaseModel):
    status: str
    version: str = "1.0.0"



@app.post("/v1/chat/completions", tags=["chat"])
async def chat_completion(req: ChatRequest):
    try:
        logger.info(f"Processing chat request with {len(req.messages)} messages")
        
        # Convert Pydantic models to dicts for nemoguardrails
        messages = [{"role": m.role, "content": m.content} for m in req.messages]
        
        # Generate response using the chat rail
        # This handles Input Rail -> LLM -> Output Rail automatically
        res = await rails.generate_async(messages=messages)
        
        response_content = res.response if res.response else ""
        
        # Construct OpenAI-compatible response
        return {
            "id": "chatcmpl-guardrails",
            "object": "chat.completion",
            "created": 0,
            "model": req.model,
            "choices": [{
                "index": 0,
                "message": {
                    "role": "assistant",
                    "content": response_content
                },
                "finish_reason": "stop"
            }],
            "usage": {
                "prompt_tokens": 0,
                "completion_tokens": 0,
                "total_tokens": 0
            }
        }

    except Exception as e:
        logger.error(f"Chat completion error: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/health", response_model=HealthResponse, tags=["health"])
async def health_check():
    """Health check endpoint for container monitoring."""
    return HealthResponse(status="healthy")

@app.get("/", response_model=Dict[str, Any])
async def root():
    """Root endpoint with service information."""
    return {
        "service": "Nemo Guardrails API",
        "version": "1.0.0",
        "status": "running",
        "endpoints": [
            "/v1/chat/completions",
            "/health",
            "/docs"
        ]
    }

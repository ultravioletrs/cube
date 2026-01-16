# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from datetime import datetime
from typing import List, Optional
from uuid import UUID

from pydantic import BaseModel, Field


# ==================== Config Schemas ====================


class ConfigCreate(BaseModel):
    """Schema for creating a new guardrail configuration."""

    name: str = Field(..., min_length=1, max_length=255, description="Unique name")
    description: Optional[str] = Field(None, description="Optional description")
    config_yaml: str = Field(..., min_length=1, description="YAML content for config.yml")
    prompts_yaml: str = Field("", description="YAML content for prompts.yml")
    colang: str = Field("", description="Colang content for rails")


class ConfigUpdate(BaseModel):
    """Schema for updating a guardrail configuration."""

    name: Optional[str] = Field(None, min_length=1, max_length=255)
    description: Optional[str] = None
    config_yaml: Optional[str] = Field(None, min_length=1)
    prompts_yaml: Optional[str] = None
    colang: Optional[str] = None


class ConfigResponse(BaseModel):
    """Schema for guardrail configuration response."""

    id: UUID
    name: str
    description: Optional[str]
    config_yaml: str
    prompts_yaml: str
    colang: str
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True


class ConfigListResponse(BaseModel):
    """Schema for paginated config list response."""

    configs: List[ConfigResponse]
    total: int
    offset: int
    limit: int


# ==================== Version Schemas ====================


class VersionCreate(BaseModel):
    """Schema for creating a new version."""

    name: str = Field(..., min_length=1, max_length=255, description="Version name")
    description: Optional[str] = Field(None, description="Optional description")


class VersionResponse(BaseModel):
    """Schema for version response."""

    id: UUID
    config_id: UUID
    name: str
    revision: int
    is_active: bool
    description: Optional[str]
    created_at: datetime

    class Config:
        from_attributes = True


class VersionListResponse(BaseModel):
    """Schema for paginated version list response."""

    versions: List[VersionResponse]
    total: int
    offset: int
    limit: int


# ==================== Activation Schemas ====================


class ActivateResponse(BaseModel):
    """Schema for activation response."""

    status: str = "activated"
    version_id: UUID
    revision: int


class ReloadResponse(BaseModel):
    """Schema for reload response."""

    status: str
    revision: int
    message: str


# ==================== Chat Schemas ====================


class ChatMessage(BaseModel):
    """Schema for a chat message."""

    role: str
    content: str


class ChatRequest(BaseModel):
    """Schema for chat completion request."""

    messages: List[ChatMessage]
    model: Optional[str] = "tinyllama"
    temperature: Optional[float] = 0.1
    max_tokens: Optional[int] = 150


class ChatChoice(BaseModel):
    """Schema for chat completion choice."""

    index: int
    message: ChatMessage
    finish_reason: str


class ChatUsage(BaseModel):
    """Schema for token usage."""

    prompt_tokens: int
    completion_tokens: int
    total_tokens: int


class ChatCompletionResponse(BaseModel):
    """Schema for chat completion response."""

    id: str
    object: str = "chat.completion"
    created: int
    model: str
    choices: List[ChatChoice]
    usage: ChatUsage


# ==================== Evaluation Schemas ====================


class EvaluationResponse(BaseModel):
    """Schema for guardrails evaluation response."""

    decision: str = Field(..., description="ALLOW, BLOCK, or MODIFY")
    reason: Optional[str] = None
    modified_messages: Optional[List[ChatMessage]] = None
    guardrails_response: Optional[str] = None
    evaluation_time_ms: float
    triggered_rails: List[str] = []


# ==================== Health Schemas ====================


class HealthResponse(BaseModel):
    """Schema for health check response."""

    status: str
    version: str = "1.0.0"
    runtime_ready: bool
    current_revision: int

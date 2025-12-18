# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from dataclasses import dataclass
from datetime import datetime
from typing import Optional
from uuid import UUID


@dataclass(frozen=True)
class GuardrailConfig:
    """
    Represents a guardrail configuration entity.
    Contains the YAML content for config, prompts, and colang files.
    """

    id: UUID
    name: str
    description: Optional[str]
    config_yaml: str
    prompts_yaml: str
    colang: str
    created_at: datetime
    updated_at: datetime

    def __post_init__(self):
        if not self.name:
            raise ValueError("Config name cannot be empty")
        if not self.config_yaml:
            raise ValueError("config_yaml cannot be empty")

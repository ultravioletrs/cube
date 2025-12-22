# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from dataclasses import dataclass
from datetime import datetime
from typing import Optional
from uuid import UUID


@dataclass(frozen=True)
class GuardrailVersion:
    """
    Represents a versioned guardrail configuration.
    Tracks revision history and active state.
    """

    id: UUID
    config_id: UUID
    name: str
    revision: int
    is_active: bool
    created_at: datetime
    description: Optional[str] = None

    def __post_init__(self):
        if self.revision < 1:
            raise ValueError("Revision must be >= 1")
        if not self.name:
            raise ValueError("Version name cannot be empty")

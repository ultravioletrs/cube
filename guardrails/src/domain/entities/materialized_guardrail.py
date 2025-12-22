# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from dataclasses import dataclass
from uuid import UUID


@dataclass(frozen=True)
class MaterializedGuardrail:
    """
    Represents a materialized (denormalized) guardrail ready for runtime use.
    Contains all content needed to initialize NeMo Guardrails.
    """

    version_id: UUID
    config_yaml: str
    prompts_yaml: str
    colang: str
    revision: int

    def __post_init__(self):
        if not self.config_yaml:
            raise ValueError("config_yaml cannot be empty")
        if self.revision < 1:
            raise ValueError("Revision must be >= 1")

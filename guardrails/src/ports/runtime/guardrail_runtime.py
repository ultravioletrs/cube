# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from abc import ABC, abstractmethod
from typing import Any, Dict, List, Optional

from src.domain.entities import MaterializedGuardrail


class GuardrailRuntime(ABC):
    """
    Port (interface) for guardrail runtime operations.
    Handles atomic swap of configurations and message generation.
    """

    @abstractmethod
    async def swap(self, materialized: MaterializedGuardrail) -> None:
        """
        Atomically swap the current guardrail configuration.
        Thread-safe with lock protection.
        Uses revision check to prevent stale updates.
        """
        pass

    @abstractmethod
    async def generate(
        self,
        messages: List[Dict[str, str]],
        options: Optional[Dict[str, Any]] = None,
    ) -> Any:
        """
        Generate a response using the current guardrail configuration.
        """
        pass

    @abstractmethod
    def get_current_revision(self) -> int:
        """Get the current revision number of the loaded configuration."""
        pass

    @abstractmethod
    def is_ready(self) -> bool:
        """Check if the runtime has a loaded configuration."""
        pass

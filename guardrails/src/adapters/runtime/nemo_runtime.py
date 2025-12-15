# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

import asyncio
import logging
from typing import Any, Dict, List, Optional

import yaml
from nemoguardrails import LLMRails, RailsConfig

from src.adapters.exceptions import ConfigLoadError
from src.domain.entities import MaterializedGuardrail
from src.ports.runtime import GuardrailRuntime

logger = logging.getLogger(__name__)


class NemoRuntime(GuardrailRuntime):
    """
    NeMo Guardrails runtime adapter.
    Handles atomic swap of configurations and thread-safe access.
    Uses RailsConfig.from_content() to load configs directly from memory.
    """

    def __init__(self):
        self._rails: Optional[LLMRails] = None
        self._revision: int = -1
        self._lock = asyncio.Lock()

    async def swap(self, materialized: MaterializedGuardrail) -> None:
        """
        Atomically swap the current guardrail configuration.
        Uses double-checked locking pattern with revision verification.
        Loads config directly from content without temp files.
        """
        # Quick check without lock
        if materialized.revision <= self._revision:
            logger.debug(
                f"Skipping swap: revision {materialized.revision} <= current {self._revision}"
            )
            return

        async with self._lock:
            # Double-check after acquiring lock
            if materialized.revision <= self._revision:
                return

            logger.info(
                f"Swapping guardrail configuration to revision {materialized.revision}"
            )

            try:
                # Parse config YAML to dict and merge prompts into it
                config_dict = yaml.safe_load(materialized.config_yaml) or {}

                # Merge prompts into the config dict if provided
                # Prompts are typically under the "prompts" key in the config
                if materialized.prompts_yaml:
                    prompts_dict = yaml.safe_load(materialized.prompts_yaml) or {}
                    if prompts_dict:
                        # Merge prompts into config
                        for key, value in prompts_dict.items():
                            if key in config_dict and isinstance(config_dict[key], list):
                                # Extend lists (e.g., prompts list)
                                config_dict[key].extend(value)
                            else:
                                config_dict[key] = value

                # Load configuration from content (no temp files needed)
                rails_config = RailsConfig.from_content(
                    yaml_content=materialized.config_yaml,
                    colang_content=materialized.colang if materialized.colang else None,
                    config=config_dict,
                )

                new_rails = LLMRails(rails_config)

                # Atomic swap
                self._rails = new_rails
                self._revision = materialized.revision

                logger.info(
                    f"Successfully loaded guardrail configuration revision {materialized.revision}"
                )

            except Exception as e:
                logger.error(f"Failed to load guardrail configuration: {e}")
                raise ConfigLoadError(f"Failed to load configuration: {e}")

    async def generate(
        self,
        messages: List[Dict[str, str]],
        options: Optional[Dict[str, Any]] = None,
    ) -> Any:
        """
        Generate a response using the current guardrail configuration.
        """
        if not self._rails:
            raise ConfigLoadError("No guardrail configuration loaded")

        return await self._rails.generate_async(
            messages=messages,
            options=options or {},
        )

    def get_current_revision(self) -> int:
        """Get the current revision number of the loaded configuration."""
        return self._revision

    def is_ready(self) -> bool:
        """Check if the runtime has a loaded configuration."""
        return self._rails is not None

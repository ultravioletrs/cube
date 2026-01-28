# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

import asyncio
import logging
from typing import Any, Dict, List, Optional

import yaml
from nemoguardrails import LLMRails, RailsConfig
from nemoguardrails.llm.providers import register_llm_provider

from src.adapters.exceptions import ConfigLoadError
from src.adapters.llm.extended_ollama import ExtendedOllama
from src.domain.entities import MaterializedGuardrail
from src.ports.runtime import GuardrailRuntime

logger = logging.getLogger(__name__)

register_llm_provider("CubeLLM", ExtendedOllama)
logger.info("CubeLLM LLM provider registered successfully")


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
                # Parse config YAML to dict
                config_dict = yaml.safe_load(materialized.config_yaml) or {}

                logger.info(f"Config YAML loaded, keys: {list(config_dict.keys())}")
                logger.info(
                    f"Prompts YAML provided: {bool(materialized.prompts_yaml)}, "
                    f"length: {len(materialized.prompts_yaml) if materialized.prompts_yaml else 0}"
                )

                # Merge prompts into the config dict if provided
                # RailsConfig.from_content() requires all YAML content merged into
                # a single yaml_content string, so we merge prompts into config_dict
                # and regenerate the combined yaml_content
                if materialized.prompts_yaml:
                    prompts_dict = yaml.safe_load(materialized.prompts_yaml) or {}
                    logger.info(f"Prompts dict keys: {list(prompts_dict.keys()) if prompts_dict else 'None'}")
                    if prompts_dict:
                        # Merge prompts into config dict
                        for key, value in prompts_dict.items():
                            if key in config_dict and isinstance(config_dict[key], list):
                                # Extend lists (e.g., prompts list)
                                config_dict[key].extend(value)
                            else:
                                config_dict[key] = value
                        logger.info(f"Merged prompts, config now has keys: {list(config_dict.keys())}")
                        if "prompts" in config_dict:
                            prompt_tasks = [p.get("task") for p in config_dict.get("prompts", [])]
                            logger.info(f"Prompt tasks in merged config: {prompt_tasks}")
                else:
                    logger.warning("No prompts_yaml provided in materialized guardrail!")
                    # Check if the config requires prompts (e.g., self check flows)
                    rails_config = config_dict.get("rails", {})
                    input_flows = rails_config.get("input", {}).get("flows", [])
                    output_flows = rails_config.get("output", {}).get("flows", [])
                    all_flows = input_flows + output_flows

                    requires_prompts = any(
                        "self check" in flow.lower() for flow in all_flows
                    )
                    if requires_prompts:
                        logger.error(
                            "Configuration uses self-check flows but no prompts_yaml provided! "
                            "Please include prompts_yaml with self_check_input and/or "
                            "self_check_output templates when creating the config."
                        )

                # Convert merged config dict back to YAML for from_content()
                merged_yaml_content = yaml.dump(config_dict, default_flow_style=False)

                logger.debug(f"Merged YAML content:\n{merged_yaml_content}")

                # Load configuration from content (no temp files needed)
                # Pass the merged yaml_content containing both config and prompts
                rails_config = RailsConfig.from_content(
                    yaml_content=merged_yaml_content,
                    colang_content=materialized.colang if materialized.colang else None,
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

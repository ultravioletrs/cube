# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from typing import Optional
from uuid import UUID

from src.domain.entities import GuardrailConfig
from src.ports.repositories import GuardrailRepository
from src.use_cases.exceptions import ConfigNotFoundError


class GetConfig:
    """
    Use case for retrieving a guardrail configuration.
    """

    def __init__(self, repo: GuardrailRepository):
        self.repo = repo

    async def execute(self, config_id: UUID) -> GuardrailConfig:
        """
        Get a guardrail configuration by ID.

        Args:
            config_id: UUID of the configuration

        Returns:
            The GuardrailConfig entity

        Raises:
            ConfigNotFoundError: If the config is not found
        """
        config = await self.repo.get_config(config_id)
        if not config:
            raise ConfigNotFoundError(str(config_id))
        return config

    async def by_name(self, name: str) -> Optional[GuardrailConfig]:
        """
        Get a guardrail configuration by name.

        Args:
            name: Name of the configuration

        Returns:
            The GuardrailConfig entity or None if not found
        """
        return await self.repo.get_config_by_name(name)

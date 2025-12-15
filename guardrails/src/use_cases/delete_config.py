# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from uuid import UUID

from src.ports.repositories import GuardrailRepository
from src.use_cases.exceptions import ConfigNotFoundError


class DeleteConfig:
    """
    Use case for deleting a guardrail configuration.
    """

    def __init__(self, repo: GuardrailRepository):
        self.repo = repo

    async def execute(self, config_id: UUID) -> bool:
        """
        Delete a guardrail configuration by ID.

        Args:
            config_id: UUID of the configuration to delete

        Returns:
            True if deletion was successful

        Raises:
            ConfigNotFoundError: If the config is not found
        """
        # Verify config exists
        existing = await self.repo.get_config(config_id)
        if not existing:
            raise ConfigNotFoundError(str(config_id))

        return await self.repo.delete_config(config_id)

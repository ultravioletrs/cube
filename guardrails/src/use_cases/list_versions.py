# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from typing import List
from uuid import UUID

from src.domain.entities import GuardrailVersion
from src.ports.repositories import GuardrailRepository


class ListVersions:
    """
    Use case for listing guardrail versions with pagination.
    """

    def __init__(self, repo: GuardrailRepository):
        self.repo = repo

    async def execute(
        self, config_id: UUID, offset: int = 0, limit: int = 100
    ) -> List[GuardrailVersion]:
        """
        List all versions for a configuration with pagination.

        Args:
            config_id: Configuration ID to list versions for
            offset: Number of records to skip
            limit: Maximum number of records to return

        Returns:
            List of GuardrailVersion entities
        """
        return await self.repo.list_versions(config_id, offset=offset, limit=limit)

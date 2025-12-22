# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from dataclasses import dataclass
from typing import List
from uuid import UUID

from src.domain.entities import GuardrailVersion
from src.ports.repositories import GuardrailRepository


@dataclass
class ListVersionsResult:
    """Result container for paginated version listing."""

    versions: List[GuardrailVersion]
    total: int
    offset: int
    limit: int


class ListVersions:
    """
    Use case for listing guardrail versions with pagination.
    """

    def __init__(self, repo: GuardrailRepository):
        self.repo = repo

    async def execute(
        self, config_id: UUID, offset: int = 0, limit: int = 100
    ) -> ListVersionsResult:
        """
        List all versions for a configuration with pagination.

        Args:
            config_id: Configuration ID to list versions for
            offset: Number of records to skip
            limit: Maximum number of records to return

        Returns:
            ListVersionsResult containing versions and pagination metadata
        """
        versions = await self.repo.list_versions(config_id, offset=offset, limit=limit)
        total = await self.repo.count_versions(config_id)
        return ListVersionsResult(
            versions=versions,
            total=total,
            offset=offset,
            limit=limit,
        )

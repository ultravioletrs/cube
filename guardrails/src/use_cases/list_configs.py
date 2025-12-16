# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from dataclasses import dataclass
from typing import List
from uuid import UUID

from src.domain.entities import GuardrailConfig
from src.ports.repositories import GuardrailRepository


@dataclass
class ListConfigsResult:
    """Result container for paginated config listing."""

    configs: List[GuardrailConfig]
    total: int
    offset: int
    limit: int


class ListConfigs:
    """
    Use case for listing guardrail configurations with pagination.
    """

    def __init__(self, repo: GuardrailRepository):
        self.repo = repo

    async def execute(
        self, domain_id: UUID, offset: int = 0, limit: int = 100
    ) -> List[GuardrailConfig]:
        """
        List all guardrail configurations for a domain with pagination.

        Args:
            domain_id: Domain ID to filter by
            offset: Number of records to skip
            limit: Maximum number of records to return

        Returns:
            List of GuardrailConfig entities
        """
        return await self.repo.list_configs(domain_id=domain_id, offset=offset, limit=limit)

# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from dataclasses import dataclass
from typing import List

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
        self, offset: int = 0, limit: int = 100
    ) -> List[GuardrailConfig]:
        """
        List all guardrail configurations with pagination.

        Args:
            offset: Number of records to skip
            limit: Maximum number of records to return

        Returns:
            List of GuardrailConfig entities
        """
        return await self.repo.list_configs(offset=offset, limit=limit)

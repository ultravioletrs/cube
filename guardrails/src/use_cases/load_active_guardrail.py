# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from src.domain.entities import MaterializedGuardrail
from src.ports.repositories import GuardrailRepository
from src.use_cases.exceptions import NoActiveVersionError


class LoadActiveGuardrail:
    """
    Use case for loading the active guardrail configuration.
    Returns the materialized (denormalized) guardrail ready for runtime use.
    """

    def __init__(self, repo: GuardrailRepository):
        self.repo = repo

    async def execute(self) -> MaterializedGuardrail:
        """
        Load the active materialized guardrail.

        Returns:
            The MaterializedGuardrail entity ready for runtime use

        Raises:
            NoActiveVersionError: If no active version is configured
        """
        materialized = await self.repo.fetch_active_materialized()
        if not materialized:
            raise NoActiveVersionError()
        return materialized

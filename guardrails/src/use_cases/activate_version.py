# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from uuid import UUID

from src.domain.entities import GuardrailVersion
from src.ports.repositories import GuardrailRepository
from src.use_cases.exceptions import VersionNotFoundError


class ActivateVersion:
    """
    Use case for activating a guardrail version.
    Ensures only one version is active at a time.
    """

    def __init__(self, repo: GuardrailRepository):
        self.repo = repo

    async def execute(self, version_id: UUID) -> GuardrailVersion:
        """
        Activate a guardrail version.

        This operation:
        1. Deactivates all currently active versions
        2. Activates the specified version
        3. Returns the updated version

        Args:
            version_id: UUID of the version to activate

        Returns:
            The activated GuardrailVersion entity

        Raises:
            VersionNotFoundError: If the version is not found
        """
        # Verify version exists
        version = await self.repo.get_version(version_id)
        if not version:
            raise VersionNotFoundError(str(version_id))

        # Deactivate all versions first (ensures single active invariant)
        await self.repo.deactivate_all()

        # Activate the specified version
        await self.repo.activate(version_id)

        # Return updated version
        updated = await self.repo.get_version(version_id)
        if not updated:
            raise VersionNotFoundError(str(version_id))

        return updated

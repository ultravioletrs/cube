# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from datetime import datetime, timezone
from typing import Optional
from uuid import UUID, uuid4

from src.domain.entities import GuardrailVersion
from src.ports.repositories import GuardrailRepository
from src.use_cases.exceptions import ConfigNotFoundError


class CreateVersion:
    """
    Use case for creating a new version of a guardrail configuration.
    """

    def __init__(self, repo: GuardrailRepository):
        self.repo = repo

    async def execute(
        self,
        config_id: UUID,
        name: str,
        description: Optional[str] = None,
        version_id: Optional[UUID] = None,
    ) -> GuardrailVersion:
        """
        Create a new version for a guardrail configuration.

        Args:
            config_id: UUID of the parent configuration
            name: Name for this version
            description: Optional description
            version_id: Optional UUID (generated if not provided)

        Returns:
            The created GuardrailVersion entity

        Raises:
            ConfigNotFoundError: If the parent config is not found
        """
        # Verify config exists
        config = await self.repo.get_config(config_id)
        if not config:
            raise ConfigNotFoundError(str(config_id))

        # Get next revision number
        latest = await self.repo.get_latest_version(config_id)
        next_revision = (latest.revision + 1) if latest else 1

        version = GuardrailVersion(
            id=version_id or uuid4(),
            config_id=config_id,
            name=name,
            revision=next_revision,
            is_active=False,
            created_at=datetime.now(timezone.utc),
            description=description,
        )

        return await self.repo.create_version(version)

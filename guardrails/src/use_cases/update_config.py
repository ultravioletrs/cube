# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from datetime import datetime, timezone
from typing import Optional
from uuid import UUID

from src.domain.entities import GuardrailConfig
from src.ports.repositories import GuardrailRepository
from src.use_cases.exceptions import (
    ConfigAlreadyExistsError,
    ConfigNotFoundError,
    InvalidConfigError,
)


class UpdateConfig:
    """
    Use case for updating an existing guardrail configuration.
    """

    def __init__(self, repo: GuardrailRepository):
        self.repo = repo

    async def execute(
        self,
        config_id: UUID,
        name: Optional[str] = None,
        config_yaml: Optional[str] = None,
        prompts_yaml: Optional[str] = None,
        colang: Optional[str] = None,
        description: Optional[str] = None,
    ) -> GuardrailConfig:
        """
        Update an existing guardrail configuration.

        Args:
            config_id: UUID of the configuration to update
            name: Optional new name
            config_yaml: Optional new config YAML content
            prompts_yaml: Optional new prompts YAML content
            colang: Optional new colang content
            description: Optional new description

        Returns:
            The updated GuardrailConfig entity

        Raises:
            ConfigNotFoundError: If the config is not found
            ConfigAlreadyExistsError: If the new name conflicts with existing
            InvalidConfigError: If the configuration content is invalid
        """
        # Fetch existing config
        existing = await self.repo.get_config(config_id)
        if not existing:
            raise ConfigNotFoundError(str(config_id))

        # Check name uniqueness if changing
        if name and name != existing.name:
            conflict = await self.repo.get_config_by_name(name)
            if conflict:
                raise ConfigAlreadyExistsError(name)

        # Validate config content if provided
        new_config_yaml = config_yaml if config_yaml is not None else existing.config_yaml
        if not new_config_yaml.strip():
            raise InvalidConfigError("config_yaml cannot be empty")

        # Create updated config (immutable entity pattern)
        updated = GuardrailConfig(
            id=existing.id,
            name=name if name is not None else existing.name,
            description=description if description is not None else existing.description,
            config_yaml=new_config_yaml,
            prompts_yaml=prompts_yaml if prompts_yaml is not None else existing.prompts_yaml,
            colang=colang if colang is not None else existing.colang,
            created_at=existing.created_at,
            updated_at=datetime.now(timezone.utc),
        )

        return await self.repo.update_config(updated)

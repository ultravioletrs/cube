# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from datetime import datetime, timezone
from typing import Optional
from uuid import UUID, uuid4

from src.domain.entities import GuardrailConfig
from src.ports.repositories import GuardrailRepository
from src.use_cases.exceptions import ConfigAlreadyExistsError, InvalidConfigError


class CreateConfig:
    """
    Use case for creating a new guardrail configuration.
    """

    def __init__(self, repo: GuardrailRepository):
        self.repo = repo

    async def execute(
        self,
        name: str,
        config_yaml: str,
        prompts_yaml: str = "",
        colang: str = "",
        description: Optional[str] = None,
        config_id: Optional[UUID] = None,
    ) -> GuardrailConfig:
        """
        Create a new guardrail configuration.

        Args:
            name: Unique name for the configuration
            config_yaml: YAML content for config.yml
            prompts_yaml: YAML content for prompts.yml
            colang: Colang content for rails
            description: Optional description
            config_id: Optional UUID (generated if not provided)

        Returns:
            The created GuardrailConfig entity

        Raises:
            ConfigAlreadyExistsError: If a config with the same name exists
            InvalidConfigError: If the configuration content is invalid
        """
        # Check for duplicate name
        existing = await self.repo.get_config_by_name(name)
        if existing:
            raise ConfigAlreadyExistsError(name)

        # Validate config content
        if not config_yaml.strip():
            raise InvalidConfigError("config_yaml cannot be empty")

        now = datetime.now(timezone.utc)
        config = GuardrailConfig(
            id=config_id or uuid4(),
            name=name,
            description=description,
            config_yaml=config_yaml,
            prompts_yaml=prompts_yaml or "",
            colang=colang or "",
            created_at=now,
            updated_at=now,
        )

        return await self.repo.create_config(config)

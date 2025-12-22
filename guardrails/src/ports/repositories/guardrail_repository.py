# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from abc import ABC, abstractmethod
from typing import List, Optional
from uuid import UUID

from src.domain.entities import GuardrailConfig, GuardrailVersion, MaterializedGuardrail


class GuardrailRepository(ABC):
    """
    Port (interface) for guardrail configuration persistence.
    Implementations are provided by adapters (e.g., PostgreSQL).
    """

    # ==================== Config CRUD ====================

    @abstractmethod
    async def create_config(self, config: GuardrailConfig) -> GuardrailConfig:
        """Create a new guardrail configuration."""
        pass

    @abstractmethod
    async def get_config(self, config_id: UUID) -> Optional[GuardrailConfig]:
        """Get a guardrail configuration by ID."""
        pass

    @abstractmethod
    async def get_config_by_name(self, name: str) -> Optional[GuardrailConfig]:
        """Get a guardrail configuration by name."""
        pass

    @abstractmethod
    async def list_configs(
        self, offset: int = 0, limit: int = 100
    ) -> List[GuardrailConfig]:
        """List all guardrail configurations with pagination."""
        pass

    @abstractmethod
    async def count_configs(self) -> int:
        """Count total number of guardrail configurations."""
        pass

    @abstractmethod
    async def update_config(self, config: GuardrailConfig) -> GuardrailConfig:
        """Update an existing guardrail configuration."""
        pass

    @abstractmethod
    async def delete_config(self, config_id: UUID) -> bool:
        """Delete a guardrail configuration by ID."""
        pass

    # ==================== Version Management ====================

    @abstractmethod
    async def create_version(self, version: GuardrailVersion) -> GuardrailVersion:
        """Create a new version for a configuration."""
        pass

    @abstractmethod
    async def get_version(self, version_id: UUID) -> Optional[GuardrailVersion]:
        """Get a version by ID."""
        pass

    @abstractmethod
    async def list_versions(
        self, config_id: UUID, offset: int = 0, limit: int = 100
    ) -> List[GuardrailVersion]:
        """List all versions for a configuration."""
        pass

    @abstractmethod
    async def count_versions(self, config_id: UUID) -> int:
        """Count total number of versions for a configuration."""
        pass

    @abstractmethod
    async def get_latest_version(self, config_id: UUID) -> Optional[GuardrailVersion]:
        """Get the latest version for a configuration."""
        pass

    @abstractmethod
    async def get_active_version(self) -> Optional[GuardrailVersion]:
        """Get the currently active version (only one can be active)."""
        pass

    # ==================== Activation ====================

    @abstractmethod
    async def deactivate_all(self) -> None:
        """Deactivate all versions (ensure single active invariant)."""
        pass

    @abstractmethod
    async def activate(self, version_id: UUID) -> None:
        """Activate a specific version."""
        pass

    # ==================== Materialization ====================

    @abstractmethod
    async def fetch_active_materialized(self) -> Optional[MaterializedGuardrail]:
        """
        Fetch the materialized (denormalized) active guardrail.
        Returns None if no active version exists.
        """
        pass

    @abstractmethod
    async def materialize_version(self, version_id: UUID) -> MaterializedGuardrail:
        """
        Materialize a version for runtime use.
        Combines config, prompts, and colang into a single entity.
        """
        pass

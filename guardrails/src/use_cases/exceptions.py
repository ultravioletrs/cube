# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0


class UseCaseError(Exception):
    """Base exception for use case errors."""

    pass


class ConfigNotFoundError(UseCaseError):
    """Raised when a configuration is not found."""

    def __init__(self, config_id: str):
        self.config_id = config_id
        super().__init__(f"Configuration not found: {config_id}")


class ConfigAlreadyExistsError(UseCaseError):
    """Raised when trying to create a configuration that already exists."""

    def __init__(self, name: str):
        self.name = name
        super().__init__(f"Configuration already exists: {name}")


class VersionNotFoundError(UseCaseError):
    """Raised when a version is not found."""

    def __init__(self, version_id: str):
        self.version_id = version_id
        super().__init__(f"Version not found: {version_id}")


class NoActiveVersionError(UseCaseError):
    """Raised when no active version exists."""

    def __init__(self):
        super().__init__("No active guardrail version configured")


class InvalidConfigError(UseCaseError):
    """Raised when configuration content is invalid."""

    def __init__(self, message: str):
        super().__init__(f"Invalid configuration: {message}")


class MaterializationError(UseCaseError):
    """Raised when materialization fails."""

    def __init__(self, message: str):
        super().__init__(f"Materialization failed: {message}")

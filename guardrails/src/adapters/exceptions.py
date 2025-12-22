# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0


class AdapterError(Exception):
    """Base exception for adapter errors."""

    pass


class RepositoryError(AdapterError):
    """Raised when a repository operation fails."""

    pass


class DatabaseConnectionError(RepositoryError):
    """Raised when database connection fails."""

    pass


class RuntimeError(AdapterError):
    """Raised when runtime operation fails."""

    pass


class ConfigLoadError(RuntimeError):
    """Raised when loading configuration into runtime fails."""

    pass

# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from dataclasses import dataclass
import hashlib


@dataclass(frozen=True)
class Checksum:
    """
    Immutable hash of content for integrity verification.
    """

    value: str

    @classmethod
    def from_content(cls, content: str) -> "Checksum":
        """Create a checksum from string content using SHA-256."""
        hash_value = hashlib.sha256(content.encode("utf-8")).hexdigest()
        return cls(value=hash_value)

    @classmethod
    def from_multiple(cls, *contents: str) -> "Checksum":
        """Create a checksum from multiple content strings."""
        delimiter = "\n"
        combined = delimiter.join(contents)
        return cls.from_content(combined)

    def verify(self, content: str) -> bool:
        """Verify content matches this checksum."""
        return self == Checksum.from_content(content)

    def __str__(self) -> str:
        return self.value

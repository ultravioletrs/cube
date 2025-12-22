# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from dataclasses import dataclass


@dataclass(frozen=True)
class Revision:
    """
    Monotonic version counter for guardrail configurations.
    """

    value: int

    def __post_init__(self):
        if self.value < 1:
            raise ValueError("Revision must be >= 1")

    def next(self) -> "Revision":
        """Return the next revision number."""
        return Revision(value=self.value + 1)

    def __lt__(self, other: "Revision") -> bool:
        return self.value < other.value

    def __le__(self, other: "Revision") -> bool:
        return self.value <= other.value

    def __gt__(self, other: "Revision") -> bool:
        return self.value > other.value

    def __ge__(self, other: "Revision") -> bool:
        return self.value >= other.value

    def __int__(self) -> int:
        return self.value

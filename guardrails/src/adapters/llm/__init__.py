# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

from src.adapters.llm.extended_ollama import ExtendedOllama
from src.adapters.llm.extended_vllm import ExtendedVLLM

__all__ = ["ExtendedOllama", "ExtendedVLLM"]
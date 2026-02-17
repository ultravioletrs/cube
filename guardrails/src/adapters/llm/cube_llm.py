# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

"""
Extended LLM provider that supports dynamic headers injection and multiple backends.

This provides a single interface for OpenAI-compatible backends (vLLM, Ollama, etc.). It reads per-request
headers from NeMo Guardrails' generation_options_var context variable.

Usage:
    1. Register this provider in nemo_runtime.py:
       register_llm_provider("CubeLLM", ExtendedLLM)

    2. Configure in config.yml:
       models:
         - type: main
           engine: CubeLLM
           model: llama3.2:3b (or microsoft/DialoGPT-medium for vLLM)
           parameters:
             base_url: http://cube-proxy:8900/v1  (Ensure /v1 suffix for OpenAI compact)
             headers:
               X-Guardrails-Request: "true"
"""

import logging
from typing import Any, Dict, List, Optional

from langchain_core.messages import BaseMessage
from langchain_core.outputs import ChatResult
from langchain_openai import ChatOpenAI
from nemoguardrails.context import generation_options_var

logger = logging.getLogger(__name__)


class CubeLLM(ChatOpenAI):
    _config_headers: Dict[str, str] = {}

    def __init__(self, headers: Optional[Dict[str, str]] = None, **kwargs):
        logger.info(
            f"CubeLLM.__init__: model={kwargs.get('model_name')}, "
            f"base_url={kwargs.get('base_url')}, "
            f"config_headers={list((headers or {}).keys())}"
        )

        self._config_headers = headers or {}
        
        # Default to "EMPTY" api_key if not provided (common for vLLM/Ollama local)
        if "api_key" not in kwargs and "openai_api_key" not in kwargs:
            kwargs["api_key"] = "EMPTY"

        super().__init__(**kwargs)
        logger.info(f"CubeLLM instance created for model: {self.model_name}")

    def _get_headers_from_context(self) -> Dict[str, str]:
        try:
            gen_options = generation_options_var.get()
            if gen_options and gen_options.llm_params:
                headers = gen_options.llm_params.get("headers", {})
                if headers:
                    logger.debug(f"CubeLLM: found headers in context: {list(headers.keys())}")
                return headers or {}
            return {}
        except LookupError:
            logger.debug("CubeLLM: generation_options_var not set")
            return {}

    def _get_model_from_context(self) -> Optional[str]:
        try:
            gen_options = generation_options_var.get()
            if gen_options and gen_options.llm_params:
                model = gen_options.llm_params.get("model")
                if model:
                    logger.debug(f"CubeLLM: found model in context: {model}")
                return model
            return None
        except LookupError:
            logger.debug("CubeLLM: generation_options_var not set for model")
            return None

    def _merge_headers(self) -> Dict[str, str]:
        base_headers = self._config_headers or {}
        request_headers = self._get_headers_from_context()
        final_headers = {**base_headers, **request_headers}
        return final_headers

    def _extract_options(self, kwargs: Dict[str, Any]) -> Dict[str, Any]:
        """
        Extract valid options for ChatOpenAI and handle provider-specifics (like Ollama).
        Any unknown kwargs are typically passed as model_kwargs by LangChain, but strict
        filtering helps avoid issues.
        """
        valid_options = {
            "model_kwargs", "frequency_penalty", "presence_penalty", "n", 
            "logit_bias", "streaming", "seed", "response_format", "user"
        }
        options = {}
        for key in valid_options:
            if key in kwargs:
                options[key] = kwargs[key]
            elif hasattr(self, key):
                 # ChatOpenAI handles defaults, we just want overrides from kwargs
                 pass

        
        return options

    async def _acall(
        self,
        prompt: str,
        stop: Optional[List[str]] = None,
        run_manager: Optional[Any] = None,
        **kwargs: Any,
    ) -> str:
        """Required by NeMo Guardrails LLM provider registration."""
        from langchain_core.messages import HumanMessage

        logger.debug(f"CubeLLM._acall: prompt length={len(prompt)}")

        messages = [HumanMessage(content=prompt)]
        result = await self._agenerate(
            messages=messages,
            stop=stop,
            run_manager=run_manager,
            **kwargs
        )

        if result.generations:
            return result.generations[0].text
        return ""

    async def _agenerate(
        self,
        messages: List[BaseMessage],
        stop: Optional[List[str]] = None,
        run_manager: Optional[Any] = None,
        **kwargs: Any,
    ) -> ChatResult:
        if "headers" in kwargs:
            kwargs.pop("headers")

        # Remove arguments that we explicitly pass to ChatOpenAI to avoid "multiple values" error
        for key in ["model", "temperature", "max_tokens", "base_url", "api_key", "openai_api_key"]:
            if key in kwargs:
                kwargs.pop(key)

        final_headers = self._merge_headers()

        model = self._get_model_from_context() or self.model_name
        logger.info(f"CubeLLM: using model '{model}' (from context: {self._get_model_from_context() is not None})")

        # Normalize base_url to ensure it ends with /v1
        base_url = str(self.openai_api_base)
        if base_url and not base_url.endswith("/v1"):
            base_url = f"{base_url.rstrip('/')}/v1"

        # Create a temporary client for this request to inject dynamic headers
        temp_client = ChatOpenAI(
            model=model,
            base_url=base_url,
            api_key=self.openai_api_key.get_secret_value() if self.openai_api_key else "EMPTY",
            default_headers=final_headers,
            temperature=self.temperature,
            max_tokens=self.max_tokens,
            # Pass other config that might be on 'self' but not in kwargs
            # We trust kwargs to override or supplement
            **kwargs 
        )

        return await temp_client._agenerate(
            messages, stop=stop, run_manager=run_manager, **kwargs
        )

    def _generate(
        self,
        messages: List[BaseMessage],
        stop: Optional[List[str]] = None,
        run_manager: Optional[Any] = None,
        **kwargs: Any,
    ) -> ChatResult:
        if "headers" in kwargs:
            kwargs.pop("headers")

        # Remove arguments that we explicitly pass to ChatOpenAI to avoid "multiple values" error
        for key in ["model", "temperature", "max_tokens", "base_url", "api_key", "openai_api_key"]:
            if key in kwargs:
                kwargs.pop(key)

        final_headers = self._merge_headers()

        model = self._get_model_from_context() or self.model_name
        logger.info(f"CubeLLM: using model '{model}' (from context: {self._get_model_from_context() is not None})")

        # Normalize base_url to ensure it ends with /v1
        base_url = str(self.openai_api_base)
        if base_url and not base_url.endswith("/v1"):
            base_url = f"{base_url.rstrip('/')}/v1"

        temp_client = ChatOpenAI(
            model=model,
            base_url=base_url,
            api_key=self.openai_api_key.get_secret_value() if self.openai_api_key else "EMPTY",
            default_headers=final_headers,
            temperature=self.temperature,
            max_tokens=self.max_tokens,
            **kwargs
        )

        return temp_client._generate(
            messages, stop=stop, run_manager=run_manager, **kwargs
        )

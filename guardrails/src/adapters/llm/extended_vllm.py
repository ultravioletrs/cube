# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

"""
Extended vLLM LLM provider that supports dynamic headers injection.

This wrapper uses the OpenAI-compatible API exposed by vLLM and reads per-request
headers from NeMo Guardrails' generation_options_var context variable, allowing
per-request authentication tokens to be passed through to the backend LLM proxy.

Usage:
    1. Register this provider in nemo_runtime.py:
       register_llm_provider("CubeVLLM", ExtendedVLLM)

    2. Configure in config.yml:
       models:
         - type: main
           engine: CubeVLLM
           model: microsoft/DialoGPT-medium
           parameters:
             base_url: http://vllm:8000/v1
             headers:
               X-Guardrails-Request: "true"

    3. Pass per-request headers via generate_async options:
       await rails.generate_async(
           messages=[...],
           options={
               "llm_params": {
                   "headers": {
                       "Authorization": "Bearer <token>"
                   }
               }
           }
       )
"""

import logging
from typing import Any, Dict, List, Optional

from langchain_core.messages import BaseMessage
from langchain_core.outputs import ChatResult
from langchain_openai import ChatOpenAI
from nemoguardrails.context import generation_options_var

logger = logging.getLogger(__name__)


class ExtendedVLLM(ChatOpenAI):
    """
    Extended ChatOpenAI wrapper for vLLM that supports dynamic headers injection.
    vLLM exposes an OpenAI-compatible API, so we use ChatOpenAI as the base class.
    """

    _config_headers: Dict[str, str] = {}

    def __init__(self, headers: Optional[Dict[str, str]] = None, **kwargs):
        logger.info(
            f"ExtendedVLLM.__init__: model={kwargs.get('model_name')}, "
            f"base_url={kwargs.get('base_url')}, "
            f"config_headers={list((headers or {}).keys())}"
        )

        self._config_headers = headers or {}

        # vLLM doesn't require an API key, but ChatOpenAI requires one
        # Set a dummy key if not provided
        if "api_key" not in kwargs and "openai_api_key" not in kwargs:
            kwargs["api_key"] = "EMPTY"

        super().__init__(**kwargs)
        logger.info(f"ExtendedVLLM instance created for model: {self.model_name}")

    def _get_headers_from_context(self) -> Dict[str, str]:
        try:
            gen_options = generation_options_var.get()
            if gen_options and gen_options.llm_params:
                headers = gen_options.llm_params.get("headers", {})
                if headers:
                    logger.debug(f"ExtendedVLLM: found headers in context: {list(headers.keys())}")
                return headers or {}
            return {}
        except LookupError:
            logger.debug("ExtendedVLLM: generation_options_var not set")
            return {}

    def _get_model_from_context(self) -> Optional[str]:
        try:
            gen_options = generation_options_var.get()
            if gen_options and gen_options.llm_params:
                model = gen_options.llm_params.get("model")
                if model:
                    logger.debug(f"ExtendedVLLM: found model in context: {model}")
                return model
            return None
        except LookupError:
            logger.debug("ExtendedVLLM: generation_options_var not set for model")
            return None

    def _merge_headers(self) -> Dict[str, str]:
        base_headers = self._config_headers or {}
        request_headers = self._get_headers_from_context()
        final_headers = {**base_headers, **request_headers}
        return final_headers

    async def _acall(
        self,
        prompt: str,
        stop: Optional[List[str]] = None,
        run_manager: Optional[Any] = None,
        **kwargs: Any,
    ) -> str:
        """Required by NeMo Guardrails LLM provider registration."""
        from langchain_core.messages import HumanMessage

        logger.debug(f"ExtendedVLLM._acall: prompt length={len(prompt)}")

        messages = [HumanMessage(content=prompt)]
        result = await self._agenerate(
            messages=messages,
            stop=stop,
            run_manager=run_manager,
            **kwargs
        )

        # Extract text from ChatResult
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
        final_headers = self._merge_headers()

        # Get model from context, fallback to self.model_name
        model = self._get_model_from_context() or self.model_name
        logger.info(f"ExtendedVLLM: using model '{model}' (from context: {self._get_model_from_context() is not None})")

        # Create temp client with merged headers and dynamic model
        temp_client = ChatOpenAI(
            model=model,
            base_url=str(self.openai_api_base),
            api_key=self.openai_api_key.get_secret_value() if self.openai_api_key else "EMPTY",
            default_headers=final_headers,
            temperature=self.temperature,
            max_tokens=self.max_tokens,
            **self._extract_openai_options(kwargs)
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
        final_headers = self._merge_headers()

        model = self._get_model_from_context() or self.model_name
        logger.info(f"ExtendedVLLM: using model '{model}' (from context: {self._get_model_from_context() is not None})")

        temp_client = ChatOpenAI(
            model=model,
            base_url=str(self.openai_api_base),
            api_key=self.openai_api_key.get_secret_value() if self.openai_api_key else "EMPTY",
            default_headers=final_headers,
            temperature=self.temperature,
            max_tokens=self.max_tokens,
            **self._extract_openai_options(kwargs)
        )

        return temp_client._generate(
            messages, stop=stop, run_manager=run_manager, **kwargs
        )

    def _extract_openai_options(self, kwargs: Dict[str, Any]) -> Dict[str, Any]:
        """Extract valid OpenAI options from kwargs to pass to constructor."""
        # ChatOpenAI constructor args we want to preserve from the parent
        # or that might be passed in kwargs.
        
        valid_options = {
            "model_kwargs", "frequency_penalty", "presence_penalty", "n", 
            "logit_bias", "streaming"
        }
        
        options = {}
        for key in valid_options:
            if key in kwargs:
                options[key] = kwargs[key]
            elif hasattr(self, key):
                options[key] = getattr(self, key)
                 
        return options
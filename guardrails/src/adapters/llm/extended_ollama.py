# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

"""
Extended Ollama LLM provider that supports dynamic headers injection.

This wrapper reads per-request headers from NeMo Guardrails' generation_options_var
context variable, allowing per-request authentication tokens to be passed through
to the backend LLM proxy.

Usage:
    1. Register this provider in nemo_runtime.py:
       register_llm_provider("CubeLLM", ExtendedOllama)

    2. Configure in config.yml:
       models:
         - type: main
           engine: CubeLLM
           model: llama3.2:3b
           parameters:
             base_url: http://cube-proxy:8900
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
from typing import Any, Dict, List, Optional, Sequence

from langchain_core.messages import BaseMessage
from langchain_core.outputs import ChatResult
from langchain_ollama import ChatOllama
from nemoguardrails.context import generation_options_var

logger = logging.getLogger(__name__)


class ExtendedOllama(ChatOllama):

    _config_headers: Dict[str, str] = {}

    def __init__(self, headers: Optional[Dict[str, str]] = None, **kwargs):
        logger.info(
            f"ExtendedOllama.__init__: model={kwargs.get('model')}, "
            f"base_url={kwargs.get('base_url')}, "
            f"config_headers={list((headers or {}).keys())}"
        )

        self._config_headers = headers or {}

        super().__init__(**kwargs)
        logger.info(f"ExtendedOllama instance created for model: {self.model}")

    def _get_headers_from_context(self) -> Dict[str, str]:
        try:
            gen_options = generation_options_var.get()
            if gen_options and gen_options.llm_params:
                headers = gen_options.llm_params.get("headers", {})
                if headers:
                    logger.debug(f"ExtendedOllama: found headers in context: {list(headers.keys())}")
                return headers or {}
            return {}
        except LookupError:
            logger.debug("ExtendedOllama: generation_options_var not set")
            return {}

    def _get_model_from_context(self) -> Optional[str]:
        try:
            gen_options = generation_options_var.get()
            if gen_options and gen_options.llm_params:
                model = gen_options.llm_params.get("model")
                if model:
                    logger.debug(f"ExtendedOllama: found model in context: {model}")
                return model
            return None
        except LookupError:
            logger.debug("ExtendedOllama: generation_options_var not set for model")
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
        from langchain_core.messages import HumanMessage

        logger.debug(f"ExtendedOllama._acall: prompt length={len(prompt)}")

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

    def _extract_ollama_options(self, kwargs: Dict[str, Any]) -> Dict[str, Any]:
        ollama_option_keys = {
            "temperature", "top_p", "top_k", "num_predict", "num_ctx",
            "repeat_penalty", "seed", "mirostat", "mirostat_eta", "mirostat_tau",
            "num_gpu", "main_gpu", "low_vram", "f16_kv", "vocab_only",
            "use_mmap", "use_mlock", "num_thread", "num_batch", "num_keep",
        }

        param_mapping = {
            "max_tokens": "num_predict",
            "max_new_tokens": "num_predict",
        }

        params_to_remove = {"headers", "timeout", "api_key", "model", "base_url"}

        options = {}

        # Remove params we handle separately
        for key in params_to_remove:
            kwargs.pop(key, None)

        for key in ollama_option_keys:
            if key in kwargs:
                options[key] = kwargs.pop(key)

        for src_key, dest_key in param_mapping.items():
            if src_key in kwargs:
                if dest_key not in options:
                    options[dest_key] = kwargs.pop(src_key)
                else:
                    kwargs.pop(src_key)

        return options

    async def _agenerate(
        self,
        messages: List[BaseMessage],
        stop: Optional[List[str]] = None,
        run_manager: Optional[Any] = None,
        **kwargs: Any,
    ) -> ChatResult:
        final_headers = self._merge_headers()
        ollama_options = self._extract_ollama_options(kwargs)
        
        # Get model from context, fallback to self.model
        model = self._get_model_from_context() or self.model
        logger.info(f"ExtendedOllama: using model '{model}' (from context: {self._get_model_from_context() is not None})")

        # Create temp client with merged headers and dynamic model
        temp_client = ChatOllama(
            model=model,
            base_url=self.base_url,
            format=self.format,
            keep_alive=self.keep_alive,
            client_kwargs={"headers": final_headers},
            **ollama_options,
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
        ollama_options = self._extract_ollama_options(kwargs)
        
        model = self._get_model_from_context() or self.model
        logger.info(f"ExtendedOllama: using model '{model}' (from context: {self._get_model_from_context() is not None})")

        temp_client = ChatOllama(
            model=model,
            base_url=self.base_url,
            format=self.format,
            keep_alive=self.keep_alive,
            client_kwargs={"headers": final_headers},
            **ollama_options,
        )

        return temp_client._generate(
            messages, stop=stop, run_manager=run_manager, **kwargs
        )
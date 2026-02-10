# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

"""
Extended OpenAI LLM provider that supports dynamic headers injection.

This wrapper reads per-request headers from NeMo Guardrails' generation_options_var
context variable, allowing per-request authentication tokens to be passed through
to the backend LLM proxy (e.g. vLLM).
"""

import logging
from typing import Any, Dict, List, Optional

from langchain_core.messages import BaseMessage
from langchain_core.outputs import ChatResult
from langchain_openai import ChatOpenAI
from nemoguardrails.context import generation_options_var

logger = logging.getLogger(__name__)


class ExtendedOpenAI(ChatOpenAI):
    """
    Extended OpenAI provider with dynamic header injection support.
    """

    _config_headers: Dict[str, str] = {}

    def __init__(self, headers: Optional[Dict[str, str]] = None, **kwargs):
        logger.info(
            f"ExtendedOpenAI.__init__: model={kwargs.get('model_name')}, "
            f"base_url={kwargs.get('base_url')}, "
            f"config_headers={list((headers or {}).keys())}"
        )

        self._config_headers = headers or {}
        # Filter out headers from kwargs to avoid passing them to ChatOpenAI twice
        # if they were passed in config
        if "headers" in kwargs:
            kwargs.pop("headers")

        super().__init__(**kwargs)
        logger.info(
            f"ExtendedOpenAI instance created for model: {self.model_name}"
        )

    def _get_headers_from_context(self) -> Dict[str, str]:
        try:
            gen_options = generation_options_var.get()
            if gen_options and gen_options.llm_params:
                headers = gen_options.llm_params.get("headers", {})
                if headers:
                    logger.debug(
                        f"ExtendedOpenAI: found headers in context: "
                        f"{list(headers.keys())}"
                    )
                return headers or {}
            return {}
        except LookupError:
            logger.debug("ExtendedOpenAI: generation_options_var not set")
            return {}

    def _get_model_from_context(self) -> Optional[str]:
        try:
            gen_options = generation_options_var.get()
            if gen_options and gen_options.llm_params:
                model = gen_options.llm_params.get("model")
                if model:
                    logger.debug(
                        f"ExtendedOpenAI: found model in context: {model}"
                    )
                return model
            return None
        except LookupError:
            logger.debug(
                "ExtendedOpenAI: generation_options_var not set for model"
            )
            return None

    def _merge_headers(self) -> Dict[str, str]:
        base_headers = self._config_headers or {}
        request_headers = self._get_headers_from_context()
        final_headers = {**base_headers, **request_headers}
        return final_headers

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

        # Create temp client with merged headers and dynamic model
        # We use default_headers in ChatOpenAI to inject custom headers
        temp_client = ChatOpenAI(
            model=model,
            api_key=self.openai_api_key,
            base_url=self.openai_api_base,
            organization=self.openai_organization,
            ticketing=self.ticketing,  # type: ignore
            verbose=self.verbose,
            callbacks=self.callbacks,
            tags=self.tags,
            metadata=self.metadata,
            rate_limiter=self.rate_limiter,
            default_headers=final_headers,
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

        temp_client = ChatOpenAI(
            model=model,
            api_key=self.openai_api_key,
            base_url=self.openai_api_base,
            organization=self.openai_organization,
            ticketing=self.ticketing,  # type: ignore
            verbose=self.verbose,
            callbacks=self.callbacks,
            tags=self.tags,
            metadata=self.metadata,
            rate_limiter=self.rate_limiter,
            default_headers=final_headers,
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
            "temperature", "max_tokens", "model_kwargs", "frequency_penalty",
            "presence_penalty", "n", "logit_bias", "streaming"
        }

        options = {}
        for key in valid_options:
            if key in kwargs:
                options[key] = kwargs[key]
            elif hasattr(self, key):
                options[key] = getattr(self, key)

        return options

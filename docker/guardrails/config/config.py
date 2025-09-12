"""Configuration and initialization for Cube Guardrails"""

import sys
import os

# Add the config directory to the Python path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from nemoguardrails import LLMRails
from nemoguardrails.actions import action


def init(app: LLMRails):
    """Initialize and register all custom actions"""
    from actions.logging_actions import (
        get_timestamp,
        get_message_length,
        estimate_token_count,
        log_structured
    )

    # Register logging actions
    app.register_action(get_timestamp, "get_timestamp")
    app.register_action(get_message_length, "get_message_length")
    app.register_action(estimate_token_count, "estimate_token_count")
    app.register_action(log_structured, "log_structured")

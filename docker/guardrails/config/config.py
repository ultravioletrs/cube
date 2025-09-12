"""Configuration and initialization for Cube Guardrails"""

import sys
import os

# Add the config directory to the Python path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from nemoguardrails import LLMRails
from nemoguardrails.actions import action


def init(app: LLMRails):
    """Initialize and register all custom actions"""
    pass

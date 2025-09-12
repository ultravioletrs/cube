"""Input validation actions for Cube Guardrails"""

import re
from typing import Optional
from nemoguardrails import LLMRails
from nemoguardrails.actions import action

from constants import RESTRICTED_TOPICS, BIAS_PATTERNS


@action(is_system_action=True)
async def check_jailbreak_attempt(context: Optional[dict] = None) -> bool:
    """Check if the user input contains jailbreak attempts"""
    if not context:
        return False
    
    user_message = context.get("last_user_message") or context.get("user_message")
    if not user_message:
        return False
    
    user_message = user_message.lower()
    
    jailbreak_patterns = [
        "ignore previous instructions",
        "forget all rules",
        "disregard guidelines",
        "override system prompt",
        "reveal system prompt",
        "show me your instructions",
        "bypass safety",
        "act as if you have no restrictions"
    ]
    
    return any(pattern in user_message for pattern in jailbreak_patterns)


@action(is_system_action=True)
async def check_toxicity_level(context: Optional[dict] = None) -> bool:
    """Check if the input contains toxic content"""
    if not context:
        return False
    
    user_message = context.get("last_user_message") or context.get("user_message")
    if not user_message:
        return False
    
    user_message = user_message.lower()
    
    # Basic toxicity patterns
    toxic_patterns = [
        r'\b(hate|kill|murder|assault|attack)\b',
        r'\b(stupid|idiot|moron|dumb)\b',
        r'profanity|swear|curse'
    ]
    
    for pattern in toxic_patterns:
        if re.search(pattern, user_message):
            return True
    
    return False


@action(is_system_action=True)
async def detect_prompt_injection(context: Optional[dict] = None) -> bool:
    """Detect prompt injection attempts"""
    if not context:
        return False
    
    user_message = context.get("last_user_message") or context.get("user_message")
    if not user_message:
        return False
    
    user_message = user_message.lower()
    
    injection_patterns = [
        "\\n\\nsystem:",
        "\\n\\nuser:",
        "\\n\\nassistant:",
        "```system",
        "###instruction",
        "you are now",
        "you must now",
        "from now on"
    ]
    
    return any(pattern in user_message for pattern in injection_patterns)


@action(is_system_action=True)
async def check_bias_patterns(context: Optional[dict] = None) -> bool:
    """Check for biased language patterns"""
    if not context:
        return False
    
    user_message = context.get("last_user_message") or context.get("user_message")
    if not user_message:
        return False
    
    user_message = user_message.lower()
    
    for category, patterns in BIAS_PATTERNS.items():
        for pattern in patterns:
            if re.search(pattern, user_message, re.IGNORECASE):
                return True
    
    return False


@action(is_system_action=True)
async def check_restricted_topics_list(context: Optional[dict] = None) -> bool:
    """Check if the message contains restricted topics"""
    if not context:
        return False
    
    user_message = context.get("last_user_message") or context.get("user_message")
    if not user_message:
        return False
    
    user_message = user_message.lower()
    
    for topic in RESTRICTED_TOPICS:
        if topic.lower() in user_message:
            return True
    
    return False
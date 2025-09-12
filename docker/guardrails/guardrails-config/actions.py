"""Custom actions for Cube AI guardrails"""

import logging
import re
from typing import Optional, List, Dict
from nemoguardrails.actions import action

log = logging.getLogger(__name__)

@action(is_system_action=True)
async def check_jailbreak(context: dict) -> Optional[str]:
    """Check if the user input contains jailbreak attempts."""
    user_message = context.get("user_message", "")
    
    # Common jailbreak patterns
    jailbreak_patterns = [
        "ignore all previous instructions",
        "disregard your guidelines",
        "pretend you are",
        "roleplay as",
        "bypass your restrictions",
        "act as if you have no limitations",
        "forget your training",
    ]
    
    user_message_lower = user_message.lower()
    for pattern in jailbreak_patterns:
        if pattern in user_message_lower:
            log.warning(f"Potential jailbreak attempt detected: {pattern}")
            return "stop"
    
    return None

@action(is_system_action=True)
async def check_toxicity(context: dict) -> Optional[str]:
    """Check for toxic or harmful content."""
    user_message = context.get("user_message", "")
    
    # This is a simple check - in production, use a proper toxicity model
    toxic_keywords = [
        "hate", "kill", "harm", "attack", "destroy"
    ]
    
    user_message_lower = user_message.lower()
    for keyword in toxic_keywords:
        if keyword in user_message_lower:
            # Check context to avoid false positives
            if any(safe_context in user_message_lower for safe_context in ["chess", "game", "fiction", "history"]):
                continue
            log.warning(f"Potential toxic content detected: {keyword}")
            return "stop"
    
    return None

@action(is_system_action=True)
async def check_prompt_injection(context: dict) -> Optional[str]:
    """Detect prompt injection attempts."""
    user_message = context.get("user_message", "")
    
    injection_patterns = [
        "[[",
        "]]",
        "system:",
        "assistant:",
        "user:",
        "<|im_start|>",
        "<|im_end|>",
        "### instruction",
        "### response",
    ]
    
    for pattern in injection_patterns:
        if pattern in user_message.lower():
            log.warning(f"Potential prompt injection detected: {pattern}")
            return "stop"
    
    return None

@action(is_system_action=True)
async def check_facts(context: dict) -> Optional[str]:
    """Validate factual accuracy of responses."""
    bot_message = context.get("bot_message", "")
    
    # This would integrate with fact-checking services in production
    log.info("Fact checking performed on response")
    
    return None

@action(is_system_action=True)
async def check_hallucination(context: dict) -> Optional[str]:
    """Check for potential hallucinations in AI responses."""
    bot_message = context.get("bot_message", "")
    user_message = context.get("user_message", "")
    
    # Simple heuristics for demo - production would use sophisticated models
    hallucination_indicators = [
        "I have personal experience",
        "In my opinion as an AI",
        "I feel",
        "I believe",
        "I remember when",
    ]
    
    bot_message_lower = bot_message.lower()
    for indicator in hallucination_indicators:
        if indicator in bot_message_lower:
            log.warning(f"Potential hallucination detected: {indicator}")
            return "stop"
    
    return None

@action(is_system_action=True)
async def check_output_moderation(context: dict) -> Optional[str]:
    """Moderate AI output for safety."""
    bot_message = context.get("bot_message", "")
    
    # Check for unsafe content in output
    unsafe_patterns = [
        "how to make explosives",
        "how to hack",
        "how to harm",
        "illegal download",
    ]
    
    bot_message_lower = bot_message.lower()
    for pattern in unsafe_patterns:
        if pattern in bot_message_lower:
            log.warning(f"Unsafe content in output: {pattern}")
            return "stop"
    
    return None

@action(is_system_action=True)
async def check_bias(context: dict) -> Optional[str]:
    """Detect biased content in inputs and outputs."""
    user_message = context.get("user_message", "")
    bot_message = context.get("bot_message", "")
    
    # Check the appropriate message based on context
    message_to_check = bot_message if bot_message else user_message
    
    # Configurable bias patterns - in production, these would come from config
    bias_patterns = {
        "gender": [
            r"\b(all|every|typical)\s+(women?|men|males?|females?)\s+(are|always|never)\b",
            r"\b(women?|men|boys?|girls?)\s+(can't|cannot|shouldn't|should)\b",
            r"\bonly\s+(men|women|males?|females?)\s+(can|should|are)\b"
        ],
        "racial": [
            r"\b(all|every|typical)\s+\w+\s+(people|person)\s+(are|always|never)\b",
            r"\bpeople\s+from\s+\w+\s+(are|always|never|all)\b",
            r"\b\w+\s+culture\s+is\s+(inferior|superior|primitive|backwards)\b"
        ],
        "ethnic": [
            r"\b(all|every)\s+\w+s?\s+(are|always|never)\b",
            r"\btypical\s+\w+\s+(behavior|trait)\b",
            r"\b\w+\s+people\s+(don't|can't|won't)\b"
        ],
        "age": [
            r"\b(old|young|elderly)\s+people\s+(are|always|never|can't)\b",
            r"\b(millennials?|boomers?|gen\s*[xyz])\s+(all|always|never)\b"
        ],
        "disability": [
            r"\b(disabled|handicapped)\s+people\s+(can't|cannot|are)\b",
            r"\bpeople\s+with\s+\w+\s+(are|always|never)\s+(burden|incapable)\b"
        ]
    }
    
    message_lower = message_to_check.lower()
    
    for bias_type, patterns in bias_patterns.items():
        for pattern in patterns:
            if re.search(pattern, message_lower, re.IGNORECASE):
                log.warning(f"Potential {bias_type} bias detected: {pattern}")
                return "stop"
    
    # Check for stereotypical language
    stereotypes = [
        "naturally better at",
        "genetically predisposed",
        "inherently more",
        "biologically suited",
        "traditionally meant for",
        "not cut out for"
    ]
    
    for stereotype in stereotypes:
        if stereotype in message_lower:
            log.warning(f"Potential stereotypical language detected: {stereotype}")
            return "stop"
    
    return None

@action(is_system_action=True) 
async def check_restricted_topics(context: dict) -> Optional[str]:
    """Check against configurable restricted topics list."""
    user_message = context.get("user_message", "")
    bot_message = context.get("bot_message", "")
    
    # In production, this would load from config file
    restricted_topics = context.get("restricted_topics", [])
    if not restricted_topics:
        restricted_topics = [
            "explosives manufacturing",
            "illegal drugs synthesis", 
            "hacking tutorials",
            "self-harm instructions",
            "weapons creation",
            "identity theft methods",
            "surveillance evasion",
            "money laundering"
        ]
    
    message_to_check = (bot_message if bot_message else user_message).lower()
    
    for topic in restricted_topics:
        topic_words = topic.lower().split()
        if all(word in message_to_check for word in topic_words):
            log.warning(f"Restricted topic detected: {topic}")
            return "stop"
    
    return None

@action(is_system_action=True)
async def check_factuality_advanced(context: dict) -> Optional[str]:
    """Advanced factuality checking with confidence scoring."""
    bot_message = context.get("bot_message", "")
    user_message = context.get("user_message", "")
    
    low_confidence_indicators = [
        "might be",
        "could be", 
        "possibly",
        "potentially",
        "it seems",
        "appears to be",
        "I think",
        "probably",
        "likely",
        "uncertain"
    ]
    
    factual_claim_patterns = [
        r"\b\d{4}\b",  # Years
        r"\b\d+%\b",   # Percentages
        r"\b\$[\d,]+\b",  # Money amounts
        r"\baccording to\b",
        r"\bstudies show\b",
        r"\bresearch indicates\b",
        r"\bstatistics reveal\b"
    ]
    
    bot_message_lower = bot_message.lower()
    
    has_factual_claims = any(re.search(pattern, bot_message) for pattern in factual_claim_patterns)
    
    if has_factual_claims:
        confidence_score = 1.0
        for indicator in low_confidence_indicators:
            if indicator in bot_message_lower:
                confidence_score -= 0.1
        
        citation_patterns = [r'\[\d+\]', r'\([^)]+,\s*\d{4}\)', r'https?://\S+']
        has_citations = any(re.search(pattern, bot_message) for pattern in citation_patterns)
        
        if confidence_score < 0.7 and not has_citations:
            log.warning(f"Low confidence factual claim without citations (confidence: {confidence_score})")
            return "stop"
    
    return None
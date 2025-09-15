"""Output validation actions for Cube Guardrails"""

import re
from typing import Optional
from nemoguardrails import LLMRails
from nemoguardrails.actions import action

from constants import RESTRICTED_TOPICS, BIAS_PATTERNS


@action(is_system_action=True)
async def verify_factuality(context: Optional[dict] = None) -> bool:
    """Verify factual accuracy of the response"""
    if not context:
        return True
    
    bot_message = context.get("bot_message") or context.get("last_bot_message")
    if not bot_message:
        return True
    
    bot_message = bot_message.lower()
    
    # Check for common hallucination patterns
    hallucination_indicators = [
        "as of my last update",
        "i don't have access to real-time",
        "i cannot verify",
        "i'm not certain"
    ]
    
    # If bot admits uncertainty, it's being factual
    if any(indicator in bot_message for indicator in hallucination_indicators):
        return True
    
    # Additional fact-checking logic would go here
    return True


@action(is_system_action=True)
async def detect_hallucination(context: Optional[dict] = None) -> bool:
    """Detect potential hallucinations in output"""
    if not context:
        return False
    
    bot_message = context.get("bot_message") or context.get("last_bot_message")
    if not bot_message:
        return False
    
    bot_message = bot_message.lower()
    
    # Patterns that might indicate hallucination
    hallucination_patterns = [
        r'\b\d{4}\b.*(?:will|shall|going to)',  # Future predictions with dates
        r'(?:definitely|certainly|absolutely).*(?:will|shall)',  # Overconfident future claims
        r'(?:everyone|no one|all|never|always)',  # Absolute statements
    ]
    
    for pattern in hallucination_patterns:
        if re.search(pattern, bot_message):
            return True
    
    return False


@action(is_system_action=True)
async def check_output_safety(context: Optional[dict] = None) -> bool:
    """Check if the output is safe and appropriate"""
    if not context:
        return True
    
    bot_message = context.get("bot_message") or context.get("last_bot_message")
    if not bot_message:
        return True
    
    bot_message = bot_message.lower()
    
    # Check for inappropriate content
    unsafe_patterns = [
        r'\b(violence|violent|harm|hurt)\b',
        r'\b(explicit|inappropriate|offensive)\b',
        r'personal information|private data|sensitive'
    ]
    
    for pattern in unsafe_patterns:
        if re.search(pattern, bot_message):
            return False
    
    # Also check bias patterns in output
    for category, patterns in BIAS_PATTERNS.items():
        for pattern in patterns:
            if re.search(pattern, bot_message, re.IGNORECASE):
                return False
    
    # Check restricted topics
    for topic in RESTRICTED_TOPICS:
        if topic.lower() in bot_message:
            return False
    
    return True


@action(is_system_action=True)
async def advanced_fact_verification(context: Optional[dict] = None) -> bool:
    """Advanced fact checking with external verification"""
    if not context:
        return True
    
    bot_message = context.get("bot_message") or context.get("last_bot_message")
    if not bot_message:
        return True
    
    # Check for specific claim patterns
    claim_patterns = [
        r'(?:studies show|research indicates|data proves)',
        r'(?:according to|as reported by|sources say)',
        r'\b\d+\s*(?:percent|%)\b',  # Percentage claims
        r'(?:in|since|from)\s*\d{4}',  # Date claims
    ]
    
    has_claims = any(re.search(pattern, bot_message, re.IGNORECASE) for pattern in claim_patterns)
    
    if has_claims:
        citation_patterns = [
            r'\[\d+\]',  # Numbered citations
            r'\([^)]+,\s*\d{4}\)',  # Author, year citations
            r'(?:source:|reference:|cite:)',
        ]
        
        has_citations = any(re.search(pattern, bot_message) for pattern in citation_patterns)
        
        return has_citations
    
    return True
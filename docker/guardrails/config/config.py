"""Configuration and initialization for Cube Guardrails"""

from nemoguardrails import LLMRails
from nemoguardrails.actions import action

# Additional action for checking blocked terms (referenced in blocked.co)
@action(is_system_action=True)
async def check_blocked_terms(context: dict) -> bool:
    """Check if the response contains blocked proprietary terms"""
    bot_message = context.get("bot_message", "")
    
    # Add any proprietary or sensitive terms to block
    blocked_terms = [
        # Add your specific blocked terms here
        "proprietary_technology",
        "internal_system",
        "confidential_process"
    ]
    
    bot_message_lower = bot_message.lower()
    for term in blocked_terms:
        if term in bot_message_lower:
            return True
    
    return False

# Stub for knowledge base check (referenced in output.co)
@action(is_system_action=True)
async def check_knowledge_base(context: dict) -> bool:
    """Check if knowledge base has relevant information"""
    # This is a stub - implement actual KB check if you have a knowledge base
    return False


def init(app: LLMRails):
    """Initialize and register all custom actions"""
    from actions.input_checks import (
        check_jailbreak_attempt,
        check_toxicity_level,
        detect_prompt_injection,
        check_bias_patterns,
        check_restricted_topics_list
    )

    from actions.output_checks import (
        verify_factuality,
        detect_hallucination,
        check_output_safety,
        advanced_fact_verification
    )
    
    # Register input validation actions
    app.register_action(check_jailbreak_attempt, "check_jailbreak_attempt")
    app.register_action(check_toxicity_level, "check_toxicity_level")
    app.register_action(detect_prompt_injection, "detect_prompt_injection")
    app.register_action(check_bias_patterns, "check_bias_patterns")
    app.register_action(check_restricted_topics_list, "check_restricted_topics_list")
    
    # Register output validation actions
    app.register_action(verify_factuality, "verify_factuality")
    app.register_action(detect_hallucination, "detect_hallucination")
    app.register_action(check_output_safety, "check_output_safety")
    app.register_action(advanced_fact_verification, "advanced_fact_verification")
    
    # Register additional actions
    app.register_action(check_blocked_terms, "check_blocked_terms")
    app.register_action(check_knowledge_base, "check_knowledge_base")
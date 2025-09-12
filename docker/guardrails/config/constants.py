"""Constants for Cube Guardrails"""

# Configurable restricted topics
RESTRICTED_TOPICS = [
    "explosives manufacturing",
    "illegal drugs synthesis", 
    "hacking tutorials",
    "self-harm instructions",
    "weapons creation",
    "identity theft methods",
    "surveillance evasion",
    "money laundering",
    "biological weapons",
    "chemical weapons",
    "terrorism planning",
    "child exploitation"
]

# Configurable bias patterns
BIAS_PATTERNS = {
    "gender": [
        r"(all|every|typical) (women?|men|males?|females?) (are|always|never)",
        r"(women?|men|boys?|girls?) (can't|cannot|shouldn't|should)",
        r"only (men|women|males?|females?) (can|should|are)"
    ],
    "racial": [
        r"(all|every|typical) \w+ (people|person) (are|always|never)",
        r"people from \w+ (are|always|never|all)",
        r"\w+ culture is (inferior|superior|primitive|backwards)"
    ],
    "age": [
        r"(old|young|elderly) people (are|always|never|can't)",
        r"(millennials?|boomers?|gen [xyz]) (all|always|never)"
    ],
    "disability": [
        r"(disabled|handicapped) people (can't|cannot|are)",
        r"people with \w+ (are|always|never) (burden|incapable)"
    ]
}
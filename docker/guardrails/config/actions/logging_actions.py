import json
from datetime import datetime
from typing import Any

async def get_timestamp() -> str:
    return datetime.utcnow().isoformat() + "Z"

async def get_message_length(message: str) -> int:
    return len(message) if message else 0

async def estimate_token_count(message: str) -> int:
    return len(message) // 4 if message else 0

async def log_structured(
    level: str = "INFO",
    event: str = "",
    **kwargs: Any
) -> None:
    log_entry = {
        "timestamp": await get_timestamp(),
        "level": level,
        "event": event,
        **kwargs
    }
    
    print(json.dumps(log_entry))
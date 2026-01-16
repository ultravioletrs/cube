---
id: chat-completions
title: Chat Completions
---

Cube AI supports OpenAI-compatible **Chat Completions** for conversational LLM usage.

## Endpoint

```http
POST /proxy/{domain_id}/v1/chat/completions
```

## Example Request

```bash
curl -k https://localhost/proxy/<domain_id>/v1/chat/completions \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tinyllama:1.1b",
    "messages": [
      { "role": "user", "content": "Hello from Cube AI" }
    ]
  }'
```

## Example Response

```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1766137794,
  "model": "tinyllama:1.1b",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello from Cube AI!"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 42,
    "completion_tokens": 46,
    "total_tokens": 88
  }
}
```

## Notes

- All requests are executed inside a **Trusted Execution Environment (TEE)**
- Requests are **fully domain-isolated**
- The API follows the OpenAI Chat Completions schema
- Feature support depends on the selected backend and model

---
id: completions
title: Completions
---

Cube AI supports OpenAI-compatible **text completions**.
This endpoint is considered **legacy** and is mainly provided for compatibility with older clients.

For new applications, **Chat Completions** are recommended.

---

## Endpoint

```http
POST /proxy/{domain_id}/v1/completions
```

---

## Example Request

```bash
curl -k https://localhost/proxy/<domain_id>/v1/completions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tinyllama:1.1b",
    "prompt": "Once upon a time"
  }'
```

---

## Response

Returns an OpenAI-compatible `completion` object.

---

## Notes

- Fully OpenAI-compatible request and response format
- Executed inside a Trusted Execution Environment (TEE)
- Domain-isolated execution
- Prefer **Chat Completions** for conversational or structured prompts

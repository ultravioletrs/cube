---
id: authentication
title: Authentication
sidebar_position: 2
---

Cube AI uses **token-based authentication** provided by the SuperMQ Users and Auth services.
All API requests must be authenticated using a valid **access token**.

Authentication is **required** before interacting with any Cube AI API endpoint.

---

## Overview

Cube AI authentication works in two steps:

1. **Issue an access token** using your username and password
2. **Use the access token** in API requests via the Cube AI proxy

Authentication is **user-based**, while access to models and APIs is **scoped by domain**.

---

## Issuing an Access Token

To obtain an access token, send a request to the Users service:

```bash
curl -ksSiX POST https://<cube-ai-instance>/users/tokens/issue \
  -H "Content-Type: application/json" \
  -d '{
    "username": "<your_email>",
    "password": "<your_password>"
  }'
```

### Response

```json
{
  "access_token": "<jwt_access_token>",
  "refresh_token": "<jwt_refresh_token>"
}
```

- `access_token` is used for API calls
- `refresh_token` can be used to obtain a new access token when it expires

---

## Using the Access Token

All Cube AI API requests must include the access token as a Bearer token:

```http
Authorization: Bearer <access_token>
```

Example:

```bash
curl -k https://<cube-ai-instance>/users \
  -H "Authorization: Bearer <access_token>"
```

---

## Domain-Scoped API Access

Cube AI exposes model APIs through a **domain-scoped proxy**.

All OpenAI-compatible API requests must be sent to:

```http
/proxy/<domain_id>/v1/
```

Example:

```bash
curl -k https://<cube-ai-instance>/proxy/<domain_id>/v1/models \
  -H "Authorization: Bearer <access_token>"
```

The domain determines:

- Which models are available
- Which backends are used (Ollama / vLLM)
- Access permissions

---

## Token Scope and Security

- Tokens are **issued per user**
- Domains control **model visibility and permissions**
- Tokens must be kept **secret**
- Tokens should be transmitted only over **HTTPS**

> For local development, Cube AI may run with self-signed certificates.
> In production, valid TLS certificates are required.

---

## Summary

- Cube AI uses **JWT-based authentication**
- Tokens are issued via the **Users service**
- All API calls require `Authorization: Bearer <token>`
- Model APIs are accessed via the **domain proxy**
- Authentication and domain isolation are core security features of Cube AI

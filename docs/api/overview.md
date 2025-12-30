---
id: overview
title: API Overview
---

Cube AI provides an **OpenAI-compatible API** for interacting with Large Language Models
running inside **Trusted Execution Environments (TEEs)**.

The API follows the same request and response formats as the OpenAI API, but **does not
use any OpenAI services**. All requests are handled by Cube AI infrastructure and routed
through the Cube Proxy, ensuring confidentiality and domain isolation.

---

## Base URL

All API requests must be sent through the Cube Proxy and are scoped to a **domain**.

```http
https://<cube-ai-instance>/proxy/<domain-id>/v1
```

Example (local development):

```http
https://localhost/proxy/451477f5-828f-46ca-b3fc-c59d6fd34797/v1
```

---

## Authentication

Cube AI uses **Bearer token authentication**.

You must include an access token in the `Authorization` header for every request:

```http
Authorization: Bearer <access_token>
```

Tokens can be issued using the Users service or via the Cube AI UI.

---

## Supported Endpoints

Cube AI currently supports the following OpenAI-compatible endpoints:

- **Chat Completions**
- **Completions**
- **Embeddings**
- **Models**
- **Speech to Text**
- **Translations** (limited / roadmap)

Each endpoint is documented in detail in the following sections.

---

## Confidentiality & Security

All inference requests are executed inside **hardware-protected Trusted Execution
Environments (TEEs)**.

This guarantees that:

- Prompts and responses cannot be accessed by the host OS or cloud provider
- Model weights remain protected
- Requests are isolated per domain
- Execution is tamper-resistant

Cube AI is suitable for privacy-sensitive workloads in enterprise, healthcare, finance,
and regulated environments.

---

## Compatibility Note

While Cube AI follows the OpenAI API interface, some advanced or experimental OpenAI
features may not be available.

If an endpoint or feature is not supported, Cube AI will return a standard HTTP error
response.

---
id: opencode
title: OpenCode Integration
sidebar_position: 5
---

## OpenCode Integration

This guide explains how to configure **OpenCode** to work with your Cube AI instance.  
OpenCode is an AI-powered code editor that can use the models hosted through Cube AI for code generation, editing, and general LLM assistance.

---

## Prerequisites

Before you begin, ensure you have:

- A running **Cube AI** instance  
- **OpenCode** installed locally  
- A Cube AI user account with an authentication token  
- A Cube AI domain with at least one coding-capable model pulled  
  (for example: Qwen Coder models)

---

## 1. Get Your Authentication Token

If you do not already have a Cube AI access token, generate one from the UI:

**UI:**  
`Profile → Tokens → Create Token`

Or via API:

```bash
curl -ksSiX POST https://<cube-ai-instance>/users/tokens/issue \
  -H "Content-Type: application/json" \
  -d '{
    "username": "<your_email>",
    "password": "<your_password>"
  }'
```

Copy the `access_token` from the response.

---

## 2. Pull Recommended Coding Models

Before using OpenCode, make sure coding models are available in your Cube AI domain.

Recommended models:

- `qwen2.5-coder:7b-instruct-q4_K_M`
- `qwen2.5-coder:3b`
- `deepseek-coder:6.7b-instruct-q4_K_M`

Pull a model using:

```bash
curl -k -X POST https://<cube-ai-instance>/proxy/<your_domain_id>/api/pull \
  -H "Authorization: Bearer <your_access_token>" \
  -d '{
    "name": "qwen2.5-coder:7b-instruct-q4_K_M"
  }'
```

---

## 3. Verify Available Models

To list all models available in your domain:

```bash
curl -k https://<cube-ai-instance>/proxy/<your_domain_id>/v1/models \
  -H "Authorization: Bearer <your_access_token>"
```

You should see the models you pulled earlier.

---

## 4. Install OpenCode

Follow installation instructions from the official guide:

[https://opencode.ai](https://opencode.ai)

Once installed, OpenCode will create its configuration folder:

```bash
~/.config/OpenCode/
```

---

## 5. Configure OpenCode to Use Cube AI

Edit or create the configuration file:

```json
~/.config/OpenCode/config.json
```

Add:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "cube/qwen2.5-coder:7b-instruct-q4_K_M",
  "provider": {
    "cube": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "Cube AI",
      "options": {
        "baseURL": "https://<cube-ai-instance>/proxy/<your_domain_id>/v1",
        "apiKey": "<your_access_token>"
      },
      "models": {
        "qwen2.5-coder:7b-instruct-q4_K_M": {
          "name": "Qwen 2.5 Coder 7B - Main",
          "tool_call": false,
          "reasoning": false
        }
      }
    }
  }
}
```

### Replace

- `<cube-ai-instance>` → usually `localhost`  
- `<your_domain_id>` → the domain ID  
- `<your_access_token>` → your Cube AI token  

### Important

Use **HTTPS** (`https://...`) unless Cube AI is explicitly running without TLS.

---

## 6. Verify OpenCode Connection

In the OpenCode editor, run:

```text
opencode /models
```

You should see something like:

![OpenCode Models](/img/opencode-models.png)

Select any model (example: `cube/qwen2.5-coder:7b-instruct-q4_K_M`).

Try a simple request:

![OpenCode Example](/img/opencode-example.png)

---

## Supported Features

### ✅ Code Generation

Autocompletions, refactors, rewrites, explanations.

### ✅ Chat

General AI interaction, reasoning, documentation queries.

### ✅ Tools (if the model supports them)

File operations, command execution, multi-step workflows.

---

## Troubleshooting

### ❌ Connection refused

- Ensure Cube AI is running (`make up`)  
- Check firewall rules  
- Verify HTTPS/TLS configuration  

### ❌ Authentication error

- Token may have expired  
- Ensure `"Authorization: Bearer"` header is present  
- Confirm the token belongs to the correct domain  

### ❌ Certificate errors (self-signed certs)

If running locally:

- Import your local CA certificate, or  
- Temporarily use **HTTP** during development  

---

## Summary

You have now successfully connected OpenCode to your Cube AI instance.  
This setup allows you to use powerful open-source coding models locally with full
**privacy**, **TEE security**, and **domain isolation**.

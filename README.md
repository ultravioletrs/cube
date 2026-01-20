<div align="center">

# Cube AI

Confidential computing framework for GPT-based applications.

OpenAI-compatible API, multiple LLM backends, and TEE-backed isolation for data and model privacy.

[![CI](https://github.com/ultravioletrs/cube/actions/workflows/main.yaml/badge.svg)](https://github.com/ultravioletrs/cube/actions/workflows/main.yaml)
[![UI CI](https://github.com/ultravioletrs/cube/actions/workflows/ui-ci.yaml/badge.svg)](https://github.com/ultravioletrs/cube/actions/workflows/ui-ci.yaml)
[![Check License](https://github.com/ultravioletrs/cube/actions/workflows/check-license.yaml/badge.svg)](https://github.com/ultravioletrs/cube/actions/workflows/check-license.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ultravioletrs/cube)](https://goreportcard.com/report/github.com/ultravioletrs/cube)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

[Guide](https://github.com/ultravioletrs/cube-docs) | [Docs](https://github.com/ultravioletrs/cube-docs) | [License](LICENSE)
</div>

## Introduction

Cube AI is a framework for building GPT-based applications using confidential computing. It protects user data and AI models with a trusted execution environment (TEE), which is a secure area of the processor that ensures code and data loaded inside it remain confidential and intact. This provides strong data confidentiality and code integrity even when the host environment is not fully trusted.

<p align="center">
  <img src="https://github.com/ultravioletrs/cube-docs/blob/main/docs/img/cube-ai.png">
</p>

## Why Cube AI

Traditional GPT-based applications often rely on public cloud services where operators or hardware providers can access prompts and model responses. Cube AI addresses these privacy concerns by executing inference inside TEEs, ensuring that user data and AI models remain protected from unauthorized access outside the enclave.

## Key Features

- **Secure Computing**: Cube AI uses secure enclaves to protect user data and AI models from unauthorized access.
- **Trusted Execution Environment (TEE)**: Cube AI uses a trusted execution environment to ensure that AI models are executed securely and in a controlled environment.
- **Scalability**: Cube AI can handle large amounts of data and AI models, making it suitable for applications that require high performance and scalability.
- **Multiple LLM Backend Support**: Supports both Ollama and vLLM for flexible model deployment and high-performance inference.
- **OpenAI-Compatible API**: Provides familiar API endpoints for easy integration with existing applications.

## Supported LLM Backends

### vLLM Integration

Cube AI now supports vLLM, a high-throughput and memory-efficient inference engine for Large Language Models. vLLM provides:

- **High Throughput**: Optimized for serving multiple concurrent requests with continuous batching
- **Memory Efficiency**: Advanced memory management techniques for large models
- **Fast Inference**: Optimized CUDA kernels and efficient attention mechanisms
- **Model Compatibility**: Supports popular architectures including LLaMA, Mistral, Qwen, and more

### Ollama Integration

Cube AI integrates with Ollama for local model deployment, providing:

- Model management and deployment
- Local inference
- Support for various open-source models

## How Cube AI Works

Cube AI uses TEEs to protect user data and AI models from unauthorized access. The TEE provides a secure execution space for trusted applications. In Cube AI, inference runs inside the TEE so prompts, responses, and model data are protected even if the host OS is compromised.

## Getting Started

### Prerequisites

- Docker and Docker Compose
- NVIDIA GPU with CUDA support (recommended for vLLM)
- Hardware with TEE support (AMD SEV-SNP or Intel TDX)

### Quick Start

1. **Clone the repository**

   ```bash
   git clone https://github.com/ultravioletrs/cube.git
   cd cube
   ```

2. **Start Cube AI services**

   ```bash
   make up
   ```

3. **Get your authentication token**

   All API requests require JWT authentication. Once services are running, obtain a token:

   ```bash
   curl -ksSiX POST https://localhost/users/tokens/issue \
     -H "Content-Type: application/json" \
     -d '{
       "username": "admin@example.com",
       "password": "m2N2Lfno"
     }'
   ```

   Response:

   ```json
   {
     "access_token": "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9...",
     "refresh_token": "..."
   }
   ```

4. **Create a domain**

   All API requests require a domain ID in the URL path. You can fetch a domain ID from the UI or create one via the API:

   ```bash
   curl -ksSiX POST https://localhost/domains \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     -d '{
       "name": "Magistrala",
       "route": "magistrala1",
       "tags": ["absmach", "IoT"],
       "metadata": {
         "region": "EU"
       }
     }'
   ```

   Response (includes `id`):

   ```json
   {
     "id": "d7f9b3b8-4f7e-4f44-8d47-1a6e5e6f7a2b",
     "name": "Magistrala",
     "route": "magistrala",
     "tags": ["absmach", "IoT"],
     "metadata": {
       "region": "EU"
     },
     "status": "enabled",
     "created_by": "c8c3e4f1-56b2-4a22-8e5f-8a77b1f9b2f4",
     "created_at": "2025-10-29T14:12:01Z",
     "updated_at": "2025-10-29T14:12:01Z"
   }
   ```

   Notes:
   - `name` and `route` are required fields.
   - `route` must be unique and cannot be changed after creation.
   - `metadata` must be a valid JSON object.
   - Save the `id` value for subsequent API requests.

5. **Verify the installation**

   List available models (replace `YOUR_DOMAIN_ID` with the domain ID from step 4):

   ```bash
   curl -k https://localhost/proxy/YOUR_DOMAIN_ID/v1/models \
     -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
   ```

6. **Make your first AI request**

   ```bash
   curl -k https://localhost/proxy/YOUR_DOMAIN_ID/v1/chat/completions \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     -d '{
       "model": "tinyllama:1.1b",
       "messages": [
         {
           "role": "user",
           "content": "Hello! How can you help me today?"
         }
       ]
     }'
   ```

## API Endpoints

Cube AI exposes all services through a Traefik reverse proxy. All protected endpoints require the `Authorization: Bearer <token>` header with a valid JWT token.

### Proxy Endpoints (OpenAI-Compatible)

**Base URL:** `https://localhost/proxy/`

Replace `{domainID}` with your domain ID from the Getting Started section.

| Method | Path                              | Description             |
|--------|-----------------------------------|-------------------------|
| GET    | `/{domainID}/v1/models`           | List available models   |
| POST   | `/{domainID}/v1/chat/completions` | Create chat completions |
| POST   | `/{domainID}/v1/completions`      | Create text completions |
| GET    | `/{domainID}/api/tags`            | List Ollama models      |
| POST   | `/{domainID}/api/generate`        | Generate completions    |
| POST   | `/{domainID}/api/chat`            | Chat completions        |

Example:

```bash
# OpenAI-compatible endpoint
curl -k https://localhost/proxy/YOUR_DOMAIN_ID/v1/chat/completions \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"tinyllama:1.1b","messages":[{"role":"user","content":"Hello"}]}'

# Ollama API endpoint
curl -k https://localhost/proxy/YOUR_DOMAIN_ID/api/tags \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Auth Endpoints

**Base URL:** `https://localhost/users`

| Method | Path                          | Description                            |
|--------|-------------------------------|----------------------------------------|
| POST   | `/users`                      | Register new user account              |
| POST   | `/users/tokens/issue`         | Issue access and refresh token (login) |
| POST   | `/users/tokens/refresh`       | Refresh access token                   |
| POST   | `/password/reset-request`     | Request password reset                 |
| PUT    | `/password/reset`             | Reset password with token              |

Example:

```bash
curl -ksSiX POST https://localhost/users/tokens/issue \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin@example.com",
    "password": "m2N2Lfno"
  }'
```

### Domains Endpoints

**Base URL:** `https://localhost/domains`

| Method | Path                          | Description                            |
|--------|-------------------------------|----------------------------------------|
| POST   | `/domains`                    | Create new domain                      |
| GET    | `/domains`                    | List domains with filters              |
| GET    | `/domains/{domainID}`         | Get domain details                     |
| PATCH  | `/domains/{domainID}`         | Update domain name, tags, and metadata |
| POST   | `/domains/{domainID}/enable`  | Enable a domain                        |
| POST   | `/domains/{domainID}/disable` | Disable a domain                       |
| POST   | `/domains/{domainID}/freeze`  | Freeze a domain                        |

Example:

```bash
curl -ksSiX POST https://localhost/domains \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "name": "Magistrala",
    "route": "magistrala1",
    "tags": ["absmach", "IoT"],
    "metadata": {
      "region": "EU"
    }
  }'
```

## Configuration

### vLLM Backend

Configure vLLM settings through the environment:

```bash
make up-vllm
```

### Ollama Backend

For Ollama integration:

```bash
make up-ollama
```

## Documentation

Project documentation is hosted at [Cube AI docs repository](https://github.com/ultravioletrs/cube-docs).

## License

Cube AI is published under the permissive [Apache-2.0](LICENSE) license.

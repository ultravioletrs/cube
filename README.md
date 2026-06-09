<div align="center">

# Cube AI

Confidential computing framework for GPT-based applications.

OpenAI-compatible API, multiple LLM backends, and TEE-backed isolation for data and model privacy.

[![CI](https://github.com/ultravioletrs/cube/actions/workflows/main.yaml/badge.svg)](https://github.com/ultravioletrs/cube/actions/workflows/main.yaml)
[![Check License](https://github.com/ultravioletrs/cube/actions/workflows/check-license.yaml/badge.svg)](https://github.com/ultravioletrs/cube/actions/workflows/check-license.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ultravioletrs/cube)](https://goreportcard.com/report/github.com/ultravioletrs/cube)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

[Guide](https://github.com/ultravioletrs/cube-docs) | [Docs](https://github.com/ultravioletrs/cube-docs) | [License](LICENSE)
</div>

## Introduction

Cube AI is a framework for building GPT-based applications using confidential computing. It protects user data and AI models with a trusted execution environment (TEE), which is a secure area of the processor that ensures code and data loaded inside it remain confidential and intact. This provides strong data confidentiality and code integrity even when the host environment is not fully trusted.

<p align="center">
  <img src="https://raw.githubusercontent.com/ultravioletrs/cube-docs/refs/heads/main/static/img/cube-ai.png">
</p>

## Why Cube AI

Traditional GPT-based applications often rely on public cloud services where operators or hardware providers can access prompts and model responses. Cube AI addresses these privacy concerns by executing inference inside TEEs, ensuring that user data and AI models remain protected from unauthorized access outside the enclave.

## Key Features

- **Trusted Execution Environment (TEE)**: Hardware-backed secure enclaves ensure AI models are executed in a controlled, isolated environment — protecting prompts, responses, and model data even if the host OS is compromised.
- **AI Safety Guardrails**: Input and output validation powered by NeMo Guardrails, including jailbreak and prompt-injection detection, off-topic filtering, toxicity checks, and sensitive-data masking via Presidio.
- **Comprehensive Audit Logging**: Every request is logged with trace IDs, guardrail decisions and violations, token usage, latency breakdowns, and attestation status — giving full compliance and observability visibility.
- **Remote Attestation**: SEV-SNP, TDX, and vTPM attestation verifies CVM integrity before any data is processed, with support for Azure Attestation Service.
- **Multiple LLM Backend Support**: Supports both Ollama and vLLM for flexible model deployment and high-performance inference.
- **OpenAI-Compatible API**: Provides familiar API endpoints for easy integration with existing applications.
- **Dynamic Route Management**: Create, update, and manage proxy routes at runtime through a dedicated REST API.
- **Observability**: Built-in Prometheus metrics, distributed tracing, and structured logging across all services.
- **Scalability**: Handles large-scale workloads with concurrent request batching (vLLM) and efficient resource management.

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

Cube ships two self-contained Docker stacks:

| Stack     | Path           | Use                    | Includes                                                                                                                      |
| --------- | -------------- | ---------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| **local** | `docker/local` | Day-to-day development | ATOM identity, Ollama, agent, proxy, embedder (RAG), web UI                                                                   |
| **prod**  | `docker/prod`  | Production             | Everything above **plus** Traefik TLS, guardrails, audit pipeline, image embedder, optional attestation and Cloudflare tunnel |

The local stack is intentionally minimal: **no attestation (TEE), no audit logs, no guardrails, no reverse proxy**. Services are reachable directly on `localhost` ports.

### Prerequisites

- Docker and Docker Compose
- ~8 GB RAM free (Ollama model + Postgres instances)
- TEE hardware (AMD SEV-SNP / Intel TDX) is **only** needed for attestation in the prod stack — not for local development

### Quick Start

1. **Clone the repository**

   ```bash
   git clone https://github.com/ultravioletrs/cube.git
   cd cube
   ```

2. **Build the Cube images**

   The local stack runs the `proxy`, `agent`, and `embedder` images built
   from this repo, so build them first:

   ```bash
   make docker-proxy docker-agent docker-embedder
   ```

   This tags `ghcr.io/ultravioletrs/cube/{proxy,agent,embedder}:latest`
   (and `:<version>`), exactly what the compose files reference. Use
   `make dockers` to build every service image (including guardrails,
   image-embedder, and ui).

3. **Start the local stack**

   ```bash
   make up        # start
   make logs      # follow logs
   make down      # stop
   make clean-volumes   # stop and wipe databases, models, uploads
   ```

   The first start pulls Ollama and ATOM images and downloads the
   `llama3.2:3b` and `nomic-embed-text` models — give it a few minutes.

   **Local access (direct ports, plain HTTP):**

   | Service                  | URL                           |
   | ------------------------ | ----------------------------- |
   | Cube web UI              | http://localhost:5173         |
   | ATOM identity UI         | http://localhost:3005         |
   | Cube proxy (API gateway) | http://localhost:8900         |
   | Cube embedder (RAG)      | http://localhost:8082         |
   | ATOM API (GraphQL)       | http://localhost:8080/graphql |

4. **Open the web UI and sign in**

   Open http://localhost:5173. The local ATOM has self-signup enabled, so
   create an account (email + password) from the login screen — email
   verification is skipped in local dev. After signing in, create a
   workspace (domain) and start chatting. Uploading documents in the UI
   exercises the RAG pipeline through the embedder.

   For frontend hot-reload development, see the [UI development guide](ui/README.md).

5. **(Optional) Use the API directly**

   Get a token from ATOM via GraphQL (use the account you created):

   ```bash
   curl -s http://localhost:8080/graphql \
     -H "Content-Type: application/json" \
     -d '{"query":"mutation($i:LoginInput!){login(input:$i){token entityId}}","variables":{"i":{"identity":"you@example.com","secret":"your-password","kind":"password"}}}'
   ```

   The `token` is your bearer token. Send chat requests through the proxy,
   with your workspace (domain) ID in the path:

   ```bash
   curl -s http://localhost:8900/YOUR_DOMAIN_ID/v1/chat/completions \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer YOUR_TOKEN" \
     -d '{"model":"llama3.2:3b","messages":[{"role":"user","content":"Hello!"}]}'
   ```

## API Endpoints

Inference and RAG go through the **cube-proxy** gateway. Every protected
request needs an `Authorization: Bearer <token>` header (token from ATOM —
see step 5 of the Quick Start). The workspace (domain) ID goes in the path.

- **Local base URL:** `http://localhost:8900`
- **Prod base URL:** `https://<your-domain>` (Traefik, under `/proxy`)

### Proxy Endpoints (OpenAI- and Ollama-compatible)

Replace `{domainID}` with your workspace ID.

| Method | Path                              | Description             |
| ------ | --------------------------------- | ----------------------- |
| GET    | `/{domainID}/v1/models`           | List available models   |
| POST   | `/{domainID}/v1/chat/completions` | Create chat completions |
| POST   | `/{domainID}/v1/completions`      | Create text completions |
| GET    | `/{domainID}/api/tags`            | List Ollama models      |
| POST   | `/{domainID}/api/chat`            | Chat completions        |

### Embedder Endpoints (RAG)

Document ingestion, sources, and retrieval are served by the embedder under
`/{domainID}/api/v1/...` (records, sources, conversations, retrieve, chat,
models). The web UI drives these; see the
[embedder runbook](internal/embedder/README.md) for the raw API.

### Identity (ATOM)

Authentication, users, workspaces, members, and invitations are handled by
**ATOM** over GraphQL at `http://localhost:8080/graphql` (local). The web UI
and `ui/src/lib` use it directly; manage identity from the UI or the ATOM UI
at `http://localhost:3005`.

## Configuration

### Local vs. prod

- **Local** (`docker/local/.env`, `docker/local/config.json`) — Ollama
  models, ports, and database credentials. No TEE, audit, or guardrails.
- **Prod** (`docker/prod/.env`, `docker/prod/config.json`) — set
  `CUBE_DOMAIN`, replace every `change-me-*` secret, and toggle
  `ATTESTED_TLS`. Guardrails, the audit pipeline, and the image embedder are
  always on in prod.

### Choosing the model

Edit `LLM_MODEL` / `EMBEDDING_MODEL` in the stack's `.env`, then
`make clean-volumes && make up` (or `make up-prod`) to re-pull.

### Image tags

All Cube images use `${CUBE_TAG}` (default `latest`). `make dockers` builds
`:latest` and `:<version>`; set `CUBE_TAG=<version>` in the `.env` to pin a
build. ATOM is pulled as `ghcr.io/absmach/atom:latest` and
`ghcr.io/absmach/atom-ui:latest`.

### Production stack

```bash
# edit docker/prod/.env first (CUBE_DOMAIN, secrets)
make up-prod      # start
make logs-prod    # follow logs
make down-prod    # stop
```

## Service Runbooks

- Embedder service overview and configuration: [internal/embedder/README.md](internal/embedder/README.md)
- Embedder ingest/retrieve workflow: [internal/embedder/workflows/ingest-retrieve.md](internal/embedder/workflows/ingest-retrieve.md)
- Agent attestation policy workflow: [agent/workflows/attestation-policy.md](agent/workflows/attestation-policy.md)

## Documentation

Project documentation is hosted at [Cube AI docs repository](https://github.com/ultravioletrs/cube-docs).

## License

Cube AI is published under the permissive [Apache-2.0](LICENSE) license.

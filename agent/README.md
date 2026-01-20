# Cube Agent

Cube Agent runs inside the CVM and exposes a small HTTP API for CVM attestation plus a reverse proxy to the configured LLM backend (Ollama, vLLM, or custom). Cube Proxy uses the agent for attestation and backend access.

## Configuration

The service is configured via environment variables (unset variables use defaults):

| Variable | Description | Default |
| --- | --- | --- |
| `UV_CUBE_AGENT_LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |
| `UV_CUBE_AGENT_INSTANCE_ID` | Instance ID returned by `/health` (auto-generated if empty) | `""` |
| `UV_CUBE_AGENT_HOST` | HTTP bind host | `0.0.0.0` |
| `UV_CUBE_AGENT_PORT` | HTTP bind port | `8901` |
| `UV_CUBE_AGENT_SERVER_CERT` | Path to PEM-encoded server certificate (TLS) | `""` |
| `UV_CUBE_AGENT_SERVER_KEY` | Path to PEM-encoded server key (TLS) | `""` |
| `UV_CUBE_AGENT_TARGET_URL` | LLM backend base URL for reverse proxy | `http://localhost:11434` |
| `UV_CUBE_AGENT_CA_URL` | CA URL for aTLS certificate provisioning | `""` |
| `UV_CUBE_AGENT_CERTS_TOKEN` | CA auth token used for aTLS certificate provisioning | `""` |
| `UV_CUBE_AGENT_CVM_ID` | CVM identifier used for certificate provisioning | `""` |
| `AGENT_MAA_URL` | Azure Attestation service URL | `https://sharedeus2.eus2.attest.azure.net` |
| `AGENT_OS_BUILD` | Attestation OS build identifier | `UVC` |
| `AGENT_OS_DISTRO` | Attestation OS distro identifier | `UVC` |
| `AGENT_OS_TYPE` | Attestation OS type identifier | `UVC` |
| `AGENT_VMPL` | AMD SEV-SNP VMPL value used for vTPM provider | `2` |

## Features

- **Attestation API**: Generates attestation reports for SEV-SNP and TDX-backed CVMs.
- **Reverse Proxy**: Forwards requests to the configured LLM backend.
- **Health and Metrics**: `/health` and `/metrics` endpoints for operational status.
- **Header Hygiene**: Drops `Authorization` when proxying and enforces JSON content type.

## Endpoints

| Method | Path | Description |
| --- | --- | --- |
| GET | `/health` | Health check with version and instance ID |
| GET | `/metrics` | Prometheus metrics |
| POST | `/attestation` | Generate attestation report |
| * | `/*` | Reverse proxy to `UV_CUBE_AGENT_TARGET_URL` |

### Attestation Request

`report_data` must be base64-encoded 64 bytes and `nonce` must be base64-encoded 32 bytes. Supported `attestation_type` values are `snp`, `tdx`, `vtpm`, and `snpvtpm`.

```bash
curl -X POST http://localhost:8901/attestation \
  -H "Content-Type: application/json" \
  -d '{
    "report_data": "<base64-64-bytes>",
    "nonce": "<base64-32-bytes>",
    "attestation_type": "snp",
    "to_json": true
  }'
```

If `to_json` is `true`, the response is JSON. Otherwise it returns raw binary with `Content-Type: application/octet-stream`.

## Architecture

- **Go**: Core service implementation.
- **HTTP (chi + go-kit)**: Routing and transport.
- **Cocos attestation providers**: SNP, SNPvTPM, and TDX attestation sources.
- **Reverse proxy**: Standard library `httputil` with header sanitization.

## Deployment

### Docker Compose

The `cube-agent` service is defined in `docker/cube-compose.yaml`. Update values in `docker/.env` to configure the agent.

### Local Run

```bash
UV_CUBE_AGENT_TARGET_URL=http://localhost:11434 \
UV_CUBE_AGENT_PORT=8901 \
go run ./cmd/agent
```

## Attestation Policy Workflow

For policy generation and aTLS setup, see `agent/workflows/attestation-policy.md`.

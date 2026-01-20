# Cube Guardrails

Cube Guardrails provides safety controls for LLM interactions using Nemo Guardrails, backed by a versioned configuration store. It runs as a dedicated service that Cube Proxy can call for guarded chat completions and configuration management.

## Configuration

The service is configured via environment variables (unset variables use defaults):

| Variable | Description | Default |
| --- | --- | --- |
| `UV_GUARDRAILS_DB_HOST` | PostgreSQL host | `guardrails-db` |
| `UV_GUARDRAILS_DB_PORT` | PostgreSQL port | `5432` |
| `UV_GUARDRAILS_DB_USER` | PostgreSQL user | `guardrails` |
| `UV_GUARDRAILS_DB_PASS` | PostgreSQL password | `guardrails` |
| `UV_GUARDRAILS_DB_NAME` | PostgreSQL database name | `guardrails` |

## Features

- **Config management**: CRUD for guardrail configs with versioning.
- **Runtime reload**: Swap active configurations without restarting the service.
- **Migrations**: Database schema migrations run on startup.
- **Bootstrap config**: Loads `./rails` as a default config when no active config exists.

## Endpoints

Base path: `/guardrails`

| Method | Path | Description |
| --- | --- | --- |
| GET | `/guardrails/health` | Health check |
| GET | `/guardrails` | Service info and endpoint list |
| POST | `/guardrails/configs` | Create config |
| GET | `/guardrails/configs` | List configs |
| GET | `/guardrails/configs/{config_id}` | Get config |
| PUT | `/guardrails/configs/{config_id}` | Update config |
| DELETE | `/guardrails/configs/{config_id}` | Delete config |
| POST | `/guardrails/configs/{config_id}/versions` | Create config version |
| GET | `/guardrails/configs/{config_id}/versions` | List config versions |
| POST | `/guardrails/versions/{version_id}/activate` | Activate version |
| POST | `/guardrails/reload` | Reload active configuration |

## Architecture

- **FastAPI**: REST API surface and OpenAPI spec.
- **PostgreSQL**: Stores guardrail configs and versions.
- **Nemo Guardrails**: Runtime policy engine for safe responses.
- **asyncpg**: Async database access layer.

## Deployment

### Docker Compose

The `cube-guardrails` service is defined in `docker/cube-compose.yaml`. Configure environment values in `docker/.env`.

### Local Run

```bash
cd guardrails
UV_GUARDRAILS_DB_HOST=localhost \
UV_GUARDRAILS_DB_PORT=5432 \
UV_GUARDRAILS_DB_USER=guardrails \
UV_GUARDRAILS_DB_PASS=guardrails \
UV_GUARDRAILS_DB_NAME=guardrails \
uvicorn main:app --host 0.0.0.0 --port 8001 --reload
```

## API Spec

See `api/guardrails.yaml` for the full OpenAPI definition.

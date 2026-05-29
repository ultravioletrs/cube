# Cube Embedder

Cube Embedder is the RAG ingestion and retrieval service in Cube AI. It manages:

- source registration (`google_drive`, `microsoft`, `s3`, plus `rclone` fallback types)
- record ingestion (including direct uploads)
- chunking and embedding
- user-scoped vector retrieval
- conversation persistence for chat history

## Configuration

The service is configured via `EMBEDDER_*` environment variables (see `docker/.env` for full defaults).

| Variable | Description | Default |
| --- | --- | --- |
| `EMBEDDER_HTTP_ADDR` | HTTP bind address | `:8080` |
| `EMBEDDER_DB_URL` | Postgres/pgvector connection URL | required |
| `EMBEDDER_AUTH_GRPC_URL` | SuperMQ auth gRPC endpoint | `auth:8181` |
| `EMBEDDER_LOG_LEVEL` | Log level | `info` |
| `EMBEDDER_EMBEDDING_TEXT_PROVIDER` | Embedding provider for text profile | `ollama` |
| `EMBEDDER_EMBEDDING_TEXT_BASE_URL` | Embedding API base URL | `http://ollama:11434` |
| `EMBEDDER_EMBEDDING_TEXT_MODEL` | Embedding model for text profile | `nomic-embed-text` |
| `EMBEDDER_EMBEDDING_*` | Profile and routing overrides (text/code/image/custom) | optional |
| `EMBEDDER_CHUNK_SIZE` | Chunk size | `512` |
| `EMBEDDER_CHUNK_OVERLAP` | Chunk overlap | `64` |
| `EMBEDDER_INGEST_*` | Queue polling and concurrency tuning | optional |
| `EMBEDDER_OBJECT_STORAGE_PROVIDER` | Storage backend (`s3` or `local`) | `local` |
| `EMBEDDER_S3_*` | S3/SeaweedFS credentials and endpoint | required for `s3` |
| `EMBEDDER_UPLOAD_DIR` | Local upload path when provider is `local` | `/tmp/embedder/uploads` |
| `EMBEDDER_RCLONE_BINARY` | rclone binary path for fallback sources | `/usr/bin/rclone` |
| `EMBEDDER_RCLONE_CONFIG_DIR` | rclone config base directory | `/etc/cube/rclone` |
| `EMBEDDER_RCLONE_TIMEOUT` | Timeout for rclone list/read operations | `2m` |
| `EMBEDDER_RCLONE_PREFLIGHT` | Run startup check (`rclone version` + config dir) and fail fast on errors | `true` |
| `EMBEDDER_OCR_ENABLED` | Enable OCR preprocessing | `false` |
| `EMBEDDER_OCR_IMAGE_ENABLED` | Run OCR on `image/*` records | `true` |
| `EMBEDDER_OCR_PDF_FALLBACK_ENABLED` | Run OCR for PDFs when `pdftotext` output is too small | `true` |
| `EMBEDDER_OCR_LANG` | Tesseract language(s), e.g. `eng` or `eng+srp` | `eng` |
| `EMBEDDER_OCR_BINARY` | OCR binary path/name | `tesseract` |
| `EMBEDDER_OCR_PDF_RENDER_BINARY` | PDF renderer binary for OCR fallback | `pdftoppm` |
| `EMBEDDER_OCR_TIMEOUT` | Per OCR command timeout | `2m` |
| `EMBEDDER_OCR_MIN_TEXT_CHARS` | Min extracted PDF chars before OCR fallback kicks in | `40` |
| `EMBEDDER_OCR_IMAGE_OCR_ONLY_MIN_TEXT_CHARS` | Image OCR text size at which image ingest skips visual embedding as a likely scanned text document | `1200` |
| `EMBEDDER_OCR_MAX_PDF_PAGES` | Max PDF pages rendered for OCR fallback | `20` |
| `EMBEDDER_IMAGE_EMBEDDING_URL` | Optional visual image embedding sidecar URL | optional |
| `EMBEDDER_IMAGE_EMBEDDING_MODEL` | Visual embedding model label sent to the sidecar | `openclip-vit-b-32` |
| `EMBEDDER_IMAGE_EMBEDDING_DIMENSIONS` | Expected visual embedding dimensions | `512` |
| `EMBEDDER_IMAGE_EMBEDDING_TIMEOUT` | Image embedding request timeout | `30s` |
| `EMBEDDER_GOOGLE_OAUTH_CLIENT_ID` | Google OAuth client ID | optional |
| `EMBEDDER_GOOGLE_OAUTH_CLIENT_SECRET` | Google OAuth client secret | optional |

## API Endpoints

All `/api/v1/*` routes require `Authorization: Bearer <token>`.

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/health` | Health check |
| `GET` | `/metrics` | Prometheus metrics |
| `GET` | `/api/v1/sources` | List sources |
| `POST` | `/api/v1/sources` | Create source (`google_drive`, `s3`, `microsoft`, `onedrive`, `sharepoint`, `dropbox` or `rclone`) |
| `POST` | `/api/v1/sources/{id}/sync` | Sync source and enqueue records |
| `DELETE` | `/api/v1/sources/{id}` | Delete source |
| `POST` | `/api/v1/records/upload` | Direct file upload and queue ingest |
| `GET` | `/api/v1/records` | List records |
| `GET` | `/api/v1/records/{id}` | Get record details |
| `POST` | `/api/v1/records/{id}/retry` | Retry failed record ingest |
| `DELETE` | `/api/v1/records/{id}` | Delete record |
| `POST` | `/api/v1/retrieve` | Vector/hybrid retrieval for a query |
| `GET` | `/api/v1/conversations` | List conversations |
| `POST` | `/api/v1/conversations` | Create conversation |
| `GET` | `/api/v1/conversations/{id}` | Get conversation with messages |
| `POST` | `/api/v1/conversations/{id}/messages` | Append messages |
| `DELETE` | `/api/v1/conversations/{id}` | Delete conversation |

## Deployment

In Docker Compose, Embedder runs as:

- service: `cube-embedder`
- database: `cube-embedder-db` (`pgvector/pgvector:pg16`)

The compose definition is in `docker/cube-compose.yaml`.

## Workflows

For an end-to-end upload and retrieval flow, see:

- [workflows/ingest-retrieve.md](workflows/ingest-retrieve.md)
- [workflows/ollama-vs-openai-eval.md](workflows/ollama-vs-openai-eval.md)
- [workflows/provider-rollout.md](workflows/provider-rollout.md)
- [monitoring/alerts/provider-alerts.yaml](monitoring/alerts/provider-alerts.yaml)

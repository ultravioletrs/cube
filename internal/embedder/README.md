# Cube Embedder

Cube Embedder is the RAG ingestion and retrieval service in Cube AI. It manages:

- source registration (`google_drive`, `microsoft`, `s3`, plus `rclone` fallback types)
- record ingestion (including direct uploads)
- chunking and embedding
- user-scoped vector retrieval
- conversation persistence for chat history

## Architecture

### Ingest pipeline

Every record flows through the same stages:

```
ingest → extract → chunk → embed → store → (retrieve)
```

1. **Ingest** – fetch the file from a source (`google_drive`, `s3`, …) or a direct upload.
2. **Extract** – turn the raw file into text. Format-specific: `pdftotext` for PDFs,
   a DOCX reader, HTML stripping for links, OCR (`tesseract`) for images and as a
   fallback for text-poor PDFs.
3. **Chunk** – split the extracted text into overlapping windows (`EMBEDDER_CHUNK_*`).
4. **Embed** – turn each chunk into a vector (see modalities below).
5. **Store** – persist vectors in pgvector, scoped to the owning user.
6. **Retrieve** – at query time, embed the query and return nearest chunks for chat.

### Two axes: format vs modality

The design deliberately separates two concerns that are easy to conflate:

- **Format** is an *extraction* concern. PDF, DOCX, Excel, Markdown and web links are
  all different containers that, once parsed, collapse into **text**. They share a
  single embedding path. Adding a new format means adding an extractor (a branch in
  `service/record_format.go` / `ingest/`), **not** a new service.
- **Modality** is an *embedding* concern. Text and images are fundamentally different
  inputs that need different models. Text uses the in-process embedding profiles
  (ollama/openai); images use a CLIP model with heavy Python/ML dependencies. That is
  the *only* reason image embedding is split out.

So there is one Go embedder for everything text-derived, plus one sidecar per
non-text modality.

### Embedding profiles vs the image sidecar

These are two **different** mechanisms — do not cross-wire them:

| | Text/code (and all text-yielding formats) | Images |
| --- | --- | --- |
| Mechanism | In-process embedding **profile** (`text`, `code`, …) | External **sidecar** HTTP call |
| Provider | `ollama` or `openai` only | `image-embedder` service (CLIP) |
| Config | `EMBEDDER_EMBEDDING_<PROFILE>_*` | `EMBEDDER_IMAGE_EMBEDDING_*` |
| Endpoint | n/a (library call) | `EMBEDDER_IMAGE_EMBEDDING_URL` → sidecar `:8090` |

The `image` profile name routes image records and must still name a *valid profile
provider* (`ollama`/`openai`); it is **not** where the sidecar is configured. Pointing
a profile at a provider like `image-embedder` fails startup with
`unknown embedding provider`. The sidecar is wired solely through
`EMBEDDER_IMAGE_EMBEDDING_URL`.

### Adding a new content type

- **New document format** (e.g. `.pptx`, RTF) → add an extractor; it reuses the text
  profile. No new service.
- **New modality** (audio, video) → add a dedicated sidecar (its own model and deps)
  and a corresponding `EMBEDDER_<MODALITY>_EMBEDDING_*` config block, mirroring images.

The companion CLIP service lives in [`image-embedder/`](../../image-embedder/) and is
documented in [`image-embedder/README.md`](../../image-embedder/README.md).

## Configuration

The service is configured via `EMBEDDER_*` environment variables (see `docker/local/.env` and `docker/prod/.env` for full defaults).

| Variable | Description | Default |
| --- | --- | --- |
| `EMBEDDER_HTTP_ADDR` | HTTP bind address | `:8080` |
| `EMBEDDER_DB_URL` | Postgres/pgvector connection URL | required |
| `EMBEDDER_AUTH_GRPC_URL` | ATOM auth gRPC endpoint | `atom:8081` |
| `EMBEDDER_LOG_LEVEL` | Log level | `info` |
| `EMBEDDER_EMBEDDING_TEXT_PROVIDER` | Embedding provider for text profile | `ollama` |
| `EMBEDDER_EMBEDDING_TEXT_BASE_URL` | Embedding API base URL | `http://ollama:11434` |
| `EMBEDDER_EMBEDDING_TEXT_MODEL` | Embedding model for text profile | `nomic-embed-text` |
| `EMBEDDER_EMBEDDING_*` | Profile and routing overrides (text/code/image/custom) | optional |
| `EMBEDDER_CHUNK_SIZE` | Chunk size. Large text records automatically disable overlap to reduce embedding work without exceeding local model context limits. | `512` |
| `EMBEDDER_CHUNK_OVERLAP` | Chunk overlap | `64` |
| `EMBEDDER_INGEST_*` | Queue polling and concurrency tuning | optional |
| `EMBEDDER_INGEST_RECORD_TIMEOUT` | Max wall-clock time for one record ingest | `2h` |
| `EMBEDDER_INGEST_MAX_CHUNKS` | Optional max chunks one record may produce before failing fast (`0` disables) | `0` |
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
| `EMBEDDER_IMAGE_EMBEDDING_TIMEOUT` | Image embedding request timeout | `2m` |
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
- database: `embedder-db` (`pgvector/pgvector:pg16`)
- image sidecar: `image-embedder` (CLIP)

The compose definitions are in `docker/local/docker-compose.yaml` (minimal local stack)
and `docker/prod/docker-compose.yaml` (full production stack).

## Workflows

For an end-to-end upload and retrieval flow, see:

- [workflows/ingest-retrieve.md](workflows/ingest-retrieve.md)
- [workflows/ollama-vs-openai-eval.md](workflows/ollama-vs-openai-eval.md)
- [workflows/provider-rollout.md](workflows/provider-rollout.md)
- [monitoring/alerts/provider-alerts.yaml](monitoring/alerts/provider-alerts.yaml)

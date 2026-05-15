# Embedder A/B Eval: Ollama vs OpenAI

This runbook compares retrieval + chat quality and latency between:

1. local Ollama models
2. OpenAI models (`text-embedding-3-small` + `gpt-4o-mini`)

## Why this matters

- Embedding quality drives retrieval relevance.
- Chat model quality drives final answer faithfulness and clarity.
- OCR affects whether scanned PDFs/images are searchable at all.

## Prerequisites

- Running stack (`make up-ollama` or `make up`)
- Valid user token
- A fixed test dataset containing:
  - native text PDF
  - scanned PDF (image-only)
  - image with meaningful text (PNG/JPG)
  - DOCX or Markdown with domain-specific terms

## Important schema note

`chunks.embedding` is currently dimension-constrained in migrations (`vector(768)`).

If you use OpenAI embeddings, keep dimensions at `768` so inserts/search stay compatible with existing schema and historical vectors.

Recommended OpenAI embedding setup:

```env
EMBEDDER_EMBEDDING_TEXT_PROVIDER=openai
EMBEDDER_EMBEDDING_TEXT_BASE_URL=https://api.openai.com
EMBEDDER_EMBEDDING_TEXT_MODEL=text-embedding-3-small
EMBEDDER_EMBEDDING_TEXT_DIMENSIONS=768
EMBEDDER_EMBEDDING_TEXT_API_KEY=<OPENAI_API_KEY>
```

## 1) Baseline run (Ollama)

Use existing default config from `docker/.env`:

```env
EMBEDDER_EMBEDDING_TEXT_PROVIDER=ollama
EMBEDDER_EMBEDDING_TEXT_MODEL=nomic-embed-text
EMBEDDER_LLM_PROVIDER=ollama
EMBEDDER_LLM_MODEL=llama3.1:8b
```

Restart embedder and reindex the same dataset. Then execute a fixed query set:

- exact fact lookup
- multi-hop question across 2+ files
- question where answer exists only in scanned/image docs
- out-of-scope question (should avoid hallucination)

Track:

- retrieval precision@k (manual relevance labeling is enough)
- citation usefulness (correct record/chunk)
- answer correctness
- average latency for `/api/v1/retrieve`
- time-to-first-token and full time for `/api/v1/chat`

## 2) OpenAI run (embedding + chat)

Switch env:

```env
EMBEDDER_EMBEDDING_TEXT_PROVIDER=openai
EMBEDDER_EMBEDDING_TEXT_BASE_URL=https://api.openai.com
EMBEDDER_EMBEDDING_TEXT_MODEL=text-embedding-3-small
EMBEDDER_EMBEDDING_TEXT_DIMENSIONS=768
EMBEDDER_EMBEDDING_TEXT_API_KEY=<OPENAI_API_KEY>

EMBEDDER_LLM_PROVIDER=openai
EMBEDDER_LLM_BASE_URL=https://api.openai.com
EMBEDDER_LLM_MODEL=gpt-4o-mini
EMBEDDER_LLM_API_KEY=<OPENAI_API_KEY>
```

Restart only embedder:

```bash
docker compose -f docker/compose.yaml --env-file docker/.env up -d cube-embedder
```

Reindex the exact same files and run the exact same query set.

## 3) Decide with pass/fail gates

Use OpenAI in production if all are true:

1. retrieval relevance improves by a meaningful margin on your dataset
2. hallucination rate decreases on out-of-scope queries
3. P95 latency and cost fit your SLA/budget
4. data residency/compliance constraints allow external provider use

Keep Ollama if privacy/offline and cost predictability are higher priority than quality gain.

## OCR guidance

Current behavior:

- native PDFs: extracted with `pdftotext` (good for digital PDFs)
- OCR (optional, env-controlled):
  - image files: OCR via `tesseract`
  - PDFs: OCR fallback via `pdftoppm` + `tesseract` when `pdftotext` text is too small

Recommendation:

1. Enable OCR preprocessing for image files and scanned PDFs if those formats are common.
2. Keep OCR optional per source type to avoid unnecessary ingestion cost.
3. Apply OCR before chunking/embedding so retrieval can hit actual text.

Minimum trigger policy that usually works well:

1. Run OCR for `image/*` files.
2. For PDFs, run OCR fallback only when `pdftotext` output is empty/near-empty.
3. Store OCR confidence metadata and mark low-confidence pages for review.

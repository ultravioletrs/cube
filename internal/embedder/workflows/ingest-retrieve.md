# Embedder Ingest/Retrieve Workflow

This runbook verifies the Cube Embedder end-to-end flow:

1. authenticate user
2. upload document
3. wait for indexing
4. run retrieval query

## Prerequisites

- Docker and Docker Compose installed
- Cube stack running (`make up`)
- A test file available locally (for example `sample.txt` or `sample.pdf`)

## 1) Start Cube Stack

From repository root:

```bash
make up
```

Embedder should be reachable at:

```bash
curl -s http://localhost:8080/health
```

Expected response:

```json
{"status":"ok"}
```

## 2) Get Access Token

Issue a token from ATOM:

```bash
TOKEN=$(
  curl -sS -X POST http://localhost:8080/auth/login \
    -H "Content-Type: application/json" \
    -d '{"identifier":"00000000-0000-0000-0000-000000000001","secret":"12345678","kind":"password"}' \
  | sed -n 's/.*"token":"\([^"]*\)".*/\1/p'
)

echo "${TOKEN:0:24}..."
```

## 3) Upload a Record

Direct upload automatically creates/uses the internal "Direct Uploads" source.

```bash
UPLOAD_RESP=$(
  curl -sS -X POST http://localhost:8080/api/v1/records/upload \
    -H "Authorization: Bearer $TOKEN" \
    -F "file=@./sample.txt" \
    -F "name=Sample Notes"
)

echo "$UPLOAD_RESP"
RECORD_ID=$(echo "$UPLOAD_RESP" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
echo "record_id=$RECORD_ID"
```

## 4) Wait Until Record Is Indexed

Poll the record status until `indexed` (or `failed`):

```bash
while true; do
  STATUS=$(curl -sS -H "Authorization: Bearer $TOKEN" \
    "http://localhost:8080/api/v1/records/$RECORD_ID" \
    | sed -n 's/.*"status":"\([^"]*\)".*/\1/p')
  echo "status=$STATUS"
  [ "$STATUS" = "indexed" ] && break
  [ "$STATUS" = "failed" ] && break
  sleep 2
done
```

If status is `failed`, inspect service logs:

```bash
docker compose -f docker/compose.yaml --env-file docker/.env logs -f cube-embedder
```

## 5) Run Retrieval Query

```bash
curl -sS -X POST http://localhost:8080/api/v1/retrieve \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Summarize key points",
    "top_k": 5
  }'
```

Expected result shape:

```json
{
  "chunks": [
    {
      "record_id": "...",
      "record_name": "...",
      "chunk_index": 0,
      "content": "..."
    }
  ]
}
```

## Optional Checks

- List records:

```bash
curl -sS -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/records
```

- List sources:

```bash
curl -sS -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/sources
```

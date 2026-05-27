# Cube Image Embedder

Small HTTP sidecar for visual image embeddings.

The current provider is deterministic and intended for local end-to-end plumbing tests. It returns stable vectors for identical image bytes, which lets the Go embedder, migrations, and storage path be tested without requiring GPU inference or downloading model weights.

## API

- `GET /health`
- `POST /embed-image`

```json
{
  "image_base64": "...",
  "mime_type": "image/png",
  "model": "deterministic-image-test",
  "dimensions": 512
}
```

The response is:

```json
{
  "embedding": [0.01, -0.02],
  "model": "deterministic-image-test",
  "dimensions": 512
}
```

The next production step is replacing the deterministic provider with an OpenCLIP/SigLIP backend behind the same endpoint.

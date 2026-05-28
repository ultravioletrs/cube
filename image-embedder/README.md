# Cube Image Embedder

Small HTTP sidecar for visual image embeddings.

The default provider is OpenCLIP `ViT-B-32` with `laion2b_s34b_b79k` pretrained weights. It returns normalized 512-dimensional image embeddings.

The deterministic provider is still available for local plumbing tests. It returns stable vectors for identical image bytes, which lets the Go embedder, migrations, and storage path be tested without requiring model inference or downloading weights.

## Configuration

| Variable | Description | Default |
| --- | --- | --- |
| `IMAGE_EMBEDDER_MODEL` | Model exposed by the service. Use `openclip-vit-b-32` or `deterministic-image-test`. | `openclip-vit-b-32` |
| `IMAGE_EMBEDDER_DIMENSIONS` | Returned vector dimensions. OpenCLIP ViT-B-32 supports `512`. | `512` |
| `IMAGE_EMBEDDER_OPENCLIP_BACKBONE` | OpenCLIP backbone name. | `ViT-B-32` |
| `IMAGE_EMBEDDER_OPENCLIP_PRETRAINED` | OpenCLIP pretrained weights name. | `laion2b_s34b_b79k` |
| `IMAGE_EMBEDDER_DEVICE` | `auto`, `cpu`, or `cuda`. | `auto` |

## API

- `GET /health`
- `POST /embed-image`

```json
{
  "image_base64": "...",
  "mime_type": "image/png",
  "model": "openclip-vit-b-32",
  "dimensions": 512
}
```

The response is:

```json
{
  "embedding": [0.01, -0.02],
  "model": "openclip-vit-b-32",
  "dimensions": 512
}
```

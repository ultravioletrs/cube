import base64
import hashlib
import math
import os
from typing import List, Optional

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field


DEFAULT_DIMENSIONS = int(os.getenv("IMAGE_EMBEDDER_DIMENSIONS", "512"))
DEFAULT_MODEL = os.getenv("IMAGE_EMBEDDER_MODEL", "deterministic-image-test")

app = FastAPI(title="Cube Image Embedder")


class EmbedImageRequest(BaseModel):
    image_base64: str = Field(..., min_length=1)
    mime_type: Optional[str] = None
    model: Optional[str] = None
    dimensions: Optional[int] = Field(default=None, ge=1, le=4096)


class EmbedImageResponse(BaseModel):
    embedding: List[float]
    model: str
    dimensions: int


@app.get("/health")
def health() -> dict:
    return {"status": "ok", "model": DEFAULT_MODEL, "dimensions": DEFAULT_DIMENSIONS}


@app.post("/embed-image", response_model=EmbedImageResponse)
def embed_image(req: EmbedImageRequest) -> EmbedImageResponse:
    try:
        image = base64.b64decode(req.image_base64, validate=True)
    except Exception as exc:
        raise HTTPException(status_code=400, detail="invalid image_base64") from exc
    if not image:
        raise HTTPException(status_code=400, detail="empty image")

    dims = req.dimensions or DEFAULT_DIMENSIONS
    model = req.model or DEFAULT_MODEL
    vector = deterministic_embedding(image, dims)
    return EmbedImageResponse(embedding=vector, model=model, dimensions=dims)


def deterministic_embedding(image: bytes, dims: int) -> List[float]:
    """Stable test embedding used to validate plumbing without a GPU model."""
    values = []
    seed = hashlib.sha256(image).digest()
    counter = 0
    while len(values) < dims:
        digest = hashlib.sha256(seed + counter.to_bytes(8, "big")).digest()
        for i in range(0, len(digest), 4):
            raw = int.from_bytes(digest[i : i + 4], "big")
            values.append((raw / 0xFFFFFFFF) * 2.0 - 1.0)
            if len(values) == dims:
                break
        counter += 1

    norm = math.sqrt(sum(v * v for v in values)) or 1.0
    return [v / norm for v in values]

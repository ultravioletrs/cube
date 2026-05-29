# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

import base64
import hashlib
import io
import math
import os
import threading
from dataclasses import dataclass, field
from typing import Any, List, Optional

from fastapi import FastAPI, HTTPException
from PIL import Image, UnidentifiedImageError
from pydantic import BaseModel, Field


DEFAULT_DIMENSIONS = int(os.getenv("IMAGE_EMBEDDER_DIMENSIONS", "512"))
DEFAULT_MODEL = os.getenv("IMAGE_EMBEDDER_MODEL", "openclip-vit-b-32")
DETERMINISTIC_MODEL = "deterministic-image-test"
OPENCLIP_MODEL = "openclip-vit-b-32"
OPENCLIP_BACKBONE = os.getenv("IMAGE_EMBEDDER_OPENCLIP_BACKBONE", "ViT-B-32")
OPENCLIP_PRETRAINED = os.getenv("IMAGE_EMBEDDER_OPENCLIP_PRETRAINED", "laion2b_s34b_b79k")
OPENCLIP_DEVICE = os.getenv("IMAGE_EMBEDDER_DEVICE", "auto")

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
    return {
        "status": "ok",
        "model": DEFAULT_MODEL,
        "dimensions": DEFAULT_DIMENSIONS,
        "openclip_backbone": OPENCLIP_BACKBONE,
        "openclip_pretrained": OPENCLIP_PRETRAINED,
    }


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
    vector = embed(model, image, dims)
    return EmbedImageResponse(embedding=vector, model=model, dimensions=dims)


def embed(model: str, image: bytes, dims: int) -> List[float]:
    if model == DETERMINISTIC_MODEL:
        return deterministic_embedding(image, dims)
    if model == OPENCLIP_MODEL:
        return openclip_provider().embed(image, dims)
    raise HTTPException(status_code=400, detail=f"unsupported model: {model}")


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


@dataclass
class OpenCLIPProvider:
    model_name: str
    pretrained: str
    device: str
    dimensions: int = 512
    _loaded: bool = False
    _lock: threading.Lock = field(default_factory=threading.Lock)
    model: Any = None
    preprocess: Any = None

    def embed(self, image: bytes, dims: int) -> List[float]:
        if dims != self.dimensions:
            raise HTTPException(
                status_code=400,
                detail=f"{OPENCLIP_MODEL} supports {self.dimensions} dimensions, got {dims}",
            )
        pil_image = decode_image(image)
        self._load()

        import torch

        with torch.no_grad():
            tensor = self.preprocess(pil_image).unsqueeze(0).to(self.device)
            features = self.model.encode_image(tensor)
            features = features / features.norm(dim=-1, keepdim=True)
        return features.squeeze(0).detach().cpu().tolist()

    def _load(self) -> None:
        if self._loaded:
            return
        with self._lock:
            if self._loaded:
                return
            import open_clip
            import torch

            self.device = resolve_device(self.device, torch)
            model, _, preprocess = open_clip.create_model_and_transforms(
                self.model_name,
                pretrained=self.pretrained,
                device=self.device,
            )
            model.eval()
            self.model = model
            self.preprocess = preprocess
            self._loaded = True


def decode_image(image: bytes) -> Image.Image:
    try:
        with Image.open(io.BytesIO(image)) as img:
            return img.convert("RGB")
    except UnidentifiedImageError as exc:
        raise HTTPException(status_code=400, detail="invalid image bytes") from exc


def resolve_device(device: str, torch_module) -> str:
    if device != "auto":
        return device
    return "cuda" if torch_module.cuda.is_available() else "cpu"


_openclip_provider: Optional[OpenCLIPProvider] = None
_openclip_lock = threading.Lock()


def openclip_provider() -> OpenCLIPProvider:
    global _openclip_provider
    if _openclip_provider is None:
        with _openclip_lock:
            if _openclip_provider is None:
                _openclip_provider = OpenCLIPProvider(
                    model_name=OPENCLIP_BACKBONE,
                    pretrained=OPENCLIP_PRETRAINED,
                    device=OPENCLIP_DEVICE,
                )
    return _openclip_provider

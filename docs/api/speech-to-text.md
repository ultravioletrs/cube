---
id: speech-to-text
title: Speech to Text
---

Cube AI supports OpenAI-compatible **speech-to-text** transcription for audio inputs,
depending on the enabled backend and available models.

> ⚠️ Not all Cube AI deployments enable speech-to-text by default.

---

## Endpoint

```http
POST /proxy/{domain_id}/v1/audio/transcriptions
```

---

## Example Request

```bash
curl -k https://localhost/proxy/<domain_id>/v1/audio/transcriptions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: multipart/form-data" \
  -F "file=@audio.wav" \
  -F "model=whisper"
```

---

## Request Parameters

- `file` — Audio file to transcribe (e.g. WAV, MP3)
- `model` — Speech-to-text model (for example `whisper`)

---

## Response

Returns a transcription result compatible with the OpenAI API format.

---

## Notes

- Audio is processed inside a **Trusted Execution Environment (TEE)**
- Availability depends on backend configuration
- Large audio files may increase latency

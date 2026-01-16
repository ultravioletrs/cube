---
id: embeddings
title: Embeddings & RAG
---

Large Language Models (LLMs) are powerful, but they **do not know your private or internal data**.
They are trained on public information and cannot access your documents, databases, or source
code unless you explicitly provide that context.

This is the problem that **Embeddings** and **Retrieval-Augmented Generation (RAG)** address.

> **Cube AI scope**
>
> Cube AI provides **secure embeddings generation and LLM execution inside Trusted Execution Environments (TEEs)**.
> It does **not** provide a built-in vector database or retrieval layer.
> In a RAG setup, storing embeddings and retrieving relevant context is handled
> by the application or integration layer outside of Cube AI.

---

## The Problem LLMs Have

Without embeddings and RAG:

- LLMs cannot answer questions about private data
- responses are often generic or inaccurate
- models may hallucinate answers
- updating knowledge requires retraining (slow and expensive)

---

## What Are Embeddings?

An **embedding** is a numerical (vector) representation of text that captures its meaning.

Embeddings allow applications using Cube AI to:

- compare text by semantic similarity
- search documents by meaning instead of keywords
- retrieve relevant context for LLM prompts

In simple terms:

> Embeddings make text **searchable by meaning**, not just by keywords.

Cube AI embeddings are generated inside **Trusted Execution Environments (TEEs)**,
ensuring that both input text and resulting vectors remain confidential.

---

## What Is RAG?

**Retrieval-Augmented Generation (RAG)** is an architectural pattern where:

1. Data is converted into embeddings
2. Relevant content is retrieved using a vector database
3. Retrieved content is injected into an LLM prompt
4. The LLM generates an answer grounded in that context

RAG allows LLMs to answer questions using **your data**, rather than guessing.

---

## How RAG Works with Cube AI

![RAG flow with Cube AI](/img/rag-flow.png)

A typical RAG flow using Cube AI looks like this:

1. Documents are split into chunks by the application
2. Each chunk is converted into an embedding using Cube AI
3. Embeddings are stored in a vector database (outside of Cube AI)
4. A user asks a question
5. The application retrieves the most relevant chunks
6. Cube AI generates an answer using the retrieved context

LLM inference and embeddings generation stay inside your Cube AI deployment.

---

## Why Use RAG with Cube AI?

Using RAG with Cube AI enables:

- chat over internal documentation
- question answering over PDFs and files
- AI assistants for support and operations
- safer and more accurate LLM responses
- no data leakage to external model providers

---

## Common Use Cases

### Internal Documentation Assistant

Ask questions about internal docs, wikis, or README files.

### Support & Helpdesk Bots

Answer customer questions using company knowledge bases.

### Codebase Search

Query large repositories using natural language.

### Knowledge-Based AI Assistants

Build enterprise-grade assistants backed by private data.

---

## Embeddings API Reference

The embeddings endpoint allows you to generate vector representations of text.
These vectors can be used for semantic search, clustering, retrieval-augmented
generation (RAG), and similarity comparisons.

---

### Endpoint

```http
POST /proxy/{domain_id}/v1/embeddings
```

---

### Example Request

```bash
curl -k https://localhost/proxy/<domain_id>/v1/embeddings \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "nomic-embed-text:v1.5",
    "input": "Cube AI embeddings"
  }'
```

---

### Response

Returns an OpenAI-compatible `embeddings` response object containing one or more
embedding vectors.

---

### Notes

- Embeddings are **domain-scoped**
- Input text is processed securely inside a TEE
- Cube AI does **not** store embeddings or perform retrieval
- Use embedding models such as `nomic-embed-text` for best results

---

## Next Steps

- Combine Embeddings with **Chat Completions**
- Explore available **Models**
- Build a complete RAG pipeline using Cube AI integrations

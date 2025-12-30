---
id: architecture
title: Architecture
sidebar_position: 3
---

Cube AI is built on a secure, scalable architecture designed to run Large Language Models (LLMs) inside **Trusted Execution Environments (TEEs)** while providing isolation between tenants, strong authentication, flexible backend support (Ollama and vLLM), and a unified API surface.

Below is the architecture diagram created by the team:

![Cube AI architecture overview](/img/cube-ai-architecture.png)

---

## Core Components

Cube AI consists of five primary components:

1. **SuperMQ Services**  
   - Users Service  
   - Auth Service  
   - Domains Service  

2. **Cube Proxy**  
   - Secure request gateway  
   - Domain-based routing  
   - TEE access enforcement  
   - Token validation  

3. **LLM Backend**  
   - Ollama  
   - vLLM  

4. **Trusted Execution Environment (TEE)**  
   - Protects models  
   - Protects prompts and responses  
   - Ensures confidentiality and integrity  

5. **AI Guardrails**  
   - Authentication and authorization  
   - Domain isolation  
   - Secure inference enforcement  
   - Auditing and access control  

---

## 1. SuperMQ (Users, Auth, Domains)

Cube AI uses SuperMQ’s microservices as its identity and tenant-management layer.

### ✔ Users Service

Stores user accounts, profile data, and associated metadata.

### ✔ Auth Service

Issues JWT access tokens and validates them.  
Cube Proxy uses this service to authenticate every request.

### ✔ Domains Service

Each **domain** represents an isolated tenant.  
Models, permissions, and policies are scoped per domain.

### Why this matters

SuperMQ allows Cube AI to remain fully multi-tenant, scalable, and secure without duplicating identity logic.

---

## 2. Cube Proxy

The **Cube Proxy** is the central entry point for all traffic.

It is responsible for:

- Verifying JWT tokens using the SuperMQ Auth Service  
- Checking user permissions and domain membership  
- Routing requests to the correct backend based on domain configuration  
- Enforcing that all inference requests are executed **inside a Trusted Execution Environment**  
- Normalizing requests to an OpenAI-compatible API shape  

This component ensures that **no user ever interacts directly with Ollama or vLLM**, and that all inference happens under strict security controls.

---

## 3. LLM Backends (Ollama & vLLM)

Cube AI supports two interchangeable inference engines.

### ✔ Ollama

- Lightweight, local-friendly model runner  
- Ideal for development or smaller environments  
- Fast model switching and easy model packaging  

### ✔ vLLM

- High-performance CUDA-accelerated inference  
- Continuous batching for high throughput  
- Supports larger models and production workloads  

The backend used depends on the domain configuration and available hardware.

---

## 4. Trusted Execution Environment (TEE)

The TEE is the foundation of Cube AI security.

### What the TEE protects

- **User prompts**
- **Model weights**
- **Threaded execution**
- **Intermediate state**
- **Responses before leaving the enclave**

### Key guarantees

- **Confidentiality:**  
  No external process (hypervisor, cloud operator, root user) can read memory inside the enclave.

- **Integrity:**  
  Code inside the enclave cannot be modified without detection.

- **Remote Attestation:**  
  Clients can verify that the inference occurred inside an authentic TEE with approved measurements.

This makes Cube AI fundamentally different from traditional LLM deployments.

---

## AI Guardrails and Inference Backends

Cube AI enforces a strict separation between **platform security controls**
and **model execution engines**.

- **AI Guardrails** define how models are accessed, isolated, and executed  
  → See [AI Guardrails](./guardrails.md)

- **vLLM** is used as a high-performance inference backend for production workloads  
  → See [vLLM Backend](./vllm.md)

These components work together to ensure that all inference is secure,
auditable, and isolated per domain.

---

## End-to-End Request Flow

Below is the simplified flow of a request inside Cube AI:

1. **Client UI / API Client / Continue (VS Code)**  
2. → **Cube Proxy**  
   - verifies token  
   - loads domain configuration  
   - validates permissions  
3. → **TEE**  
   - secures model execution  
   - protects inputs and outputs  
4. → **LLM Backend** (Ollama or vLLM)  
5. → **TEE**  
6. → **Cube Proxy**  
7. → **Client**

This pipeline ensures all sensitive operations remain inside the secure environment.

---

## Summary

Cube AI combines:

- SuperMQ for identity and domain management  
- A secure Cube Proxy for routing, authorization, and TEE enforcement  
- Flexible backend support (Ollama, vLLM)  
- AI Guardrails for platform-level security controls  
- Trusted Execution Environments for confidential LLM execution  

Together, these components create a secure, scalable, multi-tenant platform for running LLMs with full data protection guarantees.

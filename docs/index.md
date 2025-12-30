---
id: intro
title: Overview
sidebar_position: 1
---

## Overview

Cube AI is a secure framework for running and serving GPT-based applications using confidential computing.  
It deploys Large Language Models (LLMs) inside hardware-protected Trusted Execution Environments (TEEs)  
to ensure that both user data and model execution remain private and tamper-resistant.

A TEE is a hardware-isolated environment that prevents external systems—including the host OS, hypervisor,  
and cloud operator—from accessing or modifying the data and code running inside it. Cube AI uses this  
technology to provide strong confidentiality and integrity guarantees for AI workloads.

---

## Key Features

- **Confidential LLM Inference**: All prompts, responses, and model weights are processed inside a hardware-protected enclave.
- **Trusted Execution Environments (TEEs)**: Built-in support for AMD SEV-SNP and Intel TDX provides strong isolation and data protection.
- **Scalability**: Designed to support small local deployments as well as large multi-tenant production environments.
- **Multiple LLM Backends**: Supports both Ollama and vLLM for flexible, high-performance model serving.
- **OpenAI-Compatible API**: Integrates easily with existing applications through familiar API endpoints.

---

## Supported LLM Backends

### vLLM Integration

Cube AI includes support for **vLLM**, a high-performance inference engine optimized for GPU workloads.

vLLM offers:

- **High Throughput**: Efficient batching and parallel request handling  
- **Memory Efficiency**: Advanced memory management for large models  
- **Fast GPU Inference**: Optimized CUDA execution  
- **Broad Model Support**: Works with LLaMA, Mistral, Qwen, and other major architectures  

### Ollama Integration

For local or lightweight deployments, Cube AI also integrates with **Ollama**, providing:

- Simple model setup and management  
- Local inference without external dependencies  
- Support for a wide range of open-source models  

---

## Why Cube AI?

Public cloud LLM services require trusting the provider with both your data and your model execution environment.  
In many cases, users have limited visibility into how prompts, responses, and model weights are handled.

Cube AI eliminates this trust dependency.

By running models inside TEEs, Cube AI ensures that:

- User data cannot be accessed by the host system or cloud operator  
- Model weights remain protected inside the enclave  
- Execution is verifiable and tamper-resistant  

This makes Cube AI suitable for privacy-sensitive domains including:

- Finance  
- Healthcare  
- Government  
- Enterprise AI  

---

## How Does Cube AI Work?

Cube AI processes each request inside a hardware-isolated secure enclave:

1. A user sends a prompt to Cube AI  
2. The request enters the TEE  
3. The model executes privately inside the enclave  
4. Only the final response leaves the secure environment  

This architecture ensures **confidentiality**, **integrity**, and **end-to-end protection** of AI workloads.

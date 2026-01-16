---
id: attestation
title: Attestation
sidebar_position: 4
---

## Attestation

Attestation is a security process used to verify that a system is running trusted software inside a trusted environment.  
In Cube AI, attestation ensures that the confidential computing environment (CVM) has not been tampered with before Cube AI services start running.

Cube AI performs attestation both **internally** during the boot sequence of the confidential virtual machine (CVM) and via a **user-facing endpoint** that allows clients to verify trust guarantees.

---

## How Cube AI Uses Attestation Today

Cube AI runs inside **hardware-backed Confidential Virtual Machines (CVMs)** based on **AMD SEV-SNP and Intel TDX**.

These confidential computing technologies provide:

- Full memory encryption  
- Protection from a malicious hypervisor or cloud provider  
- Integrity verification of the VM state  
- Hardware-enforced measurement of what software boots inside the VM  

During startup:

1. The CVM performs **hardware attestation** using platform-specific attestation mechanisms (**AMD SEV-SNP or Intel TDX**).
2. Measurements are validated internally before Cube AI components start.
3. If the environment does not match expected measurements, the system refuses to start.

Because this attestation happens before any workloads run, Cube AI ensures that:

- The AI backend (Ollama or vLLM) is running on trusted code  
- The proxy and API services are verified  
- Sensitive data and model prompts are protected from host-level access  

---

## Why Attestation Matters

Confidential Computing provides strong isolation, but attestation ensures:

- The VM truly has confidential computing protections enabled  
- The system hasn’t been modified by the cloud provider  
- No unauthorized code is executed inside the confidential environment  
- All model prompts and responses remain protected even from the host  

This is essential for:

- Enterprise deployments  
- Regulated industries (healthcare, finance, defense, R&D)  
- Zero-trust security architectures  
- Protecting sensitive AI inputs and outputs  

---

## User-Facing Attestation

Cube AI exposes a user-facing attestation endpoint that allows applications
to verify that they are interacting with a trusted Cube AI instance.

### 🔹 Attestation Endpoint

Applications can request an attestation report using:

```bash
POST /attestation
```

The returned attestation data can be validated by the client to ensure that the
Cube AI instance is running inside a trusted confidential computing environment.

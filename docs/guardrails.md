---
id: guardrails
title: AI Guardrails
---

AI Guardrails in Cube AI define the **security, isolation, and control mechanisms**
applied around Large Language Model (LLM) inference.

Their purpose is to ensure that LLM usage is **safe, auditable, and predictable**
in enterprise environments.

Guardrails are not about changing how models think — they are about controlling
**how models are accessed and used**.

---

## Cube AI Scope

> **Cube AI scope**
>
> Cube AI guardrails operate at the **platform level**.
> They control access, isolation, and execution of models, but do **not**
> modify model weights, prompts, or training behavior.

Cube AI does **not**:

- train models
- fine-tune models
- alter model outputs for ethical or policy reasons

---

## What Guardrails Do

![Cube AI guardrails overview](/img/cube-ai-guardrails.png)
This diagram shows how Cube AI guardrails enforce security and isolation at the
platform level without interfering with application logic or model behavior.

Cube AI guardrails provide:

### Authentication & Authorization

- token-based access control
- domain-scoped permissions
- per-domain model visibility

### Domain Isolation

- strict separation between domains
- no data or model leakage across domains
- isolated execution contexts

### Request Validation

- validation of incoming requests
- enforcement of API contracts
- rejection of malformed or unauthorized calls

### Model Access Control

- control which models are available per domain
- backend-specific model exposure
- prevention of unauthorized model usage

### Secure Execution (TEE)

- all inference runs inside **Trusted Execution Environments**
- prompts and outputs remain confidential
- memory isolation during execution

### Auditing & Observability (Optional)

- request metadata logging
- traceability of inference calls
- integration with audit pipelines when enabled

---

## What Guardrails Do NOT Do

To avoid confusion, Cube AI guardrails do **not**:

- rewrite or sanitize prompts
- filter or censor model outputs
- implement AI ethics policies
- perform content moderation or alignment
- replace application-level safety logic

Guardrails ensure **platform safety**, not application behavior.

---

## Why Guardrails Matter

Without guardrails, LLM deployments risk:

- unauthorized access
- data leakage between tenants
- untraceable model usage
- unpredictable behavior in production

Cube AI guardrails make LLM usage suitable for:

- enterprise deployments
- multi-tenant environments
- regulated industries
- confidential workloads

---

## Relationship to Applications

Guardrails complement — but do not replace — application-level controls.

Applications are still responsible for:

- prompt design
- output validation
- business logic enforcement
- user-facing safety mechanisms

Cube AI ensures the **infrastructure layer is secure and controlled**.

---

## Next Steps

- Learn how Cube AI executes models using **vLLM**
- Explore **Models** to understand backend configuration
- Use **Chat Completions** with guardrails enabled

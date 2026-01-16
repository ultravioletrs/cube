---
id: auth-and-request-flow
title: Authentication & Request Flow
---

This document explains how requests flow through Cube AI,
including authentication, authorization, and routing to LLM backends.

## Overview

![Cube AI auth and proxy flow](/img/cube-ai-auth-flow.png)

## Flow Description

1. Client requests enter through Traefik.
2. Cube Proxy intercepts all incoming requests.
3. Authentication and authorization are handled via SuperMQ Auth.
4. Authorized requests are routed to the configured LLM backend (e.g. Ollama).
5. UI traffic is routed separately to Cube AI UI and SuperMQ Users services.

This flow ensures that all inference requests are authenticated, authorized,
and securely routed before reaching any model backend.

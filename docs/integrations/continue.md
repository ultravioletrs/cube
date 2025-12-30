---
id: continue
title: Continue for VS Code
sidebar_position: 1
---

## Continue Integration for VS Code

> **Cube AI scope**
>
> Cube AI acts as a **secure LLM backend** responsible for authentication,
> model routing, and model execution.
> It does **not** manage editor state, files, or code context — those are handled
> locally by the Continue extension running inside VS Code.

The **Continue** extension brings Cube AI’s LLM capabilities directly into
Visual Studio Code, enabling inline completions, refactoring help, and
chat-based assistance.

This guide explains how to connect Continue with a Cube AI domain.

---

## 1. Install Requirements

1. Install **Visual Studio Code**  
   [https://code.visualstudio.com](https://code.visualstudio.com)

2. Install the **Continue** extension  
   [https://www.continue.dev](https://www.continue.dev)

   ![Continue VS Code Marketplace](/img/continue-marketplace.png)

---

## 2. Open Continue Configuration

In Visual Studio Code:

1. Click the **Continue** icon  
2. Open the **Settings / gear** menu  
3. Select **Configure Continue**

![Continue configuration](/img/continue-config.png)

This opens the configuration file:

```yaml
.continue/config.yaml
```

---

## 3. Configure Continue to Use Cube AI

Replace the contents of `config.yaml` with the configuration below.

Before editing the file, make sure you have generated a Cube AI access token.
You can obtain it from the Cube AI UI under **Profile → Tokens**.

```yaml
name: Cube AI
version: "1.0.0"
schema: v1

models:
  - name: tinyllama
    provider: ollama
    model: tinyllama:1.1b
    apiKey: <access_token>
    apiBase: https://<your-cube-instance>/proxy/<your-domain-id>
    requestOptions:
      verifySsl: false

  - name: starcoder2
    provider: ollama
    model: starcoder2:7b
    apiKey: <access_token>
    apiBase: https://<your-cube-instance>/proxy/<your-domain-id>
    requestOptions:
      verifySsl: false

context:
  - provider: code
  - provider: terminal
  - provider: diff
  - provider: folder
  - provider: problems
  - provider: docs
```

### Replace

- `<access_token>` → your Cube AI access token  
- `<your-cube-instance>` → usually `localhost`  
- `<your-domain-id>` → the domain ID you want VS Code to use  

> `verifySsl: false` should be used **only for local development**.

---

## 4. Using Continue With Cube AI

![Continue chat example](/img/continue-chat.png)

Once configured:

- Press **Ctrl + L** to open the Continue chat  
- Ask questions or request explanations  
- Use inline completions powered by Cube AI models  

Example prompts:

- Explain this function  
- Refactor this TypeScript file  
- Write unit tests for this module  

---

## 5. Troubleshooting

### Connection issues

- Ensure Cube AI is running (`make up`)  
- Verify that the domain exists  
- Check that your access token is valid  

### SSL issues

If you are running Cube AI locally without valid TLS certificates, set:

```yaml
requestOptions:
  verifySsl: false
```

For production deployments, always use valid TLS certificates.

---

## 6. Video Tutorial

A complete walkthrough is available here:  
[https://www.youtube.com/watch?v=BGpv_iTB2NE](https://www.youtube.com/watch?v=BGpv_iTB2NE)

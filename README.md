# Cube AI

Cube AI is a framework for building GPT-based AI applications using confidential computing. It protects user data and the AI model by using a trusted execution environment (TEE). TEE is a secure area of a processor that ensures that code and data loaded inside it are protected with respect to confidentiality and integrity. Data confidentiality prevents unauthorized access of data from outside the TEE, while code integrity ensures that code inside the TEE remains unchanged and unaltered from unauthorized access.

<p align="center">
  <img src="https://github.com/ultravioletrs/cube-docs/blob/main/docs/img/cube-ai.png">
</p>

## Key Features

- **Secure Computing**: Cube AI uses secure enclaves to protect user data and AI models from unauthorized access.
- **Trusted Execution Environment (TEE)**: Cube AI uses a trusted execution environment to ensure that AI models are executed securely and in a controlled environment.
- **Scalability**: Cube AI can handle large amounts of data and AI models, making it suitable for applications that require high performance and scalability.
- **Multiple LLM Backend Support**: Supports both Ollama and vLLM for flexible model deployment and high-performance inference.
- **OpenAI-Compatible API**: Provides familiar API endpoints for easy integration with existing applications.

## Supported LLM Backends

### vLLM Integration

Cube AI now supports vLLM, a high-throughput and memory-efficient inference engine for Large Language Models. vLLM provides:

- **High Throughput**: Optimized for serving multiple concurrent requests with continuous batching
- **Memory Efficiency**: Advanced memory management techniques for large models
- **Fast Inference**: Optimized CUDA kernels and efficient attention mechanisms
- **Model Compatibility**: Supports popular architectures including LLaMA, Mistral, Qwen, and more

### Ollama Integration

Cube AI also integrates with Ollama for local model deployment, providing:

- Easy model management and deployment
- Local inference capabilities
- Support for various open-source models

## Why Cube AI?

Traditional GPT-based AI applications often rely on public cloud services, which can be vulnerable to security breaches and unauthorized access. The tenant for example openai, and the hardware provider for example Azure, are not always transparent about their security practices and can be easily compromised. They can also access your prompts and model responses. Cube AI addresses these privacy concerns by using TEEs. Using TEEs, Cube AI ensures that user data and AI models are protected from unauthorized access outside the TEE. This helps to maintain user privacy and ensures that AI models are used in a controlled and secure manner.

## How does Cube AI work?

Cube AI uses TEEs to protect user data and AI models from unauthorized access. TEE offers an execution space that provides a higher level of security for trusted applications running on the device. In Cube AI, the TEE ensures that AI models are executed securely and in a controlled environment.

## Getting Started

### Prerequisites

- Docker and Docker Compose
- NVIDIA GPU with CUDA support (recommended for vLLM)
- Hardware with TEE support (AMD SEV-SNP or Intel TDX)

### Quick Start

1. **Clone the repository**
   ```bash
   git clone https://github.com/ultravioletrs/cube.git
   cd cube
   ```

2. **Start Cube AI services**
   ```bash
   make up
   ```

3. **Get your authentication token**

   All API requests require authentication using JWT tokens. Once the services are running, obtain a JWT token:
   ```bash
   curl -ksSiX POST https://localhost/users/tokens/issue \
     -H "Content-Type: application/json" \
     -d '{
       "username": "admin@example.com",
       "password": "m2N2Lfno"
     }'
   ```
   
   The response will contain your JWT token:
   ```json
   {
     "access_token": "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9...",
     "refresh_token": "..."
   }
   ```

4. **Create a domain**

   All API requests require a domain ID in the URL path. You can either get the domain ID from the UI or create a new domain via the API:

   ```bash
   curl -ksSiX POST https://localhost/domains \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     -d '{
       "name": "Magistrala",
       "route": "magistrala1",
       "tags": ["absmach", "IoT"],
       "metadata": {
         "region": "EU"
       }
     }'
   ```
   
   The response will contain your domain information including the `id`:
   ```json
   {
     "id": "d7f9b3b8-4f7e-4f44-8d47-1a6e5e6f7a2b",
     "name": "Magistrala",
     "route": "magistrala",
     "tags": ["absmach", "IoT"],
     "metadata": {
       "region": "EU"
     },
     "status": "enabled",
     "created_by": "c8c3e4f1-56b2-4a22-8e5f-8a77b1f9b2f4",
     "created_at": "2025-10-29T14:12:01Z",
     "updated_at": "2025-10-29T14:12:01Z"
   }
   ```
   
   **Notes:**
   - `name` and `route` are required fields
   - `route` must be unique and cannot be changed after creation
   - `metadata` must be a valid JSON object
   - The `id` is automatically generated if not provided
   - Save the `id` value as you'll need it for all subsequent API requests

5. **Verify the installation**
   
   List available models (replace `YOUR_DOMAIN_ID` with the domain ID from step 4):
   ```bash
   curl -k https://localhost/proxy/YOUR_DOMAIN_ID/v1/models \
     -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
   ```

6. **Make your first AI request**
   
   Replace `YOUR_DOMAIN_ID` with your actual domain ID:
   ```bash
   curl -k https://localhost/proxy/YOUR_DOMAIN_ID/v1/chat/completions \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     -d '{
       "model": "tinyllama:1.1b",
       "messages": [
         {
           "role": "user",
           "content": "Hello! How can you help me today?"
         }
       ]
     }'
   ```

### API Endpoints

Cube AI exposes all services through Traefik reverse proxy. 

All protected endpoints require the `Authorization: Bearer <token>` header with a valid JWT token.

#### Proxy Endpoints (OpenAI-Compatible)

**Base URL:** `https://localhost/proxy/`

Replace `{domainID}` with your actual domain ID obtained from step 4 in the Getting Started guide.

### OpenAI-Compatible Endpoints
| Method | Path                              | Description             |
|--------|-----------------------------------|-------------------------|
| GET    | `/{domainID}/v1/models`           | List available models   |
| POST   | `/{domainID}/v1/chat/completions` | Create chat completions |
| POST   | `/{domainID}/v1/completions`      | Create text completions |
| GET    | `/{domainID}/api/tags`            | List Ollama models      |
| POST   | `/{domainID}/api/generate`        | Generate completions    |
| POST   | `/{domainID}/api/chat`            | Chat completions        |


**Example:**
```bash
# Using OpenAI-compatible endpoint
curl -k https://localhost/proxy/YOUR_DOMAIN_ID/v1/chat/completions \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"tinyllama:1.1b","messages":[{"role":"user","content":"Hello"}]}'

# Using Ollama API endpoint
curl -k https://localhost/proxy/YOUR_DOMAIN_ID/api/tags \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

#### Auth Endpoints
**Base URL:** `https://localhost/users`

### User Registration & Authentication
| Method | Path                          | Description                            |
|--------|-------------------------------|----------------------------------------|
| POST   | `/users`                      | Register new user account              |
| POST   | `/users/tokens/issue`         | Issue access and refresh token (login) |
| POST   | `/users/tokens/refresh`       | Refresh access token                   |
| POST   | `/password/reset-request`     | Request password reset                 |
| PUT    | `/password/reset`             | Reset password with token              |


**Example**
```bash
curl -ksSiX POST https://localhost/users/tokens/issue \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin@example.com",
    "password": "m2N2Lfno"
  }'
```

---

#### Domains Endpoints
**Base URL:** `https://localhost/domains`

### Domain Management
| Method | Path                          | Description                            |
|--------|-------------------------------|----------------------------------------|
| POST   | `/domains`                    | Create new domain                      |
| GET    | `/domains`                    | List domains with filters              |
| GET    | `/domains/{domainID}`         | Get domain details                     |
| PATCH  | `/domains/{domainID}`         | Update domain name, tags, and metadata |
| POST   | `/domains/{domainID}/enable`  | Enable a domain                        |
| POST   | `/domains/{domainID}/disable` | Disable a domain                       |
| POST   | `/domains/{domainID}/freeze`  | Freeze a domain                        |
 

**Example**
```bash
curl -ksSiX POST https://localhost/domains \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "name": "Magistrala",
    "route": "magistrala1",
    "tags": ["absmach", "IoT"],
    "metadata": {
      "region": "EU"
    }
  }'
```

---

## Configuration

### vLLM Backend

Configure vLLM settings through environment:

```bash
make up-vllm
```

### Ollama Backend

For Ollama integration:

```bash
make up-ollama
```

## Documentation

Project documentation is hosted at [Cube AI docs repository](https://github.com/ultravioletrs/cube-docs).

## License

Cube AI is published under permissive open-source [Apache-2.0](LICENSE) license.

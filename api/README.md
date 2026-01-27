# Cube API Specifications

This directory contains OpenAPI 3.0 specifications for all HTTP endpoints exposed by the Cube proxy and agent services.

## Files

### Proxy Service

#### `proxy-routes.yaml`

REST API for managing dynamic routing rules. This enables runtime creation, retrieval, updating, and deletion of route rules that determine how requests are forwarded to backend services.

**Key Endpoints:**

- `POST /api/routes` - Create a new routing rule
- `GET /api/routes` - List all routing rules
- `GET /api/routes/{name}` - Get a specific routing rule
- `PUT /api/routes/{name}` - Update a routing rule
- `DELETE /api/routes/{name}` - Delete a routing rule

#### `proxy-attestation.yaml`

API for managing attestation policies at the proxy level. Policies define trust and verification requirements for requests.

**Key Endpoints:**

- `POST /attestation/policy` - Update global attestation policy
- `GET /{domainID}/attestation/policy` - Get attestation policy for a domain

#### `proxy-request-forwarding.yaml`

API for forwarding requests to backend services through the proxy with dynamic routing. Supports all HTTP methods (GET, POST, PUT, DELETE, PATCH).

**Key Endpoints:**

- `GET /{domainID}/{path}` - Forward GET request
- `POST /{domainID}/{path}` - Forward POST request
- `PUT /{domainID}/{path}` - Forward PUT request
- `DELETE /{domainID}/{path}` - Forward DELETE request
- `PATCH /{domainID}/{path}` - Forward PATCH request

**Note:** All requests to the agent service (including attestation) are routed through the proxy via the `/{domainID}/{path}` endpoints. The proxy uses dynamic routing rules to determine whether a request should be forwarded to the agent or other backend services.

## Request Routing

### Agent Service Requests

Requests to the agent service follow this flow:

1. Client makes a request to the proxy at `/{domainID}/{path}`
2. Proxy authenticates the request using bearer token authentication
3. Proxy determines the target service using routing rules configured via `/api/routes`
4. If the route matches the agent service, the request is forwarded to the agent
5. Agent processes the request (e.g., attestation at `/attestation`)
6. Response is returned through the proxy to the client

### Supported Agent Operations

The following operations can be performed through the proxy to the agent:

- **Attestation Generation** - Request attestation reports from the agent
  - Path: `POST /{domainID}/attestation`
  - Supports multiple attestation types: SEV, SEV-SNP, TDX, vTPM
  - Returns either JSON or binary attestation reports
- **Request Forwarding** - Any other HTTP request through the agent to further backends
  - Paths: `GET|POST|PUT|DELETE|PATCH /{domainID}/*`

## Excluded Endpoints

The following endpoints are not included in these specifications as they are handled by external services:

- **OpenAI Compatible API** - Requests for LLM inference that follow the OpenAI API specification
- **Ollama Requests** - Requests to the Ollama service for local LLM operations
- **vLLM Requests** - Requests to the vLLM service for inference

These services are proxied transparently through the proxy's routing system but maintain their own API contracts external to Cube.

## Authentication

All proxy service endpoints require Bearer token authentication via JWT tokens. This is enforced by the `bearerAuth` security scheme defined in each specification.

The agent service attestation endpoint does not require authentication and can be called directly.

## Usage

These OpenAPI specifications can be used to:

1. **Generate API Documentation** - Use tools like Swagger UI or ReDoc to generate interactive documentation
2. **Generate Client Libraries** - Use code generation tools like OpenAPI Generator to create client SDKs
3. **API Testing** - Use tools like Postman or Insomnia to import and test the APIs
4. **Server Stub Generation** - Generate server implementations from the specs

### Example: Swagger UI

To view these specs in Swagger UI, serve the files and navigate to:

```bash
https://editor.swagger.io/?url=<path-to-spec>
```

## Contributing

When adding new endpoints or modifying existing ones, ensure to:

1. Update the corresponding OpenAPI specification file
2. Maintain consistency with the actual implementation
3. Document all request/response schemas
4. Include appropriate HTTP status codes and error responses

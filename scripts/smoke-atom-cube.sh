#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

ENV_FILE="$ROOT_DIR/docker/.env"

read_env_default() {
  local key="$1"
  local default_value="$2"
  local value

  value="$(awk -F= -v key="$key" '$1 == key { sub(/^[^=]*=/, ""); print; exit }' "$ENV_FILE" 2>/dev/null | tr -d '\r')"
  if [ -n "$value" ]; then
    printf '%s' "$value"
  else
    printf '%s' "$default_value"
  fi
}

export ATOM_HTTP_PORT="${ATOM_HTTP_PORT:-$(read_env_default ATOM_HTTP_PORT 8080)}"
export ATOM_UI_HTTP_PORT="${ATOM_UI_HTTP_PORT:-$(read_env_default ATOM_UI_HTTP_PORT 3005)}"
export ATOM_GRPC_PORT="${ATOM_GRPC_PORT:-$(read_env_default ATOM_GRPC_PORT 8081)}"
export UV_CUBE_PROXY_PORT="${UV_CUBE_PROXY_PORT:-$(read_env_default UV_CUBE_PROXY_PORT 8900)}"
export EMBEDDER_PORT="${EMBEDDER_PORT:-$(read_env_default EMBEDDER_PORT 8082)}"
export UI_PORT="${UI_PORT:-$(read_env_default UI_PORT 6193)}"

COMPOSE=(docker compose -f docker/compose.yaml --env-file docker/.env --profile default)

"$ROOT_DIR/scripts/generate-atom-dev-certs.sh" "$ROOT_DIR/docker/certs"

if [ "${SMOKE_BUILD_IMAGES:-1}" = "1" ]; then
  export CUBE_PROXY_IMAGE="${CUBE_PROXY_IMAGE:-ghcr.io/ultravioletrs/cube/proxy:latest}"
  export CUBE_AGENT_IMAGE="${CUBE_AGENT_IMAGE:-ghcr.io/ultravioletrs/cube/agent:latest}"
  export CUBE_EMBEDDER_IMAGE="${CUBE_EMBEDDER_IMAGE:-ghcr.io/ultravioletrs/cube/embedder:latest}"
  export CUBE_GUARDRAILS_IMAGE="${CUBE_GUARDRAILS_IMAGE:-ghcr.io/ultravioletrs/cube/guardrails:latest}"

  echo "Building local Cube smoke images"
  docker build --build-arg SVC=proxy -t "$CUBE_PROXY_IMAGE" -f docker/Dockerfile .
  docker build --build-arg SVC=agent -t "$CUBE_AGENT_IMAGE" -f docker/Dockerfile .
  docker build -t "$CUBE_EMBEDDER_IMAGE" -f docker/Dockerfile.embedder .
  docker build -t "$CUBE_GUARDRAILS_IMAGE" -f guardrails/Dockerfile ./guardrails
fi

if [ "${SMOKE_BUILD_ATOM_IMAGE:-1}" = "1" ] && [ -f /home/arvindh/absmach/atom/Dockerfile ]; then
  echo "Building local ATOM smoke image"
  docker build -t ghcr.io/absmach/atom:latest -f /home/arvindh/absmach/atom/Dockerfile /home/arvindh/absmach/atom
fi

wait_http() {
  local name="$1"
  local url="$2"
  local attempts="${3:-60}"
  local delay="${4:-2}"

  for _ in $(seq 1 "$attempts"); do
    if curl --connect-timeout 2 --max-time 5 -fsS "$url" >/dev/null; then
      return 0
    fi
    sleep "$delay"
  done

  echo "$name did not become healthy at $url" >&2
  return 1
}

atom_grpcurl() {
  if command -v grpcurl >/dev/null 2>&1; then
    grpcurl -plaintext \
      -import-path "$ROOT_DIR/proto" -proto atom/v1/atom.proto \
      "$@"
    return
  fi

  docker run --rm --network host \
    -v "$ROOT_DIR/proto:/proto:ro" \
    "${GRPCURL_IMAGE:-fullstorydev/grpcurl:v1.9.3}" \
    -plaintext -import-path /proto -proto atom/v1/atom.proto \
    "$@"
}

echo "Starting ATOM + Cube stack"
"${COMPOSE[@]}" up -d --wait --force-recreate --build atom atom-ui cube-proxy cube-embedder cube-guardrails ui ollama

curl_json() {
  local url="$1"
  curl -fsS "$url" >/dev/null
}

echo "Checking health endpoints"
wait_http "ATOM" "http://localhost:${ATOM_HTTP_PORT:-8080}/health"
wait_http "ATOM UI" "http://localhost:${ATOM_UI_HTTP_PORT:-3005}/"
wait_http "cube-proxy" "http://localhost:${UV_CUBE_PROXY_PORT:-8900}/health"
wait_http "embedder" "http://localhost:${EMBEDDER_PORT:-8082}/health"
wait_http "Cube UI" "http://localhost:${UI_PORT:-5173}/health"
"${COMPOSE[@]}" exec -T cube-guardrails curl -fsS http://localhost:8001/guardrails/health >/dev/null

echo "Checking ATOM CA chain"
curl -fsS "http://localhost:${ATOM_HTTP_PORT:-8080}/certs/ca-chain" -o /tmp/cube-atom-ca-chain.pem
openssl x509 -in /tmp/cube-atom-ca-chain.pem -noout >/dev/null

ATOM_HTTP_URL="http://localhost:${ATOM_HTTP_PORT:-8080}"
ATOM_GRAPHQL_HTTP_URL="${ATOM_HTTP_URL}/graphql"
CUBE_UI_ORIGIN="http://localhost:${UI_PORT:-6193}"

echo "Checking direct ATOM GraphQL without Authorization is blocked"
curl -sS -o /tmp/cube-atom-graphql-missing-session.json \
  -H 'Content-Type: application/json' \
  -H "Origin: ${CUBE_UI_ORIGIN}" \
  -d '{"query":"query { tenants(limit: 1, offset: 0) { total } }"}' \
  "$ATOM_GRAPHQL_HTTP_URL"
if node -e 'const fs=require("fs"); const data=JSON.parse(fs.readFileSync("/tmp/cube-atom-graphql-missing-session.json","utf8")); process.exit(data.errors?.length ? 0 : 1)'; then
  :
else
  echo "expected direct ATOM GraphQL without Authorization to return GraphQL errors" >&2
  cat /tmp/cube-atom-graphql-missing-session.json >&2 || true
  exit 1
fi

echo "Checking unauthenticated cube-proxy runtime route is blocked"
status="$(curl -sS -o /tmp/cube-proxy-missing-session.json -w '%{http_code}' \
  -H 'Content-Type: application/json' \
  "http://localhost:${UV_CUBE_PROXY_PORT:-8900}/api/routes")"
if [ "$status" != "401" ]; then
  echo "expected cube-proxy runtime route without session to return 401, got $status" >&2
  cat /tmp/cube-proxy-missing-session.json >&2 || true
  exit 1
fi

if [ -z "${SMOKE_ATOM_IDENTIFIER:-}" ] || [ -z "${SMOKE_ATOM_PASSWORD:-}" ]; then
  echo "SMOKE_ATOM_IDENTIFIER and SMOKE_ATOM_PASSWORD are not set; skipping login, tenant, and certificate mutation checks."
  echo "Base ATOM + Cube smoke checks passed."
  exit 0
fi

trap 'rm -f /tmp/cube-atom-ca-chain.pem' EXIT

echo "Checking bad GraphQL login returns errors"
SMOKE_LOGIN_IDENTIFIER="$(printf '%s' "$SMOKE_ATOM_IDENTIFIER" | tr '[:upper:]' '[:lower:]')"
if [[ "$SMOKE_LOGIN_IDENTIFIER" =~ ^[a-z0-9@._:-]+$ ]]; then
  "${COMPOSE[@]}" exec -T atom-postgres psql \
    -U "${ATOM_POSTGRES_USER:-atom}" \
    -d "${ATOM_POSTGRES_DB:-atom}" \
    -c "DELETE FROM auth_login_attempts WHERE identifier = '${SMOKE_LOGIN_IDENTIFIER}' AND tenant_id IS NULL;" >/dev/null || true
fi

BAD_LOGIN_IDENTIFIER="cube-smoke-bad-${RANDOM}-${RANDOM}"
BAD_LOGIN_IDENTIFIER="$BAD_LOGIN_IDENTIFIER" BAD_LOGIN_SECRET="wrong-${RANDOM}" node <<'NODE' >/tmp/cube-atom-login-bad-request.json
process.stdout.write(JSON.stringify({
  query: `mutation CubeLogin(\$input: LoginInput!) {
    login(input: \$input) {
      token
      entityId
      sessionId
      expiresAt
    }
  }`,
  variables: {
    input: {
      identifier: process.env.BAD_LOGIN_IDENTIFIER,
      secret: process.env.BAD_LOGIN_SECRET,
      kind: "password"
    }
  }
}));
NODE
curl -sS -o /tmp/cube-atom-login-bad.json \
  -H 'Content-Type: application/json' \
  -H "Origin: ${CUBE_UI_ORIGIN}" \
  --data-binary @/tmp/cube-atom-login-bad-request.json \
  "$ATOM_GRAPHQL_HTTP_URL"
if ! node -e 'const fs=require("fs"); const data=JSON.parse(fs.readFileSync("/tmp/cube-atom-login-bad.json","utf8")); process.exit(data.errors?.length && !data.data?.login?.token ? 0 : 1)'; then
  echo "expected bad GraphQL login to return errors and no token" >&2
  cat /tmp/cube-atom-login-bad.json >&2 || true
  exit 1
fi

echo "Logging in directly through ATOM GraphQL"
SMOKE_ATOM_IDENTIFIER="$SMOKE_ATOM_IDENTIFIER" SMOKE_ATOM_PASSWORD="$SMOKE_ATOM_PASSWORD" node <<'NODE' >/tmp/cube-atom-login-request.json
process.stdout.write(JSON.stringify({
  query: `mutation CubeLogin(\$input: LoginInput!) {
    login(input: \$input) {
      token
      entityId
      sessionId
      expiresAt
    }
  }`,
  variables: {
    input: {
      identifier: process.env.SMOKE_ATOM_IDENTIFIER,
      secret: process.env.SMOKE_ATOM_PASSWORD,
      kind: "password"
    }
  }
}));
NODE
curl -fsS \
  -H 'Content-Type: application/json' \
  -H "Origin: ${CUBE_UI_ORIGIN}" \
  --data-binary @/tmp/cube-atom-login-request.json \
  "$ATOM_GRAPHQL_HTTP_URL" >/tmp/cube-atom-login-response.json
node <<'NODE' >/tmp/cube-atom-login.json
const fs = require("fs");
const data = JSON.parse(fs.readFileSync("/tmp/cube-atom-login-response.json", "utf8"));
if (data.errors) throw new Error(data.errors[0].message);
const login = data.data?.login;
if (!login?.token || !login?.entityId || !login?.sessionId || !login?.expiresAt) {
  throw new Error("ATOM GraphQL login returned incomplete session data");
}
process.stdout.write(JSON.stringify(login));
NODE
ATOM_TOKEN="$(node -e 'const fs=require("fs"); process.stdout.write(JSON.parse(fs.readFileSync("/tmp/cube-atom-login.json","utf8")).token)')"
ENTITY_ID="$(node -e 'const fs=require("fs"); process.stdout.write(JSON.parse(fs.readFileSync("/tmp/cube-atom-login.json","utf8")).entityId)')"
SESSION_ID="$(node -e 'const fs=require("fs"); process.stdout.write(JSON.parse(fs.readFileSync("/tmp/cube-atom-login.json","utf8")).sessionId)')"
if [[ "$ATOM_TOKEN" != *.*.* ]]; then
  echo "ATOM GraphQL login did not return a JWT-shaped token" >&2
  exit 1
fi

curl -fsS \
  -H "Authorization: Bearer ${ATOM_TOKEN}" \
  -H 'Content-Type: application/json' \
  -H "Origin: ${CUBE_UI_ORIGIN}" \
  -d '{"query":"query CubeSession($id: ID!) { session(id: $id) { id entityId expiresAt revokedAt } }","variables":{"id":"'"${SESSION_ID}"'"}}' \
  "$ATOM_GRAPHQL_HTTP_URL" >/tmp/cube-atom-session.json
ENTITY_ID="$ENTITY_ID" node -e 'const fs=require("fs"); const data=JSON.parse(fs.readFileSync("/tmp/cube-atom-session.json","utf8")); if (data.errors) throw new Error(data.errors[0].message); if (data.data.session.entityId !== process.env.ENTITY_ID) throw new Error("session entity mismatch");'

SMOKE_NAME="cube-smoke-$(date +%s)"
echo "Creating, listing, and disabling smoke tenant $SMOKE_NAME"
curl -fsS \
  -H "Authorization: Bearer ${ATOM_TOKEN}" \
  -H 'Content-Type: application/json' \
  -H "Origin: ${CUBE_UI_ORIGIN}" \
  -d '{"query":"mutation CreateWorkspace($input: CreateTenantInput!) { createTenant(input: $input) { id name status } }","variables":{"input":{"name":"'"${SMOKE_NAME}"'","route":"'"${SMOKE_NAME}"'"}}}' \
  "$ATOM_GRAPHQL_HTTP_URL" >/tmp/cube-create-tenant.json

TENANT_ID="$(node -e 'const fs=require("fs"); const data=JSON.parse(fs.readFileSync("/tmp/cube-create-tenant.json","utf8")); if (data.errors) throw new Error(data.errors[0].message); process.stdout.write(data.data.createTenant.id)')"

curl -fsS \
  -H "Authorization: Bearer ${ATOM_TOKEN}" \
  -H 'Content-Type: application/json' \
  -H "Origin: ${CUBE_UI_ORIGIN}" \
  -d '{"query":"query { tenants(limit: 100, offset: 0) { items { id name status } } }"}' \
  "$ATOM_GRAPHQL_HTTP_URL" >/tmp/cube-list-tenants.json

curl -fsS \
  -H "Authorization: Bearer ${ATOM_TOKEN}" \
  -H 'Content-Type: application/json' \
  -H "Origin: ${CUBE_UI_ORIGIN}" \
  -d '{"query":"query CubeWorkspaceMembers($tenantId: ID!) { tenantMembers(tenantId: $tenantId, limit: 100, offset: 0) { items { id name kind status tenantId } total } }","variables":{"tenantId":"'"${TENANT_ID}"'"}}' \
  "$ATOM_GRAPHQL_HTTP_URL" >/tmp/cube-members.json

curl -fsS \
  -H "Authorization: Bearer ${ATOM_TOKEN}" \
  -H 'Content-Type: application/json' \
  -H "Origin: ${CUBE_UI_ORIGIN}" \
  -d '{"query":"query CubeIdentityAuditLogs($tenantId: ID) { auditLogs(tenantId: $tenantId, limit: 50, offset: 0) { items { id event outcome details createdAt } total } }","variables":{"tenantId":"'"${TENANT_ID}"'"}}' \
  "$ATOM_GRAPHQL_HTTP_URL" >/tmp/cube-identity-audit.json

curl -fsS \
  -H "Authorization: Bearer ${ATOM_TOKEN}" \
  -H 'Content-Type: application/json' \
  -H "Origin: ${CUBE_UI_ORIGIN}" \
  -d '{"query":"mutation DisableWorkspace($id: ID!) { disableTenant(id: $id) { id status } }","variables":{"id":"'"${TENANT_ID}"'"}}' \
  "$ATOM_GRAPHQL_HTTP_URL" >/tmp/cube-disable-tenant.json

routes_status="$(curl -sS -o /tmp/cube-proxy-routes.json -w '%{http_code}' \
  -H "Authorization: Bearer ${ATOM_TOKEN}" \
  "http://localhost:${UV_CUBE_PROXY_PORT:-8900}/api/routes")"
case "$routes_status" in
  200 | 403)
    ;;
  *)
    echo "expected authenticated cube-proxy route check to return 200 or 403, got $routes_status" >&2
    cat /tmp/cube-proxy-routes.json >&2 || true
    exit 1
    ;;
esac

echo "Issuing and validating a smoke certificate through ATOM"
openssl ecparam -name prime256v1 -genkey -noout -out /tmp/cube-smoke-cert.key
openssl req -new -key /tmp/cube-smoke-cert.key -subj "/CN=cube-smoke" -out /tmp/cube-smoke-cert.csr
node <<'NODE' >/tmp/cube-issue-cert-request.json
const fs = require("fs");
const login = JSON.parse(fs.readFileSync("/tmp/cube-atom-login.json", "utf8"));
const csrPem = fs.readFileSync("/tmp/cube-smoke-cert.csr", "utf8");
const entityId = login.entity_id ?? login.entityId;
if (!entityId) throw new Error("missing entity id from login session");
process.stdout.write(JSON.stringify({
  query: `mutation IssueCertificateFromCsr($input: IssueCertificateFromCsrInput!) {
    issueCertificateFromCsr(input: $input) {
      certificate {
        credentialId
        serialNumber
        certificatePem
        expiresAt
        fingerprintSha256
      }
    }
  }`,
  variables: {
    input: {
      entityId,
      csrPem,
      ttlSecs: 3600
    }
  }
}));
NODE

curl -fsS \
  -H "Authorization: Bearer ${ATOM_TOKEN}" \
  -H 'Content-Type: application/json' \
  -H "Origin: ${CUBE_UI_ORIGIN}" \
  --data-binary @/tmp/cube-issue-cert-request.json \
  "$ATOM_GRAPHQL_HTTP_URL" >/tmp/cube-issue-cert.json

node <<'NODE'
const fs = require("fs");
const data = JSON.parse(fs.readFileSync("/tmp/cube-issue-cert.json", "utf8"));
if (data.errors) throw new Error(data.errors[0].message);
const cert = data.data.issueCertificateFromCsr.certificate;
fs.writeFileSync("/tmp/cube-smoke-cert.crt", cert.certificatePem);
fs.writeFileSync("/tmp/cube-smoke-cert.json", JSON.stringify(cert));
NODE

openssl verify -CAfile /tmp/cube-atom-ca-chain.pem /tmp/cube-smoke-cert.crt >/dev/null

SERIAL="$(node -e 'const fs=require("fs"); process.stdout.write(JSON.parse(fs.readFileSync("/tmp/cube-smoke-cert.json","utf8")).serialNumber)')"
FINGERPRINT="$(node -e 'const fs=require("fs"); process.stdout.write(JSON.parse(fs.readFileSync("/tmp/cube-smoke-cert.json","utf8")).fingerprintSha256 ?? "")')"
ENTITY_ID="$(node -e 'const fs=require("fs"); const data=JSON.parse(fs.readFileSync("/tmp/cube-atom-login.json","utf8")); process.stdout.write(data.entity_id ?? data.entityId)')"
atom_grpcurl \
  -H "authorization: Bearer ${ATOM_TOKEN}" \
  -d "{\"serial_number\":\"${SERIAL}\",\"fingerprint_sha256\":\"${FINGERPRINT}\"}" \
  "localhost:${ATOM_GRPC_PORT}" atom.v1.CertificateService/ResolveCertificate >/tmp/cube-resolve-cert.json
atom_grpcurl \
  -H "authorization: Bearer ${ATOM_TOKEN}" \
  -d "{\"entity_id\":\"${ENTITY_ID}\",\"reason\":\"cube smoke\"}" \
  "localhost:${ATOM_GRPC_PORT}" atom.v1.CertificateService/RevokeEntityCertificates >/tmp/cube-revoke-cert.json
if atom_grpcurl \
  -H "authorization: Bearer ${ATOM_TOKEN}" \
  -d "{\"serial_number\":\"${SERIAL}\",\"fingerprint_sha256\":\"${FINGERPRINT}\"}" \
  "localhost:${ATOM_GRPC_PORT}" atom.v1.CertificateService/ResolveCertificate >/tmp/cube-resolve-revoked-cert.json 2>/tmp/cube-resolve-revoked-cert.err; then
  echo "expected revoked certificate resolve to fail" >&2
  exit 1
fi

echo "Logging out through ATOM GraphQL and verifying token revocation"
curl -fsS \
  -H "Authorization: Bearer ${ATOM_TOKEN}" \
  -H 'Content-Type: application/json' \
  -H "Origin: ${CUBE_UI_ORIGIN}" \
  -d '{"query":"mutation { logout }"}' \
  "$ATOM_GRAPHQL_HTTP_URL" >/tmp/cube-atom-logout.json
curl -sS -o /tmp/cube-atom-session-after-logout.json \
  -H "Authorization: Bearer ${ATOM_TOKEN}" \
  -H 'Content-Type: application/json' \
  -H "Origin: ${CUBE_UI_ORIGIN}" \
  -d '{"query":"query CubeSession($id: ID!) { session(id: $id) { id entityId } }","variables":{"id":"'"${SESSION_ID}"'"}}' \
  "$ATOM_GRAPHQL_HTTP_URL"
if ! node -e 'const fs=require("fs"); const data=JSON.parse(fs.readFileSync("/tmp/cube-atom-session-after-logout.json","utf8")); process.exit(data.errors?.length ? 0 : 1)'; then
  echo "expected logged-out token to fail GraphQL session lookup" >&2
  cat /tmp/cube-atom-session-after-logout.json >&2 || true
  exit 1
fi

if [ ! -x "$ROOT_DIR/ui/node_modules/.bin/playwright" ]; then
  echo "Installing Cube UI dependencies for Playwright smoke checks"
  npm --prefix ui ci
fi

echo "Running Cube UI Playwright smoke checks"
CUBE_UI_URL="http://localhost:${UI_PORT:-5173}" \
ATOM_API_URL="${ATOM_HTTP_URL}" \
ATOM_UI_URL="http://localhost:${ATOM_UI_HTTP_PORT:-3005}" \
SMOKE_ATOM_IDENTIFIER="$SMOKE_ATOM_IDENTIFIER" \
SMOKE_ATOM_PASSWORD="$SMOKE_ATOM_PASSWORD" \
npm --prefix ui run smoke:ui

echo "ATOM + Cube smoke checks passed."

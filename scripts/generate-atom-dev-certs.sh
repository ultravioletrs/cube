#!/usr/bin/env sh
set -eu

CERT_DIR="${1:-docker/certs}"
CERT_PARENT="$(dirname "$CERT_DIR")"
CERT_BASE="$(basename "$CERT_DIR")"

mkdir -p "$CERT_PARENT"
ABS_CERT_PARENT="$(cd "$CERT_PARENT" && pwd -P)"
ABS_CERT_DIR="$ABS_CERT_PARENT/$CERT_BASE"

if [ -d "$CERT_DIR" ] && [ ! -w "$CERT_DIR" ] && command -v docker >/dev/null 2>&1; then
  docker run --rm -v "$ABS_CERT_DIR:/target" docker:27.3.1 sh -c "chown $(id -u):$(id -g) /target" >/dev/null 2>&1 || true
fi

if [ -d "$CERT_DIR" ] && [ ! -w "$CERT_DIR" ]; then
  if [ -z "$(find "$CERT_DIR" -mindepth 1 -maxdepth 1 -print -quit 2>/dev/null)" ]; then
    rmdir "$CERT_DIR" 2>/dev/null || true
  fi
fi

mkdir -p "$CERT_DIR"

if [ ! -w "$CERT_DIR" ]; then
  echo "Cannot write ATOM dev CA files to $CERT_DIR" >&2
  echo "Remove or chown that directory, then run make up again." >&2
  exit 1
fi

ROOT_CERT="$CERT_DIR/root-ca.crt"
ROOT_KEY="$CERT_DIR/root-ca.key"
INTERMEDIATE_CERT="$CERT_DIR/intermediate-ca.crt"
INTERMEDIATE_KEY="$CERT_DIR/intermediate-ca.key"
INTERMEDIATE_CSR="$CERT_DIR/intermediate-ca.csr"
EXT_FILE="$CERT_DIR/intermediate-ca.ext"

if [ -f "$ROOT_CERT" ] && [ -f "$ROOT_KEY" ] && [ -f "$INTERMEDIATE_CERT" ] && [ -f "$INTERMEDIATE_KEY" ]; then
  echo "ATOM dev CA files already exist in $CERT_DIR"
  exit 0
fi

openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
  -keyout "$ROOT_KEY" \
  -out "$ROOT_CERT" \
  -subj "/CN=Cube ATOM Dev Root CA/O=Ultraviolet/C=US"

openssl req -newkey rsa:4096 -sha256 -nodes \
  -keyout "$INTERMEDIATE_KEY" \
  -out "$INTERMEDIATE_CSR" \
  -subj "/CN=Cube ATOM Dev Intermediate CA/O=Ultraviolet/C=US"

cat > "$EXT_FILE" <<'EOF'
basicConstraints=critical,CA:TRUE,pathlen:0
keyUsage=critical,keyCertSign,cRLSign
subjectKeyIdentifier=hash
authorityKeyIdentifier=keyid,issuer
EOF

openssl x509 -req -sha256 -days 1825 \
  -in "$INTERMEDIATE_CSR" \
  -CA "$ROOT_CERT" \
  -CAkey "$ROOT_KEY" \
  -CAcreateserial \
  -out "$INTERMEDIATE_CERT" \
  -extfile "$EXT_FILE"

rm -f "$INTERMEDIATE_CSR" "$EXT_FILE"
chmod 600 "$ROOT_KEY" "$INTERMEDIATE_KEY"

echo "Generated ATOM dev CA files in $CERT_DIR"

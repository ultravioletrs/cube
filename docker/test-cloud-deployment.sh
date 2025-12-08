#!/bin/bash
# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0
#
# Test script for local cloud deployment

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "==========================================="
echo "  Cube Cloud Deployment Test Script"
echo "==========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

function info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

function error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

function warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Step 1: Validate configuration
info "Step 1: Validating Docker Compose configuration..."
if docker compose -f cloud-compose.yaml --profile cloud config > /dev/null 2>&1; then
    info "✓ Configuration is valid"
else
    error "✗ Configuration validation failed"
    exit 1
fi
echo ""

# Step 2: Check for required files
info "Step 2: Checking required files..."
REQUIRED_FILES=(
    "cloud-compose.yaml"
    "config.json"
    "opensearch-index-template.json"
    "fluent-bit.conf"
    "parsers.conf"
    ".env"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        info "✓ Found $file"
    else
        error "✗ Missing $file"
        exit 1
    fi
done
echo ""

# Step 3: Pull images
info "Step 3: Pulling latest Docker images..."
docker compose -f cloud-compose.yaml --profile cloud pull
echo ""

# Step 4: Stop any existing services
info "Step 4: Stopping any existing services..."
docker compose -f cloud-compose.yaml --profile cloud down
echo ""

# Step 5: Start cloud services
info "Step 5: Starting cloud services..."
docker compose -f cloud-compose.yaml --profile cloud up -d
echo ""

# Step 6: Wait for services to be healthy
info "Step 6: Waiting for services to start (30 seconds)..."
sleep 30
echo ""

# Step 7: Check service status
info "Step 7: Checking service status..."
docker compose -f cloud-compose.yaml --profile cloud ps
echo ""

# Step 8: Verify critical services
info "Step 8: Verifying critical services are running..."
SERVICES="traefik opensearch nats spicedb auth users cube-proxy fluent-bit"
ALL_RUNNING=true

for SERVICE in $SERVICES; do
    if docker ps --format '{{.Names}}' | grep -q "cube-cloud-$SERVICE"; then
        info "✓ Service $SERVICE is running"
    else
        error "✗ Service $SERVICE is not running"
        ALL_RUNNING=false
    fi
done
echo ""

if [ "$ALL_RUNNING" = false ]; then
    error "Some services failed to start!"
    warn "Showing logs for failed services:"
    docker compose -f cloud-compose.yaml --profile cloud logs --tail=50
    exit 1
fi

# Step 9: Run smoke tests
info "Step 9: Running smoke tests..."

# Test OpenSearch
info "Testing OpenSearch..."
if curl -s http://localhost:9200/_cluster/health > /dev/null 2>&1; then
    info "✓ OpenSearch is responding"
else
    warn "✗ OpenSearch is not responding"
fi

# Test Jaeger
info "Testing Jaeger..."
if curl -s http://localhost:16686/api/services > /dev/null 2>&1; then
    info "✓ Jaeger is responding"
else
    warn "✗ Jaeger is not responding"
fi

# Test Traefik
info "Testing Traefik..."
if curl -s http://localhost:8080/api/http/routers > /dev/null 2>&1; then
    info "✓ Traefik dashboard is responding"
else
    warn "✗ Traefik dashboard is not responding"
fi

# Test Cube Proxy
info "Testing Cube Proxy..."
if curl -s http://localhost:8900/health > /dev/null 2>&1; then
    info "✓ Cube Proxy is responding"
else
    warn "✗ Cube Proxy health check failed (expected if agent is not running)"
fi

echo ""
info "==========================================="
info "  Cloud Deployment Test Complete!"
info "==========================================="
echo ""
info "Service URLs:"
info "  - Traefik Dashboard: http://localhost:8080"
info "  - Cube Proxy: http://localhost:8900"
info "  - Jaeger UI: http://localhost:16686"
info "  - OpenSearch: http://localhost:9200"
echo ""
info "To view logs:"
info "  docker compose -f cloud-compose.yaml --profile cloud logs -f"
echo ""
info "To stop services:"
info "  docker compose -f cloud-compose.yaml --profile cloud down"
echo ""

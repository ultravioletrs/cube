#!/bin/bash
# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

# Mount host certificates via 9p and configure mTLS
set -e

# Create mount point
mkdir -p /etc/cube/certs

# Mount the 9p shared folder
if ! mountpoint -q /etc/cube/certs; then
    mount -t 9p -o trans=virtio,version=9p2000.L certs_share /etc/cube/certs || {
        echo "Failed to mount host certificates"
        exit 1
    }
fi

# Update agent environment if certs are available
if [ -f /etc/cube/certs/ca.pem ]; then
    # Remove any existing mTLS config lines
    sed -i '/UV_CUBE_AGENT_SERVER_CA_CERTS=/d' /etc/cube/agent.env 2>/dev/null || true
    sed -i '/UV_CUBE_AGENT_SERVER_CERT=/d' /etc/cube/agent.env 2>/dev/null || true
    sed -i '/UV_CUBE_AGENT_SERVER_KEY=/d' /etc/cube/agent.env 2>/dev/null || true
    sed -i '/UV_CUBE_AGENT_CLIENT_CA_CERTS=/d' /etc/cube/agent.env 2>/dev/null || true
    sed -i '/UV_CUBE_AGENT_ATTESTED_TLS=/d' /etc/cube/agent.env 2>/dev/null || true
    
    # Append mTLS configuration
    echo "UV_CUBE_AGENT_SERVER_CA_CERTS=/etc/cube/certs/ca.pem" >> /etc/cube/agent.env
    echo "UV_CUBE_AGENT_SERVER_CERT=/etc/cube/certs/server.crt" >> /etc/cube/agent.env
    echo "UV_CUBE_AGENT_SERVER_KEY=/etc/cube/certs/server.key" >> /etc/cube/agent.env
    echo "UV_CUBE_AGENT_CLIENT_CA_CERTS=/etc/cube/certs/client_ca.pem" >> /etc/cube/agent.env
    echo "UV_CUBE_AGENT_ATTESTED_TLS=true" >> /etc/cube/agent.env
    
    echo "mTLS certificates mounted and configured successfully"
else
    echo "No certificates found in host share"
fi

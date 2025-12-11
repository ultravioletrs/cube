#!/bin/bash
# Configure mTLS environment variables for cube-agent
if [ -f /etc/cube/certs/ca.pem ]; then
    sed -i '/UV_CUBE_AGENT_SERVER_CA_CERTS=/d' /etc/cube/agent.env 2>/dev/null || true
    sed -i '/UV_CUBE_AGENT_SERVER_CERT=/d' /etc/cube/agent.env 2>/dev/null || true
    sed -i '/UV_CUBE_AGENT_SERVER_KEY=/d' /etc/cube/agent.env 2>/dev/null || true
    sed -i '/UV_CUBE_AGENT_CLIENT_CA_CERTS=/d' /etc/cube/agent.env 2>/dev/null || true
    sed -i '/UV_CUBE_AGENT_ATTESTED_TLS=/d' /etc/cube/agent.env 2>/dev/null || true
    
    echo 'UV_CUBE_AGENT_SERVER_CA_CERTS=/etc/cube/certs/ca.pem' >> /etc/cube/agent.env
    echo 'UV_CUBE_AGENT_SERVER_CERT=/etc/cube/certs/server.crt' >> /etc/cube/agent.env
    echo 'UV_CUBE_AGENT_SERVER_KEY=/etc/cube/certs/server.key' >> /etc/cube/agent.env
    echo 'UV_CUBE_AGENT_CLIENT_CA_CERTS=/etc/cube/certs/client_ca.pem' >> /etc/cube/agent.env
    echo 'UV_CUBE_AGENT_ATTESTED_TLS=true' >> /etc/cube/agent.env
    echo 'mTLS configured successfully'
fi

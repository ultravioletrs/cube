# Enhanced NeMo Guardrails Dockerfile with Dynamic Configuration Support
FROM nvidia/cuda:11.8.0-runtime-ubuntu22.04

WORKDIR /app

# Install Python and system dependencies
RUN apt-get update && apt-get install -y \
    python3.10 \
    python3-pip \
    curl \
    git \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Set Python 3.10 as default
RUN update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.10 1
RUN update-alternatives --install /usr/bin/python python /usr/bin/python3.10 1

# Copy requirements file
COPY docker/guardrails/requirements.txt /app/requirements.txt

# Upgrade pip and install dependencies
RUN pip3 install --upgrade pip setuptools wheel && \
    pip3 install --no-cache-dir -r requirements.txt

# Install additional dependencies for webhook system
RUN pip3 install --no-cache-dir \
    pyyaml \
    aiohttp

RUN python -m spacy download en_core_web_lg

# Copy configuration management scripts
COPY docker/guardrails/webhook_receiver.py /app/webhook_receiver.py
COPY docker/guardrails/webhook_startup.py /app/webhook_startup.py
RUN chmod +x /app/webhook_receiver.py /app/webhook_startup.py

# Copy configuration files
COPY docker/guardrails/config/ /config/

# Create necessary directories and set proper permissions
RUN mkdir -p /config /logs /app && \
    chmod 755 /config /logs /app

# Set environment variables
ENV GUARDRAILS_CONFIG_PATH=/config
ENV PYTHONPATH=/config
ENV OLLAMA_BASE_URL=http://109.92.195.153:7107

# Webhook configuration environment variables
ENV ENABLE_WEBHOOK=true
ENV WEBHOOK_HOST=0.0.0.0
ENV WEBHOOK_PORT=8080

# Enhanced logging configuration
ENV NEMO_LOG_LLM_CALLS=true
ENV NEMO_LOG_USER_MESSAGES=true
ENV NEMO_LOG_BOT_MESSAGES=true
ENV NEMO_LOG_SAFETY_VIOLATIONS=true
ENV NEMO_LOG_REDACTION_ACTIONS=true
ENV NEMO_LOG_FORMAT=json
ENV NEMO_LOG_FILE=/logs/nemo-guardrails.log

# Expose ports
EXPOSE 8001 8080

# Health check - checks NeMo Guardrails main port
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
    CMD curl -f http://localhost:8001/health || exit 1

# Create startup script that handles webhook mode
RUN cat > /app/startup.sh << 'EOF'
#!/bin/bash
set -e

echo "Starting NeMo Guardrails with webhook support..."
echo "Webhook enabled: ${ENABLE_WEBHOOK}"
echo "Webhook port: ${WEBHOOK_PORT}"

if [ "${ENABLE_WEBHOOK}" = "true" ]; then
    echo "Using webhook-based configuration mode"
    exec python3 /app/webhook_startup.py
else
    echo "Using static configuration mode"
    exec nemoguardrails server --config /config --port 8001
fi
EOF

RUN chmod +x /app/startup.sh

# Use startup script as entrypoint
ENTRYPOINT ["/app/startup.sh"]
FROM nvidia/cuda:11.8.0-runtime-ubuntu22.04

WORKDIR /app

RUN apt-get update && apt-get install -y \
    python3.10 \
    python3-pip \
    curl \
    git \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

RUN update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.10 1
RUN update-alternatives --install /usr/bin/python python /usr/bin/python3.10 1

COPY docker/guardrails/requirements.txt /app/requirements.txt

RUN pip3 install --upgrade pip setuptools wheel && \
    pip3 install --no-cache-dir -r requirements.txt

RUN pip3 install --no-cache-dir \
    pyyaml \
    aiohttp

#Ensure that the en_core_web_lg model is available for spaCy, which Presidio utilizes for its sensitive data detection capabilities.
RUN python -m spacy download en_core_web_lg

RUN mkdir -p /config /logs /app && \
    chmod 755 /config /logs /app

COPY docker/guardrails/webhook_receiver.py /app/webhook_receiver.py
COPY docker/guardrails/webhook_startup.py /app/webhook_startup.py
RUN chmod +x /app/webhook_receiver.py /app/webhook_startup.py

COPY docker/guardrails/config/ /config/

# Set environment variables
ENV GUARDRAILS_CONFIG_PATH=/config
ENV PYTHONPATH=/config
ENV OLLAMA_BASE_URL=http://109.92.195.153:6107

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
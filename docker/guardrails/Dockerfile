FROM python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Install NeMo Guardrails
RUN pip install --no-cache-dir nemoguardrails==0.8.0 \
    fastapi==0.109.0 \
    uvicorn==0.27.0 \
    pydantic==2.5.3

# Copy configuration files
COPY ./docker/guardrails/guardrails-config /config

# Set environment variables
ENV GUARDRAILS_CONFIG_PATH=/config
ENV PYTHONPATH=/config:${PYTHONPATH:-}

# Expose port
EXPOSE 8001

# Run the guardrails server
CMD ["nemoguardrails", "server", "--config", "/config", "--port", "8001"]
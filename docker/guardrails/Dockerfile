FROM python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements file
COPY ./docker/guardrails/requirements.txt /app/requirements.txt

# Upgrade pip and install dependencies
RUN pip install --upgrade pip setuptools wheel && \
    pip install --no-cache-dir -r requirements.txt

# Copy configuration files
COPY ./docker/guardrails/config /config

# Set environment variables
ENV GUARDRAILS_CONFIG_PATH=/config
ENV PYTHONPATH=/config
ENV OLLAMA_BASE_URL=http://ollama:11434

# Create necessary directories and set proper permissions
RUN mkdir -p /config && chmod 755 /config

# Expose port
EXPOSE 8001

# Run the guardrails server
CMD ["nemoguardrails", "server", "--config", "/config", "--port", "8001"]
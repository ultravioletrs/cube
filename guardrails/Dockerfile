# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

# Multi-stage build for smaller image size
FROM python:3.11-slim as builder

# Install build dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    build-essential \
    gcc \
    g++ \
    git \
    curl && \
    rm -rf /var/lib/apt/lists/*

# Create virtual environment
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Copy and install requirements
COPY requirements.txt .
RUN pip install --upgrade pip && \
    pip install --no-cache-dir -r requirements.txt && \
    python -m spacy download en_core_web_lg

# Production stage
FROM python:3.11-slim

# Create non-root user
RUN groupadd -r guardrails && useradd -r -g guardrails guardrails

# Copy virtual environment from builder
COPY --from=builder /opt/venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Set working directory
WORKDIR /app

# Copy application files
COPY --chown=guardrails:guardrails main.py .
COPY --chown=guardrails:guardrails src ./src

# Copy rails configuration (for legacy mode)
COPY --chown=guardrails:guardrails rails /app/rails

# Create logs directory for tracing
RUN mkdir -p /app/logs && chown -R guardrails:guardrails /app/logs

# Set permissions
RUN chown -R guardrails:guardrails /app

# Install runtime dependencies only
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    curl && \
    rm -rf /var/lib/apt/lists/*

# Switch to non-root user
USER guardrails

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8001/health || exit 1

# Expose port
EXPOSE 8001

# Start server
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8001", "--workers", "1"]

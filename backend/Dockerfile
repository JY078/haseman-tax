# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server main.go

# Runtime stage
FROM python:3.12-slim-bookworm

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata wget tesseract-ocr tesseract-ocr-chi-sim tesseract-ocr-eng && rm -rf /var/lib/apt/lists/*
COPY internal/customs/requirements.txt /tmp/customs-requirements.txt
RUN python3 -m venv /opt/customs-venv && /opt/customs-venv/bin/pip install --no-cache-dir -r /tmp/customs-requirements.txt
ENV CUSTOMS_PYTHON=/opt/customs-venv/bin/python

# Copy binary from builder
COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/internal ./internal
COPY --from=builder /app/fonts ./fonts

# Copy entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Create directories for logs and uploads
RUN mkdir -p /app/logs /app/uploads /app/data /app/config

# Expose port
EXPOSE 8080

# Run as non-root user
RUN useradd --create-home appuser
USER appuser

ENTRYPOINT ["/entrypoint.sh"]

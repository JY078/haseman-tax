#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
# Standalone Go/PM2 deployments use the loopback published port; Compose backend
# containers use REMBG_SERVICE_URL=http://rembg-service:8001 instead.
docker compose up -d --build --wait --wait-timeout 180 rembg-service
curl --fail --silent --show-error http://127.0.0.1:8011/health

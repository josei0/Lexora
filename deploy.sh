#!/usr/bin/env bash
set -euo pipefail

# ponytail: urutan deploy — migrate dulu, baru start app
cd "$(dirname "$0")"

echo "==> build & start"
docker compose -f docker-compose.prod.yml up -d --build postgres qdrant

echo "==> tunggu postgres sehat..."
docker compose -f docker-compose.prod.yml exec postgres \
  sh -c 'until pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"; do sleep 1; done'

echo "==> migrate"
docker compose -f docker-compose.prod.yml run --rm backend \
  sh -c './migrate -path ./migrations -database "$DATABASE_URL" up'

echo "==> seed (idempotent)"
docker compose -f docker-compose.prod.yml run --rm backend ./seed

echo "==> start semua service"
docker compose -f docker-compose.prod.yml up -d

echo "==> done. cek: docker compose -f docker-compose.prod.yml ps"

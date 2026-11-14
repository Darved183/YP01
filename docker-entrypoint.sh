#!/bin/sh
set -e

mkdir -p /data/uploads

# база и загруженные фото лежат в /data и не пропадают при пересборке образа
ln -sfn /data/db.db /app/db.db
ln -sfn /data/uploads /app/uploads

# программа читает .env при старте. переменные можно передать контейнеру
printf 'HOST=%s\nPORT=%s\n' "${HOST:-0.0.0.0}" "${PORT:-8181}" > /app/.env

exec /app/app

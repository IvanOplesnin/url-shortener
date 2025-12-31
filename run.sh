#!/usr/bin/env bash
set -euo pipefail

set -a
. "$ENV_FILE"
set +a

echo "----------Параметры БД-------------"
echo "owner: ${DB_OWNER_USER-}"
echo "app_user: ${APP_USER-}"
echo "DB: ${POSTGRES_DB-}"

echo "--------Параметры приложения---------"
echo "address_server: ${SERVER_ADDRESS-}"
echo "base_url: ${BASE_URL-}"
echo "file_storage_path: ${FILE_STORAGE_PATH-}"
echo "dsn: ${DATABASE_DSN-}"
echo

go run ./cmd/shortener/main.go

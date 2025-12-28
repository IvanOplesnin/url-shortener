#!/usr/bin/env bash
set -euo pipefail

: "${POSTGRES_USER:?POSTGRES_USER is required}"
: "${POSTGRES_DB:?POSTGRES_DB is required}"

: "${DB_OWNER_USER:?DB_OWNER_USER is required}"
: "${DB_OWNER_PASSWORD:?DB_OWNER_PASSWORD is required}"

: "${APP_USER:?APP_USER is required}"
: "${APP_PASSWORD:?APP_PASSWORD is required}"

DB_NAME="${DB_NAME:-$POSTGRES_DB}"

psql_base=(psql -X -v ON_ERROR_STOP=1 --username "$POSTGRES_USER")

echo "DB: ${DB_NAME}"
echo "Owner: ${DB_OWNER_USER}"
echo "App: ${APP_USER}"

# 1) Создать owner, если нет
"${psql_base[@]}" --dbname postgres \
  --set=owner="$DB_OWNER_USER" \
  --set=owner_pass="$DB_OWNER_PASSWORD" <<'SQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'owner', :'owner_pass')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'owner');
\gexec
SQL

# 2) Создать app, если нет
"${psql_base[@]}" --dbname postgres \
  --set=app="$APP_USER" \
  --set=app_pass="$APP_PASSWORD" <<'SQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'app', :'app_pass')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'app');
\gexec
SQL

# 3) Назначить владельца БД + почистить PUBLIC + CONNECT
"${psql_base[@]}" --dbname postgres \
  --set=db="$DB_NAME" \
  --set=owner="$DB_OWNER_USER" \
  --set=app="$APP_USER" <<'SQL'
ALTER DATABASE :"db" OWNER TO :"owner";
REVOKE ALL  ON DATABASE :"db" FROM PUBLIC;
REVOKE TEMP ON DATABASE :"db" FROM PUBLIC;

GRANT CONNECT ON DATABASE :"db" TO :"owner";
GRANT CONNECT ON DATABASE :"db" TO :"app";
SQL

# 4) Настройка схемы и прав внутри целевой БД
"${psql_base[@]}" --dbname "$DB_NAME" \
  --set=owner="$DB_OWNER_USER" \
  --set=app="$APP_USER" <<'SQL'
ALTER SCHEMA public OWNER TO :"owner";
REVOKE CREATE ON SCHEMA public FROM PUBLIC;

GRANT USAGE ON SCHEMA public TO :"app";
SQL

# 5) CRUD на существующие таблицы/sequence (если пока нет — просто 0 объектов)
"${psql_base[@]}" --dbname "$DB_NAME" \
  --set=app="$APP_USER" <<'SQL'
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO :"app";
GRANT USAGE, SELECT, UPDATE          ON ALL SEQUENCES IN SCHEMA public TO :"app";
SQL

# 6) Автоправа на будущие объекты, которые создаст владелец (миграции идут от owner)
"${psql_base[@]}" --dbname "$DB_NAME" \
  --set=owner="$DB_OWNER_USER" \
  --set=app="$APP_USER" <<'SQL'
ALTER DEFAULT PRIVILEGES FOR ROLE :"owner" IN SCHEMA public
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO :"app";

ALTER DEFAULT PRIVILEGES FOR ROLE :"owner" IN SCHEMA public
GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO :"app";
SQL

echo "Done."

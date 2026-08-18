#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
env_file="$project_root/.env"

if [ ! -f "$env_file" ]; then
    printf '%s\n' "Missing $env_file; run ./scripts/init-local-env.sh first" >&2
    exit 1
fi

set -a
# shellcheck disable=SC1090
. "$env_file"
set +a

server_url="http://127.0.0.1:${SERVER_HOST_PORT:-8888}"
check_response=$(curl --fail --silent --show-error \
    --request POST \
    "$server_url/init/checkdb")

if ! printf '%s' "$check_response" | grep -Eq '"needInit"[[:space:]]*:[[:space:]]*true'; then
    printf '%s\n' 'Database is already initialized.'
    exit 0
fi

payload=$(printf '{"dbType":"mysql","host":"mysql","port":"3306","userName":"%s","password":"%s","dbName":"%s","adminPassword":"%s"}' \
    "$MYSQL_USER" "$MYSQL_PASSWORD" "$MYSQL_DATABASE" "$GVA_ADMIN_PASSWORD")

init_response=$(curl --fail --silent --show-error \
    --header 'Content-Type: application/json' \
    --request POST \
    --data "$payload" \
    "$server_url/init/initdb")

if ! printf '%s' "$init_response" | grep -Eq '"code"[[:space:]]*:[[:space:]]*0'; then
    printf '%s\n' 'Database initialization failed:' >&2
    printf '%s\n' "$init_response" >&2
    exit 1
fi

printf '%s\n' 'Database schema, initial roles, menus, and admin account are ready.'

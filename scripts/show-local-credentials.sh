#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
env_file="$project_root/.env"

if [ ! -f "$env_file" ]; then
    printf '%s\n' "Missing $env_file; run make init first" >&2
    exit 1
fi

set -a
# shellcheck disable=SC1090
. "$env_file"
set +a

printf 'Web:             http://127.0.0.1:%s\n' "${WEB_HOST_PORT:-8080}"
printf 'Swagger:         http://127.0.0.1:%s/api/swagger/index.html\n' "${WEB_HOST_PORT:-8080}"
printf 'Admin username:  admin\n'
printf 'Admin password:  %s\n' "$GVA_ADMIN_PASSWORD"
printf 'Synthetic supervisor: syn_supervisor_a\n'
printf 'Synthetic steward A: syn_steward_a\n'
printf 'Synthetic steward B: syn_steward_b\n'
printf 'Synthetic clinician: syn_clinician_a\n'
printf 'Synthetic password: %s\n' "$GVA_CARE_FIXTURE_PASSWORD"
printf 'Care client login: http://127.0.0.1:%s/#/client/login\n' "${WEB_HOST_PORT:-8080}"
printf 'Care client username: linanran\n'
printf 'Care client password: %s\n' "$GVA_CARE_FIXTURE_PASSWORD"
printf 'MySQL endpoint:  127.0.0.1:%s\n' "${MYSQL_HOST_PORT:-13306}"
printf 'MySQL database:  %s\n' "$MYSQL_DATABASE"
printf 'MySQL username:  %s\n' "$MYSQL_USER"
printf 'MySQL password:  %s\n' "$MYSQL_PASSWORD"
printf 'Redis endpoint:  127.0.0.1:%s\n' "${REDIS_HOST_PORT:-16379}"
printf 'Redis password:  %s\n' "$REDIS_PASSWORD"

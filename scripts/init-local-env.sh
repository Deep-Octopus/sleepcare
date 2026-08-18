#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
env_file="$project_root/.env"

if [ -f "$env_file" ]; then
    if grep -q 'replace_with_generated_value' "$env_file"; then
        printf '%s\n' "$env_file still contains example placeholders; remove it and run make init" >&2
        exit 1
    fi
    if ! grep -Eq '^GVA_CARE_FIXTURE_PASSWORD=.+$' "$env_file"; then
        if ! command -v openssl >/dev/null 2>&1; then
            printf '%s\n' "openssl is required to generate local credentials" >&2
            exit 1
        fi
        umask 077
        fixture_password=$(openssl rand -base64 24 | tr -d '=+/\n' | cut -c1-24)
        printf 'GVA_CARE_FIXTURE_PASSWORD=%s\n' "$fixture_password" >> "$env_file"
    fi
    for required_key in MYSQL_PASSWORD MYSQL_ROOT_PASSWORD REDIS_PASSWORD GVA_ADMIN_PASSWORD GVA_CARE_FIXTURE_PASSWORD; do
        if ! grep -Eq "^${required_key}=.+$" "$env_file"; then
            printf '%s\n' "Missing ${required_key} in $env_file" >&2
            exit 1
        fi
    done
    chmod 0600 "$env_file"
    printf '%s\n' "Local environment already exists: $env_file"
    exit 0
fi

if ! command -v openssl >/dev/null 2>&1; then
    printf '%s\n' "openssl is required to generate local credentials" >&2
    exit 1
fi

umask 077
tmp_file="$env_file.tmp.$$"
trap 'rm -f "$tmp_file"' EXIT HUP INT TERM

mysql_password=$(openssl rand -hex 24)
mysql_root_password=$(openssl rand -hex 24)
redis_password=$(openssl rand -hex 24)
admin_password=$(openssl rand -base64 24 | tr -d '=+/\n' | cut -c1-24)
fixture_password=$(openssl rand -base64 24 | tr -d '=+/\n' | cut -c1-24)

{
    printf '%s\n' 'COMPOSE_PROJECT_NAME=sleep-care'
    printf '%s\n' 'MYSQL_IMAGE=m.daocloud.io/docker.io/library/mysql:8.4.11'
    printf '%s\n' 'MYSQL_DATABASE=sleep_care'
    printf '%s\n' 'MYSQL_USER=sleepcare'
    printf 'MYSQL_PASSWORD=%s\n' "$mysql_password"
    printf 'MYSQL_ROOT_PASSWORD=%s\n' "$mysql_root_password"
    printf '%s\n' 'REDIS_IMAGE=m.daocloud.io/docker.io/library/redis:7.4.10-alpine3.21'
    printf 'REDIS_PASSWORD=%s\n' "$redis_password"
    printf '%s\n' 'GO_BUILD_IMAGE=m.daocloud.io/docker.io/library/golang:1.24.13-alpine3.23'
    printf '%s\n' 'SERVER_RUNTIME_IMAGE=m.daocloud.io/docker.io/library/alpine:3.23'
    printf '%s\n' 'NODE_BUILD_IMAGE=m.daocloud.io/docker.io/library/node:22.23.2-alpine3.24'
    printf '%s\n' 'NGINX_IMAGE=m.daocloud.io/docker.io/library/nginx:1.29.8-alpine3.23'
    printf 'GVA_ADMIN_PASSWORD=%s\n' "$admin_password"
    printf 'GVA_CARE_FIXTURE_PASSWORD=%s\n' "$fixture_password"
    printf '%s\n' 'WEB_HOST_PORT=8080'
    printf '%s\n' 'SERVER_HOST_PORT=8888'
    printf '%s\n' 'MYSQL_HOST_PORT=13306'
    printf '%s\n' 'REDIS_HOST_PORT=16379'
} > "$tmp_file"

mv "$tmp_file" "$env_file"
trap - EXIT HUP INT TERM
printf '%s\n' "Generated local credentials in $env_file"

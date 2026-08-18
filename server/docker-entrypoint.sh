#!/bin/sh
set -eu

config_path="${GVA_CONFIG:-/app/config/config.yaml}"
config_dir=$(dirname "$config_path")

mkdir -p "$config_dir"
if [ ! -f "$config_path" ]; then
    cp /app/config.default.yaml "$config_path"
fi

exec /app/server -c "$config_path" "$@"

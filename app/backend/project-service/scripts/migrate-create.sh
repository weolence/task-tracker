#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
MIGRATIONS_DIR="${MIGRATIONS_DIR:-$SCRIPT_DIR/../migrations}"
RAW_NAME="${1:-}"

usage() {
    printf 'Usage: %s migration_name\n' "$(basename "$0")" >&2
    exit 1
}

slugify() {
    printf '%s' "$1" \
        | tr '[:upper:]' '[:lower:]' \
        | sed -E 's/[^a-z0-9]+/_/g; s/^_+//; s/_+$//; s/_+/_/g'
}

main() {
    if [[ -z "$RAW_NAME" ]]; then
        usage
    fi

    if [[ ! -d "$MIGRATIONS_DIR" ]]; then
        printf 'migrations directory not found: %s\n' "$MIGRATIONS_DIR" >&2
        exit 1
    fi

    slug=$(slugify "$RAW_NAME")
    if [[ -z "$slug" ]]; then
        printf 'migration name must contain letters or digits\n' >&2
        exit 1
    fi

    version=$(date -u +%Y%m%d%H%M%S)
    up_file="$MIGRATIONS_DIR/${version}_${slug}.up.sql"
    down_file="$MIGRATIONS_DIR/${version}_${slug}.down.sql"

    if [[ -e "$up_file" || -e "$down_file" ]]; then
        printf 'migration files already exist for version %s\n' "$version" >&2
        exit 1
    fi

    printf '%s\n' "-- Migration: ${slug}" >"$up_file"
    printf '%s\n' "-- Write forward migration here." >>"$up_file"

    printf '%s\n' "-- Migration: ${slug}" >"$down_file"
    printf '%s\n' "-- Write rollback migration here." >>"$down_file"

    printf 'Created migration files:\n%s\n%s\n' "$up_file" "$down_file"
}

main "$@"

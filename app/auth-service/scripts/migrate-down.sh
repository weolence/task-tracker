#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
MIGRATIONS_DIR="${MIGRATIONS_DIR:-$SCRIPT_DIR/../migrations}"
PSQL_BIN="${PSQL_BIN:-psql}"
TARGET_VERSION="${1:-}"

psql_exec() {
    if [[ -n "${DATABASE_URL:-}" ]]; then
        "$PSQL_BIN" "$DATABASE_URL" "$@"
        return
    fi

    "$PSQL_BIN" "$@"
}

ensure_requirements() {
    if ! command -v "$PSQL_BIN" >/dev/null 2>&1; then
        printf 'psql binary not found: %s\n' "$PSQL_BIN" >&2
        exit 1
    fi

    if [[ ! -d "$MIGRATIONS_DIR" ]]; then
        printf 'migrations directory not found: %s\n' "$MIGRATIONS_DIR" >&2
        exit 1
    fi
}

ensure_migrations_table() {
    psql_exec -v ON_ERROR_STOP=1 -c "
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version    text        PRIMARY KEY,
            applied_at timestamptz NOT NULL DEFAULT now()
        );
    "
}

resolve_target_version() {
    if [[ -n "$TARGET_VERSION" ]]; then
        printf '%s' "$TARGET_VERSION"
        return
    fi

    psql_exec -At -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;"
}

main() {
    ensure_requirements
    ensure_migrations_table

    target_version=$(resolve_target_version)

    if [[ -z "$target_version" ]]; then
        printf 'No applied migrations to roll back.\n'
        exit 0
    fi

    applied=$(psql_exec -At -c "SELECT 1 FROM schema_migrations WHERE version = '$target_version' LIMIT 1;")
    if [[ "$applied" != "1" ]]; then
        printf 'Migration %s is not marked as applied.\n' "$target_version" >&2
        exit 1
    fi

    down_file="$MIGRATIONS_DIR/$target_version.down.sql"
    if [[ ! -f "$down_file" ]]; then
        printf 'Down migration file not found: %s\n' "$down_file" >&2
        exit 1
    fi

    printf 'Reverting %s\n' "$target_version"
    psql_exec -v ON_ERROR_STOP=1 \
        -c "BEGIN" \
        -f "$down_file" \
        -c "DELETE FROM schema_migrations WHERE version = '$target_version'" \
        -c "COMMIT"
}

main "$@"

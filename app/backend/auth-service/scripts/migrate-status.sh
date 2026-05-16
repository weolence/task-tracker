#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
MIGRATIONS_DIR="${MIGRATIONS_DIR:-$SCRIPT_DIR/../migrations}"
PSQL_BIN="${PSQL_BIN:-psql}"

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

main() {
    ensure_requirements
    ensure_migrations_table

    shopt -s nullglob
    mapfile -t up_files < <(find "$MIGRATIONS_DIR" -maxdepth 1 -type f -name '*.up.sql' | LC_ALL=C sort)

    if [[ ${#up_files[@]} -eq 0 ]]; then
        printf 'No migrations found in %s\n' "$MIGRATIONS_DIR"
        exit 0
    fi

    printf '%-12s %s\n' 'STATUS' 'VERSION'
    for file_path in "${up_files[@]}"; do
        file_name=$(basename "$file_path")
        version="${file_name%.up.sql}"
        applied=$(psql_exec -At -c "SELECT 1 FROM schema_migrations WHERE version = '$version' LIMIT 1;")

        if [[ "$applied" == "1" ]]; then
            printf '%-12s %s\n' 'applied' "$version"
        else
            printf '%-12s %s\n' 'pending' "$version"
        fi
    done
}

main "$@"

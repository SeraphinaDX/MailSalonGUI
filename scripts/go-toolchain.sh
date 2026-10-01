#!/bin/sh
# Fyne's desktop driver uses cgo. Prefer zgo; honor explicit GO/ZGO overrides.
set -eu
if [ -n "${ZGO:-}" ]; then
    if ! command -v "$ZGO" >/dev/null 2>&1; then
        echo "Requested ZGO executable not found: $ZGO" >&2
        exit 1
    fi
    exec "$ZGO" "$@"
fi
if command -v zgo >/dev/null 2>&1; then
    exec zgo "$@"
fi
exec "${GO:-go}" "$@"

#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLUGIN_PATH="${1:-"$ROOT_DIR/plugins/baseline"}"
VALIDATOR="${PLUGIN_VALIDATOR:-"${CODEX_HOME:-"$HOME/.codex"}/skills/.system/plugin-creator/scripts/validate_plugin.py"}"

if [[ ! -f "$VALIDATOR" ]]; then
  echo "Codex plugin validator not found: $VALIDATOR" >&2
  echo "Set PLUGIN_VALIDATOR to the plugin-creator validate_plugin.py path." >&2
  exit 2
fi

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

if ! python3 -c 'import yaml' >/dev/null 2>&1; then
  python3 -m pip install --quiet --target "$TMP_DIR" 'PyYAML>=6,<7'
  export PYTHONPATH="$TMP_DIR${PYTHONPATH:+:$PYTHONPATH}"
fi

python3 "$VALIDATOR" "$PLUGIN_PATH"

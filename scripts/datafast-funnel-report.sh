#!/usr/bin/env bash
set -euo pipefail

: "${DATAFAST_TOKEN:?Set DATAFAST_TOKEN to a DataFast dft_ account token or df_ website key.}"

WEBSITE_ID="${DATAFAST_WEBSITE_ID:-6a0c48aa9a21aee7bf04cf6e}"
PERIOD="${DATAFAST_PERIOD:-last7d}"
DATAFAST_BIN="${DATAFAST_BIN:-npx --yes @datafast/cli}"

echo "DataFast website: $WEBSITE_ID"
echo "Period: $PERIOD"
echo

echo "== Overview =="
$DATAFAST_BIN --json analytics overview --website "$WEBSITE_ID" --period "$PERIOD"
echo

echo "== Goals =="
$DATAFAST_BIN --json analytics goals --website "$WEBSITE_ID" --period "$PERIOD"
echo

echo "== Top pages =="
$DATAFAST_BIN --json analytics pages --website "$WEBSITE_ID" --period "$PERIOD" --limit 10
echo

echo "== Referrers =="
$DATAFAST_BIN --json analytics referrers --website "$WEBSITE_ID" --period "$PERIOD" --limit 10
echo

echo "== Funnels =="
$DATAFAST_BIN --json funnels list "$WEBSITE_ID"

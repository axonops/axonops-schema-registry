#!/usr/bin/env bash
#
# generate-api-docs.sh -- Generate markdown API reference from the OpenAPI spec.
#
# Usage:
#   ./scripts/generate-api-docs.sh
#
# Prerequisites:
#   - Node.js (npx) for widdershins and @redocly/cli
#
# Outputs:
#   - docs/api-reference.md   (Markdown API reference)
#   - docs/api/index.html     (ReDoc interactive HTML)

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OPENAPI_SPEC="$REPO_ROOT/api/openapi.yaml"
OUTPUT_MD="$REPO_ROOT/docs/api-reference.md"
OUTPUT_HTML="$REPO_ROOT/docs/api/index.html"
TMPFILE="$(mktemp)"

trap 'rm -f "$TMPFILE"' EXIT

# ── Check prerequisites ──────────────────────────────────────────────

if ! command -v npx >/dev/null 2>&1; then
    echo "ERROR: npx is required but not found. Install Node.js first." >&2
    exit 1
fi

if [ ! -f "$OPENAPI_SPEC" ]; then
    echo "ERROR: OpenAPI spec not found at $OPENAPI_SPEC" >&2
    exit 1
fi

# ── Generate markdown via widdershins ─────────────────────────────────

echo "Generating markdown from $OPENAPI_SPEC ..."

npx --yes widdershins@4 "$OPENAPI_SPEC" -o "$TMPFILE" \
    --language_tabs 'shell:curl' \
    --summary \
    --omitHeader \
    --resolve \
    2>/dev/null

# ── Post-process: convert HTML tags to clean markdown ─────────────────
#
# Widdershins emits HTML heading tags, anchor tags, and other HTML elements.
# We convert these to standard markdown for compatibility with static site
# generators and GitHub rendering.

perl -0777 -pe '
    # Remove generator comment
    s/<!-- Generator: Widdershins[^>]*-->\n?//g;

    # Remove "Scroll down" instruction line
    s/^> Scroll down for code samples[^\n]*\n\n?//gm;

    # Convert HTML headings to markdown headings
    s|<h1 id="[^"]*">(.*?)</h1>|# $1|g;
    s|<h2 id="[^"]*">(.*?)</h2>|## $1|g;
    s|<h3 id="[^"]*">(.*?)</h3>|### $1|g;
    s|<h4 id="[^"]*">(.*?)</h4>|#### $1|g;
    s|<h5 id="[^"]*">(.*?)</h5>|##### $1|g;

    # Remove standalone anchor tags (operation IDs)
    s|<a id="[^"]*"></a>\n?||g;
    s|<a id="[^"]*"/>\n?||g;

    # Convert HTML links to markdown links
    s|<a href="([^"]*)">(.*?)</a>|[$2]($1)|g;

    # Convert aside blocks to blockquotes
    s|<aside class="warning">|> **Warning:** |g;
    s|<aside class="notice">|> **Note:** |g;
    s|<aside class="success">|> **Success:** |g;
    s|</aside>||g;

    # Remove stray HTML break tags
    s|<br\s*/?>||g;

    # Collapse runs of 3+ blank lines to 2
    s/\n{4,}/\n\n\n/g;
' "$TMPFILE" > "$OUTPUT_MD.tmp"

# Prepend auto-generation header
{
    printf '%s\n' '<!-- This file is auto-generated from api/openapi.yaml -->'
    printf '%s\n' '<!-- Do not edit manually. Regenerate with: make docs-api -->'
    printf '\n'
    cat "$OUTPUT_MD.tmp"
} > "$OUTPUT_MD"

rm -f "$OUTPUT_MD.tmp"

echo "  -> $OUTPUT_MD ($(wc -l < "$OUTPUT_MD" | tr -d ' ') lines)"

# ── Generate ReDoc HTML ───────────────────────────────────────────────

echo "Generating ReDoc HTML from $OPENAPI_SPEC ..."
mkdir -p "$(dirname "$OUTPUT_HTML")"
npx --yes @redocly/cli build-docs "$OPENAPI_SPEC" -o "$OUTPUT_HTML" 2>/dev/null

echo "  -> $OUTPUT_HTML ($(wc -c < "$OUTPUT_HTML" | tr -d ' ') bytes)"

echo "Done."

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
TMPFILE2="$(mktemp)"

trap 'rm -f "$TMPFILE" "$TMPFILE2"' EXIT

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
' "$TMPFILE" > "$TMPFILE2"

# ── Generate table of contents ────────────────────────────────────────
#
# Extract # and ## headings (outside code fences) and build a TOC.
# Code fences (```) toggle an in_fence flag so shell comments like
# "# You can also use wget" are excluded.

generate_toc() {
    perl -ne '
        BEGIN { $in_fence = 0; }
        if (/^```/) { $in_fence = !$in_fence; next; }
        next if $in_fence;
        if (/^(#{1,2})\s+(.+)/) {
            my $level = length($1);
            my $title = $2;
            # Build GitHub-style anchor: lowercase, spaces to hyphens, strip non-alnum
            my $anchor = lc($title);
            $anchor =~ s/\s+/-/g;
            $anchor =~ s/[^a-z0-9_-]//g;
            $anchor =~ s/-+/-/g;
            $anchor =~ s/^-|-$//g;
            my $indent = ($level == 1) ? "" : "  ";
            print "${indent}- [${title}](#${anchor})\n";
        }
    ' "$1"
}

TOC="$(generate_toc "$TMPFILE2")"

# ── Assemble final output ─────────────────────────────────────────────
#
# Insert the TOC after the title (first # heading) and its description block.
# We find the second # heading and insert the TOC before it.

{
    printf '%s\n' '<!-- This file is auto-generated from api/openapi.yaml -->'
    printf '%s\n' '<!-- Do not edit manually. Regenerate with: make docs-api -->'
    printf '\n'

    # Use perl to insert TOC right after the title heading and its description,
    # before the first ## heading (Key Concepts, Content Types, etc.)
    perl -e '
        use strict;
        use warnings;
        my $toc = $ENV{"TOC"};
        my $content = do { local $/; open my $fh, "<", $ARGV[0] or die $!; <$fh> };

        my @lines = split /\n/, $content;
        my $in_fence = 0;
        my $found_title = 0;
        my $insert_at = -1;

        for my $i (0..$#lines) {
            if ($lines[$i] =~ /^```/) {
                $in_fence = !$in_fence;
                next;
            }
            next if $in_fence;
            # Find the first # title heading
            if (!$found_title && $lines[$i] =~ /^# /) {
                $found_title = 1;
                next;
            }
            # After finding the title, insert TOC before the first ## heading
            if ($found_title && $lines[$i] =~ /^## /) {
                $insert_at = $i;
                last;
            }
        }

        if ($insert_at > 0) {
            # Print title and its description paragraph
            for my $i (0..($insert_at-1)) {
                print "$lines[$i]\n";
            }
            # Insert TOC
            print "\n## Contents\n\n";
            print "$toc\n\n";
            # Print remaining lines (starting from the first ## heading)
            for my $i ($insert_at..$#lines) {
                print "$lines[$i]\n";
            }
        } else {
            # Fallback: print as-is
            print "$content";
        }
    ' "$TMPFILE2"
} > "$OUTPUT_MD"

echo "  -> $OUTPUT_MD ($(wc -l < "$OUTPUT_MD" | tr -d ' ') lines)"

# ── Generate ReDoc HTML ───────────────────────────────────────────────

echo "Generating ReDoc HTML from $OPENAPI_SPEC ..."
mkdir -p "$(dirname "$OUTPUT_HTML")"
npx --yes @redocly/cli build-docs "$OPENAPI_SPEC" -o "$OUTPUT_HTML" 2>/dev/null

echo "  -> $OUTPUT_HTML ($(wc -c < "$OUTPUT_HTML" | tr -d ' ') bytes)"

echo "Done."

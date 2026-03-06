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

    # Strip version suffix from the title heading (e.g. "v1.0.0")
    s/^(# .+?) v\d+\.\d+\.\d+\s*$/$1/gm;

    # Collapse runs of 3+ blank lines to 2
    s/\n{4,}/\n\n\n/g;
' "$TMPFILE" > "$TMPFILE2"

# ── Append API Compatibility Reference section ────────────────────────
#
# Extract the x-compatibility marker from each tag in the OpenAPI spec
# and generate grouped endpoint tables. Uses Node.js (already required
# by this script for widdershins/redocly).

{
    node -e '
        const fs = require("fs");
        const yaml = require(require.resolve("yaml", {paths: [process.cwd(), __dirname]}));

        const spec = yaml.parse(fs.readFileSync(process.argv[1], "utf8"));
        const tags = spec.tags || [];

        // Group tags by compatibility tier
        const tiers = {
            "confluent-community": { label: "Confluent Compatible (Community)", tags: {} },
            "confluent-enterprise": { label: "Confluent Compatible (Enterprise)", tags: {} },
            "axonops": { label: "AxonOps Extensions", tags: {} },
        };

        for (const t of tags) {
            const compat = t["x-compatibility"] || "unknown";
            if (tiers[compat]) {
                let desc = t.description || "";
                desc = desc.replace(/^\*\*Confluent compatible \(Community\)\.\*\*\s*/, "");
                desc = desc.replace(/^\*\*Confluent compatible \(Enterprise\)\.\*\*\s*/, "");
                desc = desc.replace(/^\*\*AxonOps extension\.\*\*\s*/, "");
                tiers[compat].tags[t.name] = desc;
            }
        }

        // Collect endpoints per tag
        const tagEps = {};
        for (const [path, methods] of Object.entries(spec.paths || {}).sort()) {
            for (const [method, op] of Object.entries(methods).sort()) {
                if (method.startsWith("x-") || method === "parameters") continue;
                for (const tag of (op.tags || [])) {
                    if (!tagEps[tag]) tagEps[tag] = [];
                    tagEps[tag].push({
                        method: method.toUpperCase(),
                        path,
                        summary: (op.summary || "").replace(/\|/g, "\\|"),
                    });
                }
            }
        }

        let out = "\n---\n\n";
        out += "## API Compatibility Reference\n\n";
        out += "AxonOps Schema Registry implements the full Confluent Schema Registry API and\n";
        out += "extends it with additional capabilities. Each endpoint group below indicates\n";
        out += "its compatibility tier.\n\n";
        out += "| Tier | Description |\n";
        out += "|------|-------------|\n";
        out += "| **Confluent Compatible (Community)** | Available in the free/open-source Confluent Schema Registry |\n";
        out += "| **Confluent Compatible (Enterprise)** | Requires a Confluent Enterprise license — included free in AxonOps |\n";
        out += "| **AxonOps Extension** | Unique to AxonOps Schema Registry |\n\n";

        for (const [compat, tier] of Object.entries(tiers)) {
            const tagNames = Object.keys(tier.tags).sort();
            if (tagNames.length === 0) continue;

            out += "### " + tier.label + "\n\n";
            for (const tag of tagNames) {
                const eps = tagEps[tag] || [];
                out += "#### " + tag + "\n\n";
                out += tier.tags[tag] + "\n\n";
                if (eps.length > 0) {
                    out += "| Method | Endpoint | Description |\n";
                    out += "|--------|----------|-------------|\n";
                    for (const ep of eps) {
                        out += "| `" + ep.method + "` | `" + ep.path + "` | " + ep.summary + " |\n";
                    }
                    out += "\n";
                }
            }
        }
        process.stdout.write(out);
    ' "$OPENAPI_SPEC" 2>/dev/null || {
        # Fallback: if Node yaml module unavailable, generate a static section.
        echo ""
        echo "---"
        echo ""
        echo "## API Compatibility Reference"
        echo ""
        echo "AxonOps Schema Registry implements the full Confluent Schema Registry API and"
        echo "extends it with additional capabilities. See the tag sections above for full"
        echo "endpoint details. Each tag description indicates its compatibility tier:"
        echo ""
        echo "- **Confluent Compatible (Community):** Schemas, Subjects, Config, Mode, Compatibility, Metadata, Health"
        echo "- **Confluent Compatible (Enterprise):** Import, Contexts, Exporters, DEK Registry"
        echo "- **AxonOps Extension:** Analysis, Admin, Account, Documentation"
    }
} >> "$TMPFILE2"

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
export TOC

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

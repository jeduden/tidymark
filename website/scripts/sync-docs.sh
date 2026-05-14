#!/usr/bin/env bash
# Sync ../docs/ into content/docs/ for the Hugo build.
#
# Steps:
#   1. (optional, on by default) run `mdsmith fix` against the source
#      docs/ so every <?catalog?> and <?include?> body is current.
#      Skip with --no-fix when you want to lint-check without mutating
#      the source tree.
#   2. Mirror docs/ into content/docs/ — this directory is .gitignored
#      and treated as a build artifact.
#   3. Escape literal Hugo shortcode patterns ({{< ... >}}, {{% ... %}})
#      in the copy so Hugo renders them as text instead of trying to
#      parse them. The source markdown is left untouched.
#
# Run from the website/ directory:
#   ./scripts/sync-docs.sh           # mdsmith fix + sync + escape
#   ./scripts/sync-docs.sh --no-fix  # sync + escape only

set -euo pipefail

run_fix=1
for arg in "$@"; do
  case "$arg" in
    --no-fix) run_fix=0 ;;
    -h|--help)
      sed -n '2,18p' "$0" | sed 's/^# \?//'
      exit 0 ;;
    *) echo "unknown flag: $arg" >&2; exit 2 ;;
  esac
done

here="$(cd "$(dirname "$0")/.." && pwd)"
repo="$(cd "$here/.." && pwd)"
src="$repo/docs"
dst="$here/content/docs"

if [[ ! -d "$src" ]]; then
  echo "source not found: $src" >&2
  exit 1
fi

if (( run_fix )); then
  echo "==> mdsmith fix $src"
  (cd "$repo" && go run ./cmd/mdsmith fix ./docs) || {
    echo "mdsmith fix failed" >&2
    exit 1
  }
fi

echo "==> sync $src -> $dst"
rm -rf "$dst"
mkdir -p "$dst"
# Use cp -R rather than rsync to keep the dependency set minimal.
cp -R "$src/." "$dst/"

# proto.md files are schema templates — their front matter holds CUE
# constraint strings, not real values. Hugo tries to parse them as page
# metadata and fails. Drop them from the synced copy.
echo "==> drop proto.md schema templates from copy"
find "$dst" -type f -name "proto.md" -delete

# Hugo distinguishes index.md (leaf bundle — siblings become resources
# of THIS page) from _index.md (section overview — siblings become
# pages under THIS one). The repo uses index.md for the latter sense
# (a directory overview), so rename to _index.md before Hugo sees it.
echo "==> rename index.md -> _index.md (section overviews)"
find "$dst" -type f -name "index.md" -print0 |
  while IFS= read -r -d '' f; do
    mv "$f" "$(dirname "$f")/_index.md"
  done

# Strip non-markdown content that travelled along (Go embed files etc.)
# Keep only .md and image-like assets; everything else is repo plumbing
# that has no place in the rendered site.
echo "==> prune non-content files from copy"
find "$dst" -type f \
  ! -name "*.md" \
  ! -name "*.svg" ! -name "*.png" ! -name "*.jpg" ! -name "*.jpeg" \
  ! -name "*.gif" ! -name "*.webp" \
  -delete
find "$dst" -type d -empty -delete

echo "==> escape Hugo shortcode patterns in $dst"
# {{< shortcode >}}  ->  {{</* shortcode */>}}
# {{% shortcode %}}  ->  {{%/* shortcode */%}}
# These are Hugo's documented escape forms for literal shortcode text.
find "$dst" -type f -name "*.md" -print0 |
  xargs -0 sed -i -E \
    -e 's|\{\{<([^}]*)>\}\}|{{</*\1*/>}}|g' \
    -e 's|\{\{%([^}]*)%\}\}|{{%/*\1*/%}}|g'

echo "==> done"

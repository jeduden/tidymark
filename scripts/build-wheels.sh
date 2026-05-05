#!/usr/bin/env bash
# build-wheels.sh — assemble one platform-tagged wheel per supported
# host from the prebuilt binaries the release workflow downloaded as
# GitHub artifacts. Each wheel bundles the matching binary under
# mdsmith/_bin/ and ships an `mdsmith` console script.
#
# Run scripts/set-version.sh BEFORE this script so python/pyproject.toml
# already pins the published version.
#
# Usage:
#   scripts/build-wheels.sh <artifacts-dir> <out-dir>

set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "usage: $0 <artifacts-dir> <out-dir>" >&2
  exit 2
fi

artifacts="$1"
out_abs="$(mkdir -p "$2" && cd "$2" && pwd)"

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
src="$repo_root/python"

# (release-asset, wheel-platform-tag, executable-name) — the wheel
# tag mirrors what `pip` uses to pick the right artifact, and stays
# in lock-step with the build matrix in
# .github/workflows/release.yml.
build_one() {
  local asset="$1"
  local plat_tag="$2"
  local exe="$3"

  local asset_path="$artifacts/$asset"
  if [ ! -f "$asset_path" ]; then
    echo "missing release asset: $asset_path" >&2
    exit 1
  fi

  # Build each wheel from a clean copy of python/ so the staged
  # binary does not bleed across platform tags. A subshell with an
  # EXIT trap guarantees the temp dir is removed even when a
  # downstream command (python -m build, python -m wheel) exits
  # non-zero under `set -e`. A plain `trap … RETURN` would only
  # fire on a normal return and leak the dir on failure.
  (
    local stage
    stage="$(mktemp -d)"
    trap 'rm -rf "$stage"' EXIT

    cp -R "$src/." "$stage/"
    mkdir -p "$stage/mdsmith/_bin"
    install -m 0755 "$asset_path" "$stage/mdsmith/_bin/$exe"

    # `python -m build --wheel` honours pyproject.toml. Hatchling
    # cannot infer a platform tag on its own when the binary is
    # staged at build time, so the wheel comes out as
    # `*-py3-none-any.whl`. `python -m wheel tags --platform-tag`
    # rewrites both the filename AND the dist-info/WHEEL metadata
    # so PyPI and pip see a consistent platform tag.
    (cd "$stage" && python -m build --wheel --outdir "$out_abs/.staging-$plat_tag")
    for whl in "$out_abs/.staging-$plat_tag"/*.whl; do
      python -m wheel tags --remove --platform-tag "$plat_tag" "$whl"
    done
    mv "$out_abs/.staging-$plat_tag"/*.whl "$out_abs/"
    rmdir "$out_abs/.staging-$plat_tag"
  )
}

build_one "mdsmith-linux-amd64"        "manylinux_2_17_x86_64.manylinux2014_x86_64"  "mdsmith"
build_one "mdsmith-linux-arm64"        "manylinux_2_17_aarch64.manylinux2014_aarch64" "mdsmith"
build_one "mdsmith-darwin-amd64"       "macosx_11_0_x86_64"                          "mdsmith"
build_one "mdsmith-darwin-arm64"       "macosx_11_0_arm64"                           "mdsmith"
build_one "mdsmith-windows-amd64.exe"  "win_amd64"                                   "mdsmith.exe"

echo "built wheels under $out_abs:"
ls -1 "$out_abs"

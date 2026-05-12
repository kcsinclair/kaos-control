#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-or-later
#
# package-release.sh — bundle each cross-compiled binary in dist/ into a
# zip archive containing a `kaos-control/` directory with the binary
# (renamed to `kaos-control`), README, LICENSE, and CONTRIBUTING.
#
# Version comes from the top-level `VERSION` file and is embedded in
# every archive filename: `kaos-control-<version>-<os>-<arch>.zip`.
#
# Expected layout BEFORE running (produced by `make release`):
#   dist/kaos-control-linux-amd64
#   dist/kaos-control-linux-arm64
#   dist/kaos-control-darwin-amd64
#   dist/kaos-control-darwin-arm64
#   dist/kaos-control-windows-amd64.exe
#
# Layout produced by this script (for VERSION=0.1.1):
#   dist/kaos-control-0.1.1-linux-amd64.zip
#   dist/kaos-control-0.1.1-linux-arm64.zip
#   dist/kaos-control-0.1.1-darwin-amd64.zip
#   dist/kaos-control-0.1.1-darwin-arm64.zip
#   dist/kaos-control-0.1.1-windows-amd64.zip
#
# Each zip extracts to:
#   kaos-control/
#   ├── kaos-control      (or kaos-control.exe on Windows; mode 0755)
#   ├── README.md
#   ├── LICENSE
#   └── CONTRIBUTING.md
#
# Also writes `dist/SHA256SUMS` listing every produced zip, in the format
# accepted by `sha256sum -c SHA256SUMS` (or `shasum -a 256 -c SHA256SUMS`).
#
# Usage:
#   make release
#   scripts/package-release.sh
#
# Or via the top-level Makefile target:
#   make package
#
# Exit status:
#   0 — at least one archive produced
#   1 — fatal error (missing dependency, README, dist dir, or zero archives)

set -euo pipefail

# Resolve repo root from the script's own location so this works from any cwd.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DIST_DIR="$REPO_ROOT/dist"
VERSION_FILE="$REPO_ROOT/VERSION"

# Files copied into every zip alongside the binary.
DOC_FILES=(
    "README.md"
    "LICENSE"
    "CONTRIBUTING.md"
)

PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

# --- Sanity checks ---------------------------------------------------------

if ! command -v zip >/dev/null 2>&1; then
    echo "ERROR: 'zip' is not installed. Install with:" >&2
    echo "  macOS:  brew install zip          # usually preinstalled" >&2
    echo "  Debian: sudo apt-get install zip" >&2
    exit 1
fi

for f in "${DOC_FILES[@]}"; do
    if [[ ! -f "$REPO_ROOT/$f" ]]; then
        echo "ERROR: $f not found at $REPO_ROOT/$f" >&2
        exit 1
    fi
done

if [[ ! -f "$VERSION_FILE" ]]; then
    echo "ERROR: VERSION file not found at $VERSION_FILE" >&2
    exit 1
fi
# Trim leading/trailing whitespace (including the trailing newline `cat`
# would leave in place) so the version slots cleanly into a filename.
VERSION="$(tr -d '[:space:]' < "$VERSION_FILE")"
if [[ -z "$VERSION" ]]; then
    echo "ERROR: VERSION file is empty" >&2
    exit 1
fi
echo "Packaging version: $VERSION"
echo

# Remove any stale zips (from prior runs at different versions) and the
# old SHA256SUMS file, so dist/ contains only the current release set.
rm -f "$DIST_DIR"/kaos-control-*.zip "$DIST_DIR/SHA256SUMS"

# Pick a SHA256 tool. macOS ships `shasum`; Linux usually has `sha256sum`.
sha256_cmd=""
if command -v sha256sum >/dev/null 2>&1; then
    sha256_cmd="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
    sha256_cmd="shasum -a 256"
else
    echo "ERROR: neither 'sha256sum' nor 'shasum' is installed." >&2
    exit 1
fi

if [[ ! -d "$DIST_DIR" ]]; then
    echo "ERROR: $DIST_DIR does not exist. Run 'make release' first." >&2
    exit 1
fi

# --- Stage + zip each platform --------------------------------------------

# Single temp dir; cleaned up on exit (including failures).
STAGE_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/kaos-control-package.XXXXXX")"
trap 'rm -rf "$STAGE_ROOT"' EXIT

produced=0
skipped=0

for platform in "${PLATFORMS[@]}"; do
    os="${platform%/*}"
    arch="${platform#*/}"
    ext=""
    [[ "$os" == "windows" ]] && ext=".exe"

    bin_src="$DIST_DIR/kaos-control-${os}-${arch}${ext}"
    if [[ ! -f "$bin_src" ]]; then
        printf '  skip %-22s (no binary at %s)\n' "${os}/${arch}" "$bin_src"
        skipped=$((skipped + 1))
        continue
    fi

    stage_dir="$STAGE_ROOT/${os}-${arch}/kaos-control"
    mkdir -p "$stage_dir"
    cp "$bin_src" "$stage_dir/kaos-control${ext}"
    chmod 0755 "$stage_dir/kaos-control${ext}"
    for f in "${DOC_FILES[@]}"; do
        cp "$REPO_ROOT/$f" "$stage_dir/$f"
    done

    zip_name="kaos-control-${VERSION}-${os}-${arch}.zip"
    zip_out="$DIST_DIR/$zip_name"
    rm -f "$zip_out"

    # Run zip from the parent of `kaos-control/` so the archive contains
    # `kaos-control/...` paths (not absolute paths or a flat layout).
    (cd "$STAGE_ROOT/${os}-${arch}" && zip -qr "$zip_out" kaos-control)

    size=$(du -h "$zip_out" | awk '{print $1}')
    printf '  ✓    %-22s → %s (%s)\n' "${os}/${arch}" "$zip_name" "$size"
    produced=$((produced + 1))
done

# --- Summary --------------------------------------------------------------

echo
echo "Produced $produced archive(s), skipped $skipped."

if [[ $produced -eq 0 ]]; then
    echo "ERROR: no archives produced. Did you run 'make release'?" >&2
    exit 1
fi

echo "Archives:"
ls -lh "$DIST_DIR"/kaos-control-${VERSION}-*.zip 2>/dev/null \
    | awk '{printf "  %-10s %s\n", $5, $9}'

# Generate SHA256SUMS in the dist dir so the file contains relative names
# (so `sha256sum -c SHA256SUMS` works when run from inside dist/).
sums_file="$DIST_DIR/SHA256SUMS"
rm -f "$sums_file"
(
    cd "$DIST_DIR"
    # shellcheck disable=SC2086  # word-splitting on $sha256_cmd is intentional
    $sha256_cmd kaos-control-${VERSION}-*.zip > "$sums_file"
)

echo
echo "Wrote $sums_file:"
sed 's/^/  /' "$sums_file"

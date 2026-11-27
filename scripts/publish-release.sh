#!/usr/bin/env bash
#
# publish-release.sh — create the GitHub release for the current VERSION and
# upload the packaged archives produced by `make package`.
#
# Naming is derived from the top-level `VERSION` file so every release is
# consistent:
#
#   tag:    v<VERSION>               e.g. v0.4.0
#   title:  kaos-control v<VERSION>  e.g. kaos-control v0.4.0
#   assets: dist/kaos-control-<VERSION>-<os>-<arch>.zip + dist/SHA256SUMS
#
# Usage:
#   make publish NOTES=path/to/release-notes.md            # draft (default)
#   make publish NOTES=path/to/release-notes.md DRAFT=0    # publish as latest
#
# Preconditions (each is checked; the script refuses to continue otherwise):
#   - `gh` is installed and authenticated
#   - the tag v<VERSION> exists locally and on origin
#   - every expected zip and SHA256SUMS exist in dist/ and the sums verify
#   - every packaged binary was built from the tagged commit (vcs.revision)
#   - no GitHub release already exists for the tag
#
# Typical release flow:
#   echo 0.5.0 > VERSION   (commit it)
#   git tag -a v0.5.0 -m v0.5.0 && git push origin v0.5.0
#   make build-web && make package
#   make publish NOTES=release-notes.md

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DIST_DIR="$REPO_ROOT/dist"
PLATFORMS=(linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64)

NOTES="${NOTES:-}"
DRAFT="${DRAFT:-1}"

die() { echo "ERROR: $*" >&2; exit 1; }

cd "$REPO_ROOT"

# --- Inputs ----------------------------------------------------------------

[[ -f VERSION ]] || die "VERSION file not found"
VERSION="$(tr -d '[:space:]' < VERSION)"
[[ -n "$VERSION" ]] || die "VERSION file is empty"
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.]+)?$ ]] \
    || die "VERSION '$VERSION' is not semver (expected e.g. 0.4.0)"

TAG="v${VERSION}"
TITLE="kaos-control v${VERSION}"

[[ -n "$NOTES" ]] || die "NOTES is required: make publish NOTES=path/to/notes.md"
[[ -s "$NOTES" ]] || die "release notes file '$NOTES' is missing or empty"

case "$DRAFT" in
    1|0) ;;
    *) die "DRAFT must be 1 (draft, default) or 0 (publish now)" ;;
esac

# --- Tooling ---------------------------------------------------------------

command -v gh >/dev/null 2>&1 || die "'gh' (GitHub CLI) is not installed"
gh auth status >/dev/null 2>&1 || die "'gh' is not authenticated; run: gh auth login"
command -v go >/dev/null 2>&1 || die "'go' is required to read build info from the binaries"

if command -v sha256sum >/dev/null 2>&1; then
    sha256_check=(sha256sum -c --quiet)
else
    sha256_check=(shasum -a 256 -c --quiet)
fi

# --- Tag -------------------------------------------------------------------

git rev-parse -q --verify "refs/tags/$TAG" >/dev/null \
    || die "tag $TAG does not exist locally. Create it with:
  git tag -a $TAG -m $TAG && git push origin $TAG"
TAG_COMMIT="$(git rev-list -n 1 "$TAG")"

REMOTE_COMMIT="$(git ls-remote origin "refs/tags/${TAG}^{}" | awk '{print $1}')"
[[ -n "$REMOTE_COMMIT" ]] || REMOTE_COMMIT="$(git ls-remote origin "refs/tags/${TAG}" | awk '{print $1}')"
[[ -n "$REMOTE_COMMIT" ]] || die "tag $TAG is not on origin. Push it with: git push origin $TAG"
[[ "$REMOTE_COMMIT" == "$TAG_COMMIT" ]] \
    || die "tag $TAG points at ${REMOTE_COMMIT:0:12} on origin but ${TAG_COMMIT:0:12} locally"

# --- Assets ----------------------------------------------------------------

assets=()
for p in "${PLATFORMS[@]}"; do
    os="${p%/*}"; arch="${p#*/}"
    ext=""; [[ "$os" == "windows" ]] && ext=".exe"

    zip="$DIST_DIR/kaos-control-${VERSION}-${os}-${arch}.zip"
    [[ -f "$zip" ]] || die "missing $zip — run: make build-web && make package"
    assets+=("$zip")

    bin="$DIST_DIR/kaos-control-${os}-${arch}${ext}"
    [[ -f "$bin" ]] || die "missing $bin — run: make package"
    rev="$(go version -m "$bin" | awk '$2 ~ /^vcs\.revision=/ {sub(/^vcs\.revision=/, "", $2); print $2}')"
    [[ "$rev" == "$TAG_COMMIT" ]] \
        || die "$bin was built from ${rev:0:12}, not $TAG (${TAG_COMMIT:0:12}). Check out the tag and re-run: make build-web && make package"
done

[[ -f "$DIST_DIR/SHA256SUMS" ]] || die "missing $DIST_DIR/SHA256SUMS — run: make package"
(cd "$DIST_DIR" && "${sha256_check[@]}" SHA256SUMS) || die "SHA256SUMS does not verify"
assets+=("$DIST_DIR/SHA256SUMS")

# --- Release ---------------------------------------------------------------

if gh release view "$TAG" >/dev/null 2>&1; then
    die "a GitHub release for $TAG already exists"
fi

flags=(--verify-tag --title "$TITLE" --notes-file "$NOTES")
if [[ "$DRAFT" == "1" ]]; then
    flags+=(--draft)
else
    flags+=(--latest)
fi

echo "Creating release '$TITLE' (tag $TAG @ ${TAG_COMMIT:0:12}, $([[ $DRAFT == 1 ]] && echo draft || echo published))"
for a in "${assets[@]}"; do echo "  + $(basename "$a")"; done

url="$(gh release create "$TAG" "${flags[@]}" "${assets[@]}")"
echo
echo "Release: $url"
if [[ "$DRAFT" == "1" ]]; then
    echo "Review it, then publish with:"
    echo "  gh release edit $TAG --draft=false --latest"
fi

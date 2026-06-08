#!/usr/bin/env bash
# Fill the Homebrew formula's SHA256 values from a published GitHub release.
#
# Usage: packaging/homebrew/update-formula.sh <version>      # e.g. 0.1.0
#
# Requires: gh (authenticated) or curl, plus shasum/sha256sum.
# Output: packaging/homebrew/portman-<version>.rb  (copy this into your tap).
set -euo pipefail

VERSION="${1:?usage: update-formula.sh <version>   e.g. 0.1.0}"
REPO="avlunvu/lvdtvd-portman"
HERE="$(cd "$(dirname "$0")" && pwd)"
TMP="$(mktemp -d)"
OUT="$HERE/portman-${VERSION}.rb"

sha() { # sha <file>
  if command -v sha256sum >/dev/null; then sha256sum "$1" | awk '{print $1}'
  else shasum -a 256 "$1" | awk '{print $1}'; fi
}

fetch() { # fetch <asset>
  local a="$1"
  if command -v gh >/dev/null; then
    gh release download "v$VERSION" -R "$REPO" -p "$a" -D "$TMP" --clobber
  else
    curl -fsSL -o "$TMP/$a" "https://github.com/$REPO/releases/download/v$VERSION/$a"
  fi
}

for a in portman-darwin-arm64 portman-darwin-amd64 portman-linux-amd64; do fetch "$a"; done

S_ARM=$(sha "$TMP/portman-darwin-arm64")
S_AMD=$(sha "$TMP/portman-darwin-amd64")
S_LNX=$(sha "$TMP/portman-linux-amd64")

sed \
  -e "s/^  version \".*\"/  version \"$VERSION\"/" \
  -e "s/__SHA_DARWIN_ARM64__/$S_ARM/" \
  -e "s/__SHA_DARWIN_AMD64__/$S_AMD/" \
  -e "s/__SHA_LINUX_AMD64__/$S_LNX/" \
  "$HERE/portman.rb" > "$OUT"

rm -rf "$TMP"
echo "Wrote $OUT"
echo "Next: copy it into your tap repo as Formula/portman.rb and push:"
echo "  https://github.com/avlunvu/homebrew-tap"

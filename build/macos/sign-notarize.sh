#!/usr/bin/env bash
# Code-sign (Developer ID, hardened runtime) and notarize a macOS .app, then
# staple the notarization ticket so it launches with no Gatekeeper warning —
# even when downloaded via a browser.
#
# Usage: sign-notarize.sh <path-to-.app>
#
# Required environment variables (provide via GitHub Actions secrets):
#   APPLE_CERT_P12_BASE64   base64 of your Developer ID Application .p12
#   APPLE_CERT_PASSWORD     password for that .p12
#   APPLE_SIGNING_IDENTITY  e.g. "Developer ID Application: Your Name (TEAMID)"
#   APPLE_ID                Apple ID email used for notarization
#   APPLE_APP_PASSWORD      app-specific password (appleid.apple.com)
#   APPLE_TEAM_ID           10-char Apple Developer Team ID
set -euo pipefail

APP="${1:?usage: sign-notarize.sh <path-to-.app>}"
: "${APPLE_CERT_P12_BASE64:?}" "${APPLE_CERT_PASSWORD:?}" "${APPLE_SIGNING_IDENTITY:?}"
: "${APPLE_ID:?}" "${APPLE_APP_PASSWORD:?}" "${APPLE_TEAM_ID:?}"

TMP="${RUNNER_TEMP:-$(mktemp -d)}"
KEYCHAIN="$TMP/portman-signing.keychain-db"
KEYCHAIN_PASS="$(uuidgen)"
CERT="$TMP/cert.p12"

cleanup() { security delete-keychain "$KEYCHAIN" 2>/dev/null || true; rm -f "$CERT"; }
trap cleanup EXIT

# 1. Import the signing certificate into a throwaway keychain.
echo "$APPLE_CERT_P12_BASE64" | base64 --decode > "$CERT"
security create-keychain -p "$KEYCHAIN_PASS" "$KEYCHAIN"
security set-keychain-settings -lut 21600 "$KEYCHAIN"
security unlock-keychain -p "$KEYCHAIN_PASS" "$KEYCHAIN"
security import "$CERT" -P "$APPLE_CERT_PASSWORD" -A -t cert -f pkcs12 -k "$KEYCHAIN"
security set-key-partition-list -S apple-tool:,apple:,codesign: -s -k "$KEYCHAIN_PASS" "$KEYCHAIN" >/dev/null
# Make the keychain searchable for codesign.
security list-keychain -d user -s "$KEYCHAIN" $(security list-keychains -d user | tr -d '"')

# 2. Sign with the hardened runtime + secure timestamp.
codesign --force --deep --options runtime --timestamp \
  --sign "$APPLE_SIGNING_IDENTITY" "$APP"
codesign --verify --strict --verbose=2 "$APP"

# 3. Notarize: zip the .app, submit, wait, then staple the ticket.
NZIP="$TMP/notarize.zip"
ditto -c -k --keepParent "$APP" "$NZIP"
xcrun notarytool submit "$NZIP" \
  --apple-id "$APPLE_ID" \
  --password "$APPLE_APP_PASSWORD" \
  --team-id "$APPLE_TEAM_ID" \
  --wait
xcrun stapler staple "$APP"
xcrun stapler validate "$APP"

echo "signed + notarized + stapled: $APP"

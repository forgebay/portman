# macOS code-signing & notarization

When the secrets below are present, the release workflow signs the `.app` with a
Developer ID certificate (hardened runtime), notarizes it with Apple, and
staples the ticket — so users can open it with **no Gatekeeper warning**, even
after downloading via a browser. Without the secrets, CI ships the unsigned
`.app` as before (right-click → Open on first launch).

Requires a paid **Apple Developer Program** membership ($99/yr).

## Repository secrets to create

In GitHub → repo → **Settings → Secrets and variables → Actions → New
repository secret**, add:

| Secret | What it is |
|--------|-----------|
| `APPLE_CERT_P12_BASE64` | Your *Developer ID Application* certificate exported as `.p12`, base64-encoded |
| `APPLE_CERT_PASSWORD` | Password you set when exporting the `.p12` |
| `APPLE_SIGNING_IDENTITY` | e.g. `Developer ID Application: Jane Doe (ABCDE12345)` |
| `APPLE_ID` | Your Apple ID email |
| `APPLE_APP_PASSWORD` | App-specific password from appleid.apple.com → Sign-In & Security |
| `APPLE_TEAM_ID` | 10-char Team ID (developer.apple.com → Membership) |

## How to get each value

1. **Certificate** — in *Keychain Access* (or Xcode → Settings → Accounts →
   Manage Certificates) create/locate a **Developer ID Application** cert.
   Export it (with its private key) as `Certificates.p12`, then:
   ```sh
   base64 -i Certificates.p12 | pbcopy   # paste into APPLE_CERT_P12_BASE64
   ```
2. **Signing identity** — the exact certificate name:
   ```sh
   security find-identity -v -p codesigning
   ```
3. **App-specific password** — appleid.apple.com → Sign-In & Security →
   App-Specific Passwords → generate one.
4. **Team ID** — developer.apple.com/account → Membership details.

## Verify locally (optional)

```sh
export APPLE_CERT_P12_BASE64=... APPLE_CERT_PASSWORD=... APPLE_SIGNING_IDENTITY=... \
       APPLE_ID=... APPLE_APP_PASSWORD=... APPLE_TEAM_ID=...
go build -o portman ./cmd/portman
bash build/macos/make-bundle.sh ./portman 0.1.0
bash build/macos/sign-notarize.sh "dist/Port Manager.app"
spctl -a -vvv "dist/Port Manager.app"   # should say: accepted, source=Notarized Developer ID
```

> Note: the raw per-platform binaries used by npm/Homebrew are downloaded over
> HTTPS (curl), which does not set the quarantine flag, so they typically run
> without a Gatekeeper prompt. Signing/notarization above targets the
> browser-downloaded `.app`.

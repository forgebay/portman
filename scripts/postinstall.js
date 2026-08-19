#!/usr/bin/env node
// Downloads the prebuilt portman binary that matches this package version and
// the current platform/arch from GitHub Releases, into bin/. The CLI launcher
// (bin/cli.js) then execs it. CGO means we cannot build at install time, so we
// fetch the artifact produced by the release workflow.
//
// We resolve and download the asset via the GitHub *API* (api.github.com),
// which is more reliable than the browser download URL
// (github.com/.../releases/download/...) — the latter intermittently 504s.
// Every request is retried with backoff to ride out transient failures.
'use strict';

const fs = require('fs');
const path = require('path');
const https = require('https');

const REPO = 'forgebay/portman';
const { version } = require('../package.json');

const PLATFORM = { darwin: 'darwin', linux: 'linux' }[process.platform];
const ARCH = { x64: 'amd64', arm64: 'arm64' }[process.arch];
const RETRIES = 4;

function fail(msg) {
  console.error('\n[portman] ' + msg);
  console.error('[portman] You can also download a binary manually from:');
  console.error('          https://github.com/' + REPO + '/releases\n');
  process.exit(1);
}

if (!PLATFORM || !ARCH) {
  fail(`Unsupported platform ${process.platform}/${process.arch} (supported: macOS & Linux, x64/arm64).`);
}

const asset = `portman-${PLATFORM}-${ARCH}`;
const binDir = path.join(__dirname, '..', 'bin');
const dest = path.join(binDir, asset);
fs.mkdirSync(binDir, { recursive: true });

const UA = { 'User-Agent': 'portman-postinstall' };
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// GET a URL following redirects. Resolves to { status, headers, stream } on a
// final (non-redirect) response; the caller consumes the body.
function get(url, headers) {
  return new Promise((resolve, reject) => {
    const req = https.get(url, { headers: { ...UA, ...headers } }, (res) => {
      const { statusCode } = res;
      if ([301, 302, 303, 307, 308].includes(statusCode) && res.headers.location) {
        res.resume();
        return resolve(get(res.headers.location, headers));
      }
      resolve({ status: statusCode, headers: res.headers, stream: res });
    });
    req.on('error', reject);
    req.setTimeout(30000, () => req.destroy(new Error('request timed out')));
  });
}

// Run fn with retries; retry on thrown errors and on the 5xx/429 we map to them.
async function withRetry(label, fn) {
  let lastErr;
  for (let attempt = 1; attempt <= RETRIES; attempt++) {
    try {
      return await fn();
    } catch (e) {
      lastErr = e;
      const wait = 1000 * attempt;
      console.error(`[portman] ${label} failed (attempt ${attempt}/${RETRIES}): ${e.message}. Retrying in ${wait / 1000}s...`);
      await sleep(wait);
    }
  }
  throw lastErr;
}

function readJSON(stream) {
  return new Promise((resolve, reject) => {
    let body = '';
    stream.on('data', (c) => (body += c));
    stream.on('end', () => {
      try { resolve(JSON.parse(body)); } catch (e) { reject(e); }
    });
    stream.on('error', reject);
  });
}

async function resolveAssetApiUrl() {
  const api = `https://api.github.com/repos/${REPO}/releases/tags/v${version}`;
  const res = await get(api, { Accept: 'application/vnd.github+json' });
  if (res.status !== 200) {
    res.stream.resume();
    throw new Error(`release lookup HTTP ${res.status} (v${version} may not be published yet)`);
  }
  const rel = await readJSON(res.stream);
  const found = (rel.assets || []).find((a) => a.name === asset);
  if (!found) throw new Error(`asset ${asset} not found in release v${version}`);
  return found.url; // API asset URL; fetch with Accept: application/octet-stream
}

async function downloadTo(url, file) {
  const res = await get(url, { Accept: 'application/octet-stream' });
  if (res.status !== 200) {
    res.stream.resume();
    throw new Error(`download HTTP ${res.status}`);
  }
  await new Promise((resolve, reject) => {
    const out = fs.createWriteStream(file, { mode: 0o755 });
    res.stream.pipe(out);
    out.on('finish', () => out.close(resolve));
    out.on('error', reject);
  });
}

(async () => {
  try {
    const assetUrl = await withRetry('release lookup', resolveAssetApiUrl);
    await withRetry('binary download', () => downloadTo(assetUrl, dest));
    fs.chmodSync(dest, 0o755);
    console.log(`[portman] Installed ${asset} (v${version}).`);
    if (PLATFORM === 'linux') {
      console.log('[portman] Linux runtime deps if the tray icon is missing:');
      console.log('          sudo apt-get install -y libgtk-3-0 libayatana-appindicator3-1');
    }
  } catch (e) {
    fail(`Could not install the prebuilt binary: ${e.message}`);
  }
})();

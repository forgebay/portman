#!/usr/bin/env node
// Downloads the prebuilt portman binary that matches this package version and
// the current platform/arch from GitHub Releases, into bin/. The CLI launcher
// (bin/cli.js) then execs it. CGO means we cannot build at install time, so we
// fetch the artifact produced by the release workflow.
'use strict';

const fs = require('fs');
const path = require('path');
const https = require('https');

const REPO = 'avlunvu/lvdtvd-portman';
const { version } = require('../package.json');

// Map Node's platform/arch to the release asset suffix.
const PLATFORM = { darwin: 'darwin', linux: 'linux' }[process.platform];
const ARCH = { x64: 'amd64', arm64: 'arm64' }[process.arch];

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
const url = `https://github.com/${REPO}/releases/download/v${version}/${asset}`;
const binDir = path.join(__dirname, '..', 'bin');
const dest = path.join(binDir, asset);

fs.mkdirSync(binDir, { recursive: true });

function download(u, file, redirects = 0) {
  if (redirects > 10) return fail('Too many redirects while downloading binary.');
  https.get(u, { headers: { 'User-Agent': 'portman-postinstall' } }, (res) => {
    if ([301, 302, 303, 307, 308].includes(res.statusCode)) {
      res.resume();
      return download(res.headers.location, file, redirects + 1);
    }
    if (res.statusCode !== 200) {
      res.resume();
      return fail(`Download failed (HTTP ${res.statusCode}) for ${u}\n          The release for v${version} may not be published yet.`);
    }
    const out = fs.createWriteStream(file, { mode: 0o755 });
    res.pipe(out);
    out.on('finish', () => out.close(() => {
      fs.chmodSync(file, 0o755);
      console.log(`[portman] Installed ${asset} (v${version}).`);
      if (PLATFORM === 'linux') {
        console.log('[portman] Linux runtime deps if the tray icon is missing:');
        console.log('          sudo apt-get install -y libgtk-3-0 libayatana-appindicator3-1');
      }
    }));
    out.on('error', (e) => fail('Write error: ' + e.message));
  }).on('error', (e) => fail('Network error: ' + e.message));
}

download(url, dest);

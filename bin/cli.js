#!/usr/bin/env node
// Launches the prebuilt portman binary (downloaded by postinstall) detached, so
// the tray app keeps running on its own and this CLI process exits immediately.
'use strict';

const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');

const PLATFORM = { darwin: 'darwin', linux: 'linux' }[process.platform];
const ARCH = { x64: 'amd64', arm64: 'arm64' }[process.arch];
const bin = path.join(__dirname, `portman-${PLATFORM}-${ARCH}`);

if (!PLATFORM || !ARCH || !fs.existsSync(bin)) {
  console.error('[portman] Binary not found. Try reinstalling:');
  console.error('          npm install -g @avlunvu/portman');
  process.exit(1);
}

const child = spawn(bin, process.argv.slice(2), {
  detached: true,
  stdio: 'ignore',
});
child.unref();
console.log('[portman] started — look for the icon in your menu bar / tray.');

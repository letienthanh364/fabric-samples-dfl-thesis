#!/usr/bin/env node
'use strict';

/**
 * Sign every unsigned VC JSON located under nodes-setup/vc-unsigned using the
 * admin Ed25519 private key and store the signed documents under
 * nodes-setup/vc-signed.
 *
 * Usage:
 *   node scripts/sign-trainer-vcs.js
 *   node scripts/sign-trainer-vcs.js --key nodes-setup/keys/admin_ed25519_sk.pem --force
 *
 * Flags:
 *   --key <path>         Path to admin Ed25519 private key PEM.
 *   --unsigned <dir>     Directory containing unsigned VC JSON files.
 *   --signed <dir>       Output directory for signed VC JSON files.
 *   --vctool <path>      Path to the vctool binary. Default: ./api/vctool
 *   --force              Overwrite existing signed VC files.
 */

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

const ROOT = path.resolve(__dirname, '..');
const DEFAULT_KEY = path.join(ROOT, 'nodes-setup', 'keys', 'admin_ed25519_sk.pem');
const DEFAULT_UNSIGNED_DIR = path.join(ROOT, 'nodes-setup', 'vc-unsigned');
const DEFAULT_SIGNED_DIR = path.join(ROOT, 'nodes-setup', 'vc-signed');
const DEFAULT_VCTOOL = path.join(ROOT, 'api', 'vctool');

function parseArgs(argv) {
  const opts = {
    key: DEFAULT_KEY,
    unsignedDir: DEFAULT_UNSIGNED_DIR,
    signedDir: DEFAULT_SIGNED_DIR,
    vctool: DEFAULT_VCTOOL,
    force: false,
  };

  for (let i = 2; i < argv.length; i += 1) {
    const arg = argv[i];
    switch (arg) {
      case '--key':
        opts.key = path.resolve(argv[++i]);
        break;
      case '--unsigned':
        opts.unsignedDir = path.resolve(argv[++i]);
        break;
      case '--signed':
        opts.signedDir = path.resolve(argv[++i]);
        break;
      case '--vctool':
        opts.vctool = path.resolve(argv[++i]);
        break;
      case '--force':
        opts.force = true;
        break;
      default:
        console.error(`Unknown flag: ${arg}`);
        process.exit(1);
    }
  }
  return opts;
}

function ensurePaths(opts) {
  if (!fs.existsSync(opts.key)) {
    console.error(`Admin key not found at ${opts.key}`);
    process.exit(1);
  }
  if (!fs.existsSync(opts.vctool)) {
    console.error(`vctool binary not found at ${opts.vctool}`);
    process.exit(1);
  }
  if (!fs.existsSync(opts.unsignedDir)) {
    console.error(`Unsigned VC directory not found: ${opts.unsignedDir}`);
    process.exit(1);
  }
  fs.mkdirSync(opts.signedDir, { recursive: true });
}

function getUnsignedFiles(dir) {
  return fs.readdirSync(dir)
    .filter((file) => file.endsWith('.json'))
    .sort()
    .map((file) => path.join(dir, file));
}

function signVc(unsignedPath, signedPath, opts) {
  if (!opts.force && fs.existsSync(signedPath)) {
    return { unsignedPath, signedPath, created: false };
  }
  const args = ['-vc', unsignedPath, '-key', opts.key, '-out', signedPath];
  const result = spawnSync(opts.vctool, args, {
    cwd: ROOT,
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  if (result.status !== 0) {
    throw new Error(`vctool failed for ${unsignedPath}: ${result.stderr || result.stdout}`);
  }
  return { unsignedPath, signedPath, created: true };
}

function main() {
  const opts = parseArgs(process.argv);
  ensurePaths(opts);
  const unsignedFiles = getUnsignedFiles(opts.unsignedDir);
  if (!unsignedFiles.length) {
    console.error(`No unsigned VC files found in ${opts.unsignedDir}`);
    process.exit(1);
  }

  const summary = [];
  unsignedFiles.forEach((unsignedPath) => {
    const basename = path.basename(unsignedPath);
    const baseNoExt = basename.replace(/\.json$/i, '');
    const signedFilename = `${baseNoExt}_vc.json`;
    const signedPath = path.join(opts.signedDir, signedFilename);
    const info = signVc(unsignedPath, signedPath, opts);
    summary.push({ basename, signedPath: info.signedPath, created: info.created });
  });

  console.log('Signed VC artifacts:');
  summary.forEach((entry) => {
    const rel = path.relative(ROOT, entry.signedPath);
    console.log(`- ${entry.basename} -> ${rel} (new file: ${entry.created})`);
  });
  console.log('Done.');
}

main();

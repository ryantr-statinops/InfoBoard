import { readFile, mkdir, rm, stat } from 'node:fs/promises';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { resolve } from 'node:path';

const execFileAsync = promisify(execFile);

const manifestRaw = await readFile('extension/manifest.json', 'utf8');
const manifest = JSON.parse(manifestRaw);
const version = manifest.version || '0.1.0';

const releaseDir = resolve('release');
const zipName = `infoboard-v${version}.zip`;
const zipPath = resolve(releaseDir, zipName);

await mkdir(releaseDir, { recursive: true });
await rm(zipPath, { force: true });

// Files to include in the release zip relative to extension/
const itemsToInclude = ['manifest.json', 'dist', 'icons'];

// Check if _locales exists
try {
  await stat('extension/_locales');
  itemsToInclude.push('_locales');
} catch {
  // Not yet added or optional
}

console.log(`Packaging InfoBoard v${version}...`);

await execFileAsync('zip', ['-r', zipPath, ...itemsToInclude], {
  cwd: resolve('extension')
});

const zipStat = await stat(zipPath);
const sizeKb = (zipStat.size / 1024).toFixed(2);
console.log(`✓ Created release package: release/${zipName} (${sizeKb} KB)`);

import { copyFile, mkdir, access } from 'node:fs/promises';

await mkdir('extension/dist/search', { recursive: true });
await mkdir('extension/dist/icons', { recursive: true });

async function copyIfExists(src, dest) {
  try {
    await access(src);
    await copyFile(src, dest);
  } catch (error) {
    if (error.code !== 'ENOENT') throw error;
  }
}

await copyIfExists('extension/src/search/surface-shell.html', 'extension/dist/search/surface-shell.html');
await copyIfExists('extension/src/search/surface-shell.css', 'extension/dist/search/surface-shell.css');

for (const size of [16, 32, 48, 128]) {
  await copyIfExists(`extension/icons/icon-${size}.png`, `extension/dist/icons/icon-${size}.png`);
}
await copyIfExists('extension/icons/icon.svg', 'extension/dist/icons/icon.svg');

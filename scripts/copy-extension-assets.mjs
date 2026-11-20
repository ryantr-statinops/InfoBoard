import { copyFile, mkdir, access } from 'node:fs/promises';

await mkdir('extension/dist/search', { recursive: true });

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

import { copyFile, mkdir, access } from 'node:fs/promises';
await mkdir('extension/dist/search', { recursive: true });
try {
  await access('extension/src/search/surface-shell.html');
  await copyFile('extension/src/search/surface-shell.html', 'extension/dist/search/surface-shell.html');
} catch (error) {
  if (error.code !== 'ENOENT') throw error;
}

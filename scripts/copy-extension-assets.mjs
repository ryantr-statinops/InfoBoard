import { copyFile, mkdir, access } from 'node:fs/promises';
await mkdir('extension/dist/search', { recursive: true });
await mkdir('extension/dist/settings', { recursive: true });
for (const [source, target] of [
  ['extension/src/search/surface-shell.html', 'extension/dist/search/surface-shell.html'],
  ['extension/src/settings/settings-shell.html', 'extension/dist/settings/settings-shell.html'],
]) {
  try {
    await access(source);
    await copyFile(source, target);
  } catch (error) {
    if (error.code !== 'ENOENT') throw error;
  }
}

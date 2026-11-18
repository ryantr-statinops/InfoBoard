import test from 'node:test';
import assert from 'node:assert/strict';
import { scoreTab, searchTabs } from '../extension/dist/src/search/scorer.js';

const mockTabs = [
  { id: 1, windowId: 1, title: 'TikTok - Make Your Day', url: 'https://www.tiktok.com', domain: 'tiktok.com', mruRank: 0, active: false, pinned: false },
  { id: 2, windowId: 1, title: 'Instagram - Threads', url: 'https://threads.net', domain: 'threads.net', mruRank: 1, active: false, pinned: false },
  { id: 3, windowId: 1, title: 'Tiki - Mua hàng online', url: 'https://tiki.vn', domain: 'tiki.vn', mruRank: 2, active: false, pinned: false },
  { id: 4, windowId: 1, title: 'GitHub: Where the world builds software', url: 'https://github.com', domain: 'github.com', mruRank: 3, active: false, pinned: false },
  { id: 5, windowId: 1, title: 'YouTube', url: 'https://youtube.com', domain: 'youtube.com', mruRank: 4, active: false, pinned: false }
];

test('empty query returns tabs ordered strictly by MRU', () => {
  const results = searchTabs(mockTabs, '');
  assert.equal(results.length, 5);
  assert.equal(results[0].domain, 'tiktok.com');
  assert.equal(results[1].domain, 'threads.net');
  assert.equal(results[2].domain, 'tiki.vn');
});

test('prefix "ti" matches tiki and tiktok with high score', () => {
  const results = searchTabs(mockTabs, 'ti');
  assert.ok(results.length >= 2);
  const domains = results.slice(0, 2).map(r => r.domain);
  assert.ok(domains.includes('tiktok.com'));
  assert.ok(domains.includes('tiki.vn'));
});

test('query "t" ranks tiktok before threads due to mru recency boost', () => {
  const results = searchTabs(mockTabs, 't');
  assert.ok(results.length >= 3);
  // TikTok has mruRank 0, Threads has mruRank 1
  assert.equal(results[0].domain, 'tiktok.com');
});

test('acronym "gh" matches GitHub', () => {
  const results = searchTabs(mockTabs, 'gh');
  assert.ok(results.length > 0);
  assert.equal(results[0].domain, 'github.com');
});

test('acronym "yt" matches YouTube', () => {
  const results = searchTabs(mockTabs, 'yt');
  assert.ok(results.length > 0);
  assert.equal(results[0].domain, 'youtube.com');
});

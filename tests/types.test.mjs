import test from 'node:test';
import assert from 'node:assert/strict';
import { extractDomain, normalizeTab } from '../extension/dist/src/search/types.js';

test('extractDomain: strips www prefix and parses valid URLs', () => {
  assert.equal(extractDomain('https://www.google.com/search?q=test'), 'google.com');
  assert.equal(extractDomain('https://WWW.TikTok.COM/explore'), 'tiktok.com');
  assert.equal(extractDomain('http://sub.domain.tiki.vn:3000/cart'), 'sub.domain.tiki.vn');
  assert.equal(extractDomain('https://github.com'), 'github.com');
});

test('extractDomain: handles invalid or scheme-less URLs gracefully', () => {
  assert.equal(extractDomain(''), '');
  assert.equal(extractDomain('not-a-valid-url'), '');
  assert.equal(extractDomain('javascript:alert(1)'), '');
});

test('normalizeTab: correctly normalizes standard open tab', () => {
  const mruMap = new Map([[42, 0]]);
  const tab = {
    id: 42,
    windowId: 1,
    title: 'Google Search',
    url: 'https://www.google.com',
    favIconUrl: 'https://www.google.com/favicon.ico',
    active: true,
    pinned: true,
    groupId: 5,
    audible: true,
    mutedInfo: { muted: false },
    highlighted: true,
    incognito: false,
    index: 0
  };

  const item = normalizeTab(tab, mruMap);
  assert.ok(item !== null);
  assert.equal(item.id, 42);
  assert.equal(item.windowId, 1);
  assert.equal(item.title, 'Google Search');
  assert.equal(item.domain, 'google.com');
  assert.equal(item.mruRank, 0);
  assert.equal(item.active, true);
  assert.equal(item.pinned, true);
  assert.equal(item.groupId, 5);
  assert.equal(item.audible, true);
  assert.equal(item.muted, false);
});

test('normalizeTab: falls back title to domain when title is missing or whitespace', () => {
  const mruMap = new Map();
  const tab = {
    id: 10,
    windowId: 1,
    title: '   ',
    url: 'https://threads.net/trending',
    active: false,
    pinned: false,
    highlighted: false,
    incognito: false,
    index: 1
  };

  const item = normalizeTab(tab, mruMap);
  assert.ok(item !== null);
  assert.equal(item.title, 'threads.net');
  assert.equal(item.domain, 'threads.net');
  assert.equal(item.mruRank, 9999); // Fallback when not in map
});

test('normalizeTab: rejects chrome-extension:// internal URLs', () => {
  const mruMap = new Map([[1, 0]]);
  const tab = {
    id: 1,
    windowId: 1,
    title: 'Extension Options',
    url: 'chrome-extension://abcdefg/options.html',
    active: false,
    pinned: false,
    highlighted: false,
    incognito: false,
    index: 0
  };

  const item = normalizeTab(tab, mruMap);
  assert.equal(item, null);
});

test('normalizeTab: rejects tabs with missing id or missing url', () => {
  const mruMap = new Map();
  // @ts-ignore
  assert.equal(normalizeTab({ id: undefined, url: 'https://google.com' }, mruMap), null);
  // @ts-ignore
  assert.equal(normalizeTab({ id: 10, url: undefined }, mruMap), null);
  // @ts-ignore
  assert.equal(normalizeTab({ id: 10, url: '' }, mruMap), null);
});

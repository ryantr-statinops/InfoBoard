import test from 'node:test';
import assert from 'node:assert/strict';
import { scoreTab, searchTabs, matchSubsequence, damerauLevenshtein } from '../extension/dist/src/search/scorer.js';

const mockTabs = [
  { id: 1, windowId: 1, title: 'TikTok - Make Your Day', url: 'https://www.tiktok.com', domain: 'tiktok.com', mruRank: 0, active: false, pinned: false },
  { id: 2, windowId: 1, title: 'Instagram - Threads', url: 'https://threads.net', domain: 'threads.net', mruRank: 1, active: false, pinned: false },
  { id: 3, windowId: 1, title: 'Tiki - Mua hàng online', url: 'https://tiki.vn', domain: 'tiki.vn', mruRank: 2, active: false, pinned: false },
  { id: 4, windowId: 1, title: 'GitHub: Where the world builds software', url: 'https://github.com', domain: 'github.com', mruRank: 3, active: false, pinned: false },
  { id: 5, windowId: 1, title: 'YouTube', url: 'https://youtube.com', domain: 'youtube.com', mruRank: 4, active: false, pinned: false },
  { id: 6, windowId: 1, title: '[Bug] Fix navigation (PR #42)', url: 'https://gitlab.com/issue', domain: 'gitlab.com', mruRank: 5, active: false, pinned: false }
];

test('empty query returns tabs ordered strictly by MRU', () => {
  const results = searchTabs(mockTabs, '');
  assert.equal(results.length, 6);
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

test('regex safety: query with special regex characters does not throw', () => {
  assert.doesNotThrow(() => searchTabs(mockTabs, '['));
  assert.doesNotThrow(() => searchTabs(mockTabs, '('));
  assert.doesNotThrow(() => searchTabs(mockTabs, '*'));
  assert.doesNotThrow(() => searchTabs(mockTabs, '+'));
  assert.doesNotThrow(() => searchTabs(mockTabs, '?'));
  assert.doesNotThrow(() => searchTabs(mockTabs, '\\'));
  assert.doesNotThrow(() => searchTabs(mockTabs, '.*'));

  const bracketResults = searchTabs(mockTabs, '[');
  assert.ok(bracketResults.some(r => r.id === 6));
});

test('case insensitivity: uppercase, lowercase, and mixed queries score equally', () => {
  const scoreLower = scoreTab(mockTabs[0], 'tiktok');
  const scoreUpper = scoreTab(mockTabs[0], 'TIKTOK');
  const scoreMixed = scoreTab(mockTabs[0], 'TiKToK');

  assert.equal(scoreLower, scoreUpper);
  assert.equal(scoreLower, scoreMixed);
});

test('whitespace trimming: surrounding whitespace does not alter query matches', () => {
  const normal = searchTabs(mockTabs, 'tiki');
  const padded = searchTabs(mockTabs, '   tiki   ');

  assert.equal(normal.length, padded.length);
  assert.equal(normal[0].domain, padded[0].domain);
});

test('no match query returns empty array', () => {
  const results = searchTabs(mockTabs, 'xyznotfound123');
  assert.equal(results.length, 0);
});

test('exact prefix score (1000) dominates over word-boundary match (800) regardless of recency', () => {
  // Tab A has exact prefix on domain ("tik") with older mruRank (2)
  // Tab B has word boundary match ("tik") with newest mruRank (0)
  const tabA = { id: 1, title: 'Tiki', domain: 'tiki.vn', mruRank: 2, url: 'https://tiki.vn' };
  const tabB = { id: 2, title: 'Shop - tiki seller', domain: 'seller.com', mruRank: 0, url: 'https://seller.com' };

  // @ts-ignore
  const scoreA = scoreTab(tabA, 'tiki');
  // @ts-ignore
  const scoreB = scoreTab(tabB, 'tiki');

  assert.ok(scoreA > scoreB, `Exact domain prefix score (${scoreA}) should beat word-boundary score (${scoreB})`);
});

test('subsequence matching: matches abbreviations across words and CamelCase (gdoc, ytm, amz)', () => {
  const customTabs = [
    { id: 101, title: 'Google Docs - Untitled document', domain: 'docs.google.com', mruRank: 5, url: 'https://docs.google.com' },
    { id: 102, title: 'YouTube Music - Web Player', domain: 'music.youtube.com', mruRank: 5, url: 'https://music.youtube.com' },
    { id: 103, title: 'Amazon.com: Online Shopping', domain: 'amazon.com', mruRank: 5, url: 'https://amazon.com' }
  ];

  const gdocRes = searchTabs(customTabs, 'gdoc');
  assert.ok(gdocRes.length > 0);
  assert.equal(gdocRes[0].id, 101);

  const ytmRes = searchTabs(customTabs, 'ytm');
  assert.ok(ytmRes.length > 0);
  assert.equal(ytmRes[0].id, 102);

  const amzRes = searchTabs(customTabs, 'amz');
  assert.ok(amzRes.length > 0);
  assert.equal(amzRes[0].id, 103);
});

test('subsequence scoring bonus: consecutive and compact matches score higher than scattered matches', () => {
  const compactScore = matchSubsequence('Google Docs', 'gdoc');
  const scatteredScore = matchSubsequence('Getting Documents Organized Cleanly', 'gdoc');
  assert.ok(compactScore > 0);
  assert.ok(scatteredScore > 0);
  assert.ok(compactScore > scatteredScore, `Compact match (${compactScore}) should outscore scattered match (${scatteredScore})`);
});

test('damerauLevenshtein: correctly identifies edit distance <= 1', () => {
  // Transposition
  assert.equal(damerauLevenshtein('githbu', 'github', 1), 1);
  // Substitution
  assert.equal(damerauLevenshtein('gathub', 'github', 1), 1);
  // Insertion
  assert.equal(damerauLevenshtein('githb', 'github', 1), 1);
  // Deletion
  assert.equal(damerauLevenshtein('githubb', 'github', 1), 1);
  // Identical
  assert.equal(damerauLevenshtein('github', 'github', 1), 0);
  // Distance > 1 terminates
  assert.ok(damerauLevenshtein('gihtubxx', 'github', 1) > 1);
});

test('two-phase typo fallback: triggers for transposed and misspelled queries when Phase 1 returns 0', () => {
  // "githbu" (typo for GitHub)
  const ghRes = searchTabs(mockTabs, 'githbu');
  assert.ok(ghRes.length > 0);
  assert.equal(ghRes[0].domain, 'github.com');

  // "youtbe" (typo for YouTube)
  const ytRes = searchTabs(mockTabs, 'youtbe');
  assert.ok(ytRes.length > 0);
  assert.equal(ytRes[0].domain, 'youtube.com');

  // Short query typo (< 4 chars) does not trigger fallback to prevent false positives
  const shortTypo = searchTabs(mockTabs, 'xyt');
  assert.equal(shortTypo.length, 0);
});

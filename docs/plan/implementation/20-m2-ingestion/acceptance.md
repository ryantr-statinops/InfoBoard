# M2 acceptance checklist

- [ ] Text, TXT, Markdown, PDF text, and public URL imports create the expected snapshot.
- [ ] Normalize/hash/chunk/version behavior is consistent; duplicates do not create a new item.
- [ ] File, HTML, and text limits are rejected clearly.
- [ ] SSRF and redirects into private networks are blocked.
- [ ] Worker retry/requeue/restart does not duplicate chunks or jobs.

**Evidence:** fixtures, local server transcript, recovery test, and job-state sample.

# M2 acceptance checklist

- [ ] Text, TXT, Markdown, PDF text và public URL tạo snapshot đúng.
- [ ] Normalize/hash/chunk/version nhất quán; duplicate không tạo item mới.
- [ ] File, HTML, text quá giới hạn bị từ chối rõ ràng.
- [ ] SSRF/redirect vào mạng private bị chặn.
- [ ] Worker retry/requeue/restart không nhân đôi chunks hoặc jobs.

**Evidence:** fixture, local server transcript, recovery test và job state sample.

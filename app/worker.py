from .db import connect, rebuild_fts


def process_pending(limit: int = 20) -> int:
    """Process durable indexing jobs synchronously; safe to call after restart."""
    processed = 0
    with connect() as c:
        jobs = c.execute("SELECT id,item_id FROM index_jobs WHERE state IN ('queued','failed') AND attempts < 3 ORDER BY created_at LIMIT ?", (limit,)).fetchall()
        for job in jobs:
            c.execute("UPDATE index_jobs SET state='extracting',attempts=attempts+1,updated_at=CURRENT_TIMESTAMP WHERE id=?", (job['id'],))
            c.execute("UPDATE index_jobs SET state='chunking',updated_at=CURRENT_TIMESTAMP WHERE id=?", (job['id'],))
            c.execute("UPDATE index_jobs SET state='embedding',updated_at=CURRENT_TIMESTAMP WHERE id=?", (job['id'],))
            c.execute("UPDATE index_jobs SET state='indexed',updated_at=CURRENT_TIMESTAMP,error=NULL WHERE id=?", (job['id'],))
            processed += 1
        rebuild_fts(c)
    return processed

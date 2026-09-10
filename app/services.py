import hashlib, re, unicodedata
from .db import connect, rebuild_fts

def normalize(text: str) -> str:
    return unicodedata.normalize("NFC", text).strip()

def chunks(text: str, size: int = 200, overlap: int = 30) -> list[str]:
    words = text.split(); step = max(1, size - overlap)
    return [" ".join(words[i:i+size]) for i in range(0, len(words), step) if words[i:i+size]]

def add_item(title: str, content: str, source_type="text", source_url=None) -> dict:
    content = normalize(content); digest = hashlib.sha256(content.encode()).hexdigest()
    with connect() as c:
        old = c.execute("SELECT * FROM items WHERE content_hash=? AND deleted_at IS NULL", (digest,)).fetchone()
        if old: return dict(old)
        cur = c.execute("INSERT INTO items(title,source_type,source_url,content_hash) VALUES(?,?,?,?)", (title,source_type,source_url,digest)); item_id = cur.lastrowid
        c.execute("INSERT INTO item_contents(item_id,content) VALUES(?,?)", (item_id,content))
        c.executemany("INSERT INTO chunks(item_id,position,content) VALUES(?,?,?)", [(item_id,i,x) for i,x in enumerate(chunks(content))])
        c.execute("INSERT INTO index_jobs(item_id,state) VALUES(?, 'indexed')", (item_id,)); rebuild_fts(c)
        return dict(c.execute("SELECT * FROM items WHERE id=?", (item_id,)).fetchone())

def search(query: str, limit=20) -> list[dict]:
    with connect() as c:
        c.execute("INSERT INTO search_history(query) VALUES(?)", (query,))
        rows = c.execute("SELECT i.*, snippet(items_fts,1,'<mark>','</mark>','…',24) excerpt FROM items_fts JOIN items i ON i.id=items_fts.rowid WHERE items_fts MATCH ? AND i.deleted_at IS NULL LIMIT ?", (re.sub(r'[^\w ]',' ',query),limit)).fetchall()
        return [dict(r) for r in rows]

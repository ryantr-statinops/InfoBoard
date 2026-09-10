import hashlib
import re
import unicodedata
from html.parser import HTMLParser
from pathlib import Path
from urllib.parse import urlparse, urlunparse
from urllib.request import Request, urlopen

from .db import connect, rebuild_fts


def normalize(text: str) -> str:
    text = unicodedata.normalize("NFC", text).strip()
    if len(text) > 1_000_000: raise ValueError("text exceeds 1,000,000 characters")
    return text

def chunks(text: str, size: int = 200, overlap: int = 30) -> list[str]:
    words = text.split(); step = max(1, size - overlap)
    return [" ".join(words[i:i+size]) for i in range(0, len(words), step) if words[i:i+size]]

def add_item(title: str, content: str, source_type="text", source_url=None) -> dict:
    title = normalize(title)
    if not title or not content.strip(): raise ValueError("title and content are required")
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

def add_note(item_id: int, body: str) -> dict:
    with connect() as c:
        if not c.execute("SELECT id FROM items WHERE id=? AND deleted_at IS NULL", (item_id,)).fetchone(): raise KeyError("item")
        if not body.strip(): raise ValueError("note body is required")
        cur=c.execute("INSERT INTO notes(item_id,body) VALUES(?,?)", (item_id, normalize(body)))
        return dict(c.execute("SELECT * FROM notes WHERE id=?", (cur.lastrowid,)).fetchone())

def add_collection(name: str) -> dict:
    with connect() as c:
        c.execute("INSERT OR IGNORE INTO collections(name) VALUES(?)", (normalize(name),))
        return dict(c.execute("SELECT * FROM collections WHERE name=?", (normalize(name),)).fetchone())

class _Text(HTMLParser):
    def __init__(self): super().__init__(); self.parts=[]
    def handle_data(self, data): self.parts.append(data)

def import_file(path: str) -> dict:
    p=Path(path); raw=p.read_bytes()
    if len(raw)>20*1024*1024: raise ValueError("file exceeds 20 MB limit")
    if p.suffix.lower()=='.pdf':
        try:
            from pypdf import PdfReader
            content='\n'.join(page.extract_text() or '' for page in PdfReader(str(p)).pages)
        except ImportError: content=raw.decode('utf-8','ignore')
    else: content=raw.decode('utf-8','replace')
    return add_item(p.stem, content, p.suffix.lower().lstrip('.') or 'text')

def import_url(url: str) -> dict:
    u=urlparse(url)
    import ipaddress
    import socket
    if u.scheme not in ('http','https') or not u.hostname: raise ValueError('URL is not allowed')
    try: addresses={x[4][0] for x in socket.getaddrinfo(u.hostname, None)}
    except socket.gaierror: raise ValueError('URL host cannot be resolved')
    if any(ipaddress.ip_address(a).is_private or ipaddress.ip_address(a).is_loopback or ipaddress.ip_address(a).is_link_local or ipaddress.ip_address(a).is_reserved for a in addresses): raise ValueError('URL is not allowed')
    canonical=urlunparse((u.scheme,u.netloc,u.path or '/', '',u.query,''))
    data=urlopen(Request(canonical, headers={'User-Agent':'InfoBoard/0.1'}), timeout=10).read(5*1024*1024+1)
    if len(data)>5*1024*1024: raise ValueError('HTML exceeds 5 MB limit')
    parser=_Text(); parser.feed(data.decode('utf-8','replace'))
    return add_item(u.hostname or url, ' '.join(parser.parts), 'url', canonical)

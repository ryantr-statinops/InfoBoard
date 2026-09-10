from pathlib import Path
import sqlite3

ROOT = Path(__file__).resolve().parents[1]
DB_PATH = Path(__import__("os").environ.get("INFOBOARD_DB", ROOT / "data" / "infoboard.db"))

def connect() -> sqlite3.Connection:
    DB_PATH.parent.mkdir(parents=True, exist_ok=True)
    conn = sqlite3.connect(DB_PATH)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA foreign_keys=ON")
    return conn

def init_db() -> None:
    with connect() as c:
        c.executescript("""
        CREATE TABLE IF NOT EXISTS items (id INTEGER PRIMARY KEY, title TEXT NOT NULL, source_type TEXT NOT NULL, source_url TEXT, content_hash TEXT NOT NULL UNIQUE, status TEXT NOT NULL DEFAULT 'active', created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, deleted_at TEXT);
        CREATE TABLE IF NOT EXISTS item_contents (item_id INTEGER PRIMARY KEY REFERENCES items(id) ON DELETE CASCADE, content TEXT NOT NULL);
        CREATE TABLE IF NOT EXISTS chunks (id INTEGER PRIMARY KEY, item_id INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE, position INTEGER NOT NULL, content TEXT NOT NULL);
        CREATE TABLE IF NOT EXISTS collections (id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
        CREATE TABLE IF NOT EXISTS item_collections (item_id INTEGER REFERENCES items(id) ON DELETE CASCADE, collection_id INTEGER REFERENCES collections(id) ON DELETE CASCADE, PRIMARY KEY(item_id, collection_id));
        CREATE TABLE IF NOT EXISTS notes (id INTEGER PRIMARY KEY, item_id INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE, body TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
        CREATE TABLE IF NOT EXISTS search_history (id INTEGER PRIMARY KEY, query TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
        CREATE TABLE IF NOT EXISTS index_jobs (id INTEGER PRIMARY KEY, item_id INTEGER REFERENCES items(id) ON DELETE CASCADE, state TEXT NOT NULL, error TEXT, attempts INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
        CREATE VIRTUAL TABLE IF NOT EXISTS items_fts USING fts5(title, content, content='');
        """)
        c.execute("UPDATE index_jobs SET state='queued',updated_at=CURRENT_TIMESTAMP WHERE state IN ('extracting','chunking','embedding')")

def rebuild_fts(c: sqlite3.Connection) -> None:
    c.execute("DELETE FROM items_fts")
    c.execute("INSERT INTO items_fts(rowid,title,content) SELECT i.id,i.title,co.content FROM items i JOIN item_contents co ON co.item_id=i.id WHERE i.deleted_at IS NULL")

from .db import DB_PATH


def dashboard_metrics() -> dict:
    """Use DuckDB when installed, otherwise return the SQLite-compatible metrics."""
    try:
        import duckdb
        con = duckdb.connect()
        path = str(DB_PATH).replace("'", "''")
        rows = con.execute(f"SELECT count(*) AS items FROM sqlite_scan('{path}', 'items') WHERE deleted_at IS NULL").fetchone()
        chunks = con.execute(f"SELECT count(*) FROM sqlite_scan('{path}', 'chunks')").fetchone()[0]
        con.close(); return {"items": rows[0], "chunks": chunks, "engine": "duckdb"}
    except ImportError:
        return {"engine": "sqlite-fallback"}

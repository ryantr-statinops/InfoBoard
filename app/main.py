from contextlib import asynccontextmanager
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from .db import init_db, connect
from .services import add_item, search

@asynccontextmanager
async def lifespan(app): init_db(); yield
app = FastAPI(title="InfoBoard", lifespan=lifespan)

class ItemIn(BaseModel): title: str; content: str; source_type: str = "text"; source_url: str | None = None
class CollectionIn(BaseModel): name: str

@app.get("/api/health")
def health(): return {"status":"ok"}
@app.post("/api/items")
def create_item(body: ItemIn): return add_item(**body.model_dump())
@app.get("/api/items")
def list_items(limit: int=50):
    with connect() as c: return [dict(r) for r in c.execute("SELECT * FROM items WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT ?",(limit,))]
@app.get("/api/items/{item_id}")
def get_item(item_id:int):
    with connect() as c:
        r=c.execute("SELECT i.*,co.content FROM items i JOIN item_contents co ON co.item_id=i.id WHERE i.id=? AND i.deleted_at IS NULL",(item_id,)).fetchone()
        if not r: raise HTTPException(404,"Item not found")
        return dict(r)
@app.post("/api/search")
def do_search(body: dict): return {"items": search(str(body.get("query","")))}
@app.get("/api/analytics")
def analytics():
    with connect() as c: return {"items":c.execute("SELECT count(*) n FROM items WHERE deleted_at IS NULL").fetchone()["n"],"chunks":c.execute("SELECT count(*) n FROM chunks").fetchone()["n"]}

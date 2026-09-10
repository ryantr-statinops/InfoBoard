from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException, Request, UploadFile
from fastapi.templating import Jinja2Templates
from pydantic import BaseModel

from .db import ROOT, connect, init_db
from .services import add_collection, add_item, add_note, import_file, import_url, search


@asynccontextmanager
async def lifespan(app): init_db(); yield
app = FastAPI(title="InfoBoard", lifespan=lifespan)
templates = Jinja2Templates(directory=str(ROOT / "app" / "templates"))

class ItemIn(BaseModel): title: str; content: str; source_type: str = "text"; source_url: str | None = None
class CollectionIn(BaseModel): name: str

@app.get("/api/health")
def health(): return {"status":"ok"}
@app.get("/")
def dashboard(request: Request, q: str = ""):
    with connect() as c:
        items = [dict(r) for r in c.execute("SELECT * FROM items WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT 30")]
        count=c.execute("SELECT count(*) n FROM items WHERE deleted_at IS NULL").fetchone()["n"]
        chunks=c.execute("SELECT count(*) n FROM chunks").fetchone()["n"]
    if q: items = search(q)
    return templates.TemplateResponse(request=request, name="dashboard.html", context={"items":items,"count":count,"chunks":chunks,"q":q})
@app.post("/api/items")
def create_item(body: ItemIn):
    try: return add_item(**body.model_dump())
    except ValueError as e: raise HTTPException(400, str(e))
@app.get("/api/items")
def list_items(limit: int=50, offset: int=0, collection_id: int|None=None, status: str|None=None):
    with connect() as c:
        sql="SELECT DISTINCT i.* FROM items i LEFT JOIN item_collections ic ON ic.item_id=i.id WHERE i.deleted_at IS NULL"; args=[]
        if collection_id is not None: sql += " AND ic.collection_id=?"; args.append(collection_id)
        if status: sql += " AND i.status=?"; args.append(status)
        sql += " ORDER BY i.created_at DESC LIMIT ? OFFSET ?"; args.extend([min(limit,100),offset])
        return [dict(r) for r in c.execute(sql,args)]
@app.get("/api/items/{item_id}")
def get_item(item_id:int):
    with connect() as c:
        r=c.execute("SELECT i.*,co.content FROM items i JOIN item_contents co ON co.item_id=i.id WHERE i.id=? AND i.deleted_at IS NULL",(item_id,)).fetchone()
        if not r: raise HTTPException(404,"Item not found")
        item=dict(r); item["collections"]=[dict(x) for x in c.execute("SELECT col.* FROM collections col JOIN item_collections ic ON ic.collection_id=col.id WHERE ic.item_id=?",(item_id,))]; item["notes"]=[dict(x) for x in c.execute("SELECT * FROM notes WHERE item_id=? ORDER BY created_at DESC",(item_id,))]; return item
@app.post("/api/search")
def do_search(body: dict): return {"items": search(str(body.get("query","")))}
@app.post("/api/items/upload")
async def upload(file: UploadFile):
    import tempfile
    suffix='.' + (file.filename or 'txt').split('.')[-1]
    with tempfile.NamedTemporaryFile(suffix=suffix, delete=False) as f: f.write(await file.read()); path=f.name
    try: return import_file(path)
    finally:
        import os
        os.unlink(path)
@app.post("/api/items/url")
def url_item(body: dict): return import_url(str(body['url']))
@app.patch("/api/items/{item_id}")
def update_item(item_id:int, body:dict):
    with connect() as c:
        if not c.execute("SELECT id FROM items WHERE id=?",(item_id,)).fetchone(): raise HTTPException(404,"Item not found")
        if 'title' in body: c.execute("UPDATE items SET title=?,updated_at=CURRENT_TIMESTAMP WHERE id=?",(body['title'],item_id))
        return dict(c.execute("SELECT * FROM items WHERE id=?",(item_id,)).fetchone())
@app.delete("/api/items/{item_id}")
def delete_item(item_id:int):
    with connect() as c: c.execute("UPDATE items SET deleted_at=CURRENT_TIMESTAMP,status='deleted' WHERE id=?",(item_id,))
    return {"deleted":True}
@app.get("/api/collections")
def collections():
    with connect() as c: return [dict(r) for r in c.execute("SELECT * FROM collections ORDER BY name")]
@app.post("/api/collections")
def collection(body:CollectionIn): return add_collection(body.name)
@app.post("/api/items/{item_id}/collections/{collection_id}")
def attach_collection(item_id:int, collection_id:int):
    with connect() as c:
        c.execute("INSERT OR IGNORE INTO item_collections(item_id,collection_id) VALUES(?,?)",(item_id,collection_id))
    return {"ok":True}
@app.post("/api/items/{item_id}/notes")
def note(item_id:int, body:dict): return add_note(item_id, str(body.get('body','')))
@app.get("/api/items/{item_id}/related")
def related(item_id:int):
    with connect() as c:
        r=c.execute("SELECT content FROM item_contents WHERE item_id=?",(item_id,)).fetchone()
        if not r:return []
        terms=' '.join(r['content'].split()[:5])
    return [x for x in search(terms,10) if x['id']!=item_id][:5]
@app.get("/api/analytics")
def analytics():
    with connect() as c: return {"items":c.execute("SELECT count(*) n FROM items WHERE deleted_at IS NULL").fetchone()["n"],"chunks":c.execute("SELECT count(*) n FROM chunks").fetchone()["n"]}

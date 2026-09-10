from app.db import init_db
from app.services import add_collection, add_item, add_note, search


def test_item_flow(tmp_path, monkeypatch):
    from app import db
    monkeypatch.setattr(db, 'DB_PATH', tmp_path / 'flow.db')
    init_db()
    item = add_item('Python', 'Python là ngôn ngữ lập trình')
    assert add_item('Duplicate', 'Python là ngôn ngữ lập trình')['id'] == item['id']
    add_collection('Dev')
    add_note(item['id'], 'Đọc lại')
    assert search('Python')[0]['id'] == item['id']

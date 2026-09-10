from app.services import reciprocal_rank_fusion


def test_rrf_prefers_items_in_both_lists():
    result = reciprocal_rank_fusion([{'id': 1}, {'id': 2}], [{'id': 2}, {'id': 3}])
    assert result[0]['id'] == 2

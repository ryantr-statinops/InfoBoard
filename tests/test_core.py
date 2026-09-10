from app.services import chunks, normalize


def test_normalize_and_chunks():
    assert normalize("  tiếng Việt ") == "tiếng Việt"
    assert len(chunks("a " * 250)) == 2

def test_embedding_fallback():
    from app.semantic import EmbeddingProvider
    assert EmbeddingProvider().tokenize("xin chào") == ["xin", "chào"]

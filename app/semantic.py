"""Optional semantic search adapter; keyword search remains the safe fallback."""
class EmbeddingProvider:
    model_id = "sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2"
    revision = "default"
    dimension = 384
    max_tokens = 256
    def embed(self, texts: list[str]) -> list[list[float]]:
        try:
            from sentence_transformers import SentenceTransformer
        except ImportError:
            return []
        model = SentenceTransformer(self.model_id)
        return model.encode(texts, normalize_embeddings=True).tolist()
    def tokenize(self, text: str) -> list[str]: return text.split()

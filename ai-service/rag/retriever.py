"""RAG 检索器

TopK 召回：5
重排 Top：3
"""
from dataclasses import dataclass
from typing import Optional

from config.settings import settings
from rag.embedder import Embedder, TextChunk, chunk_text
from rag.vector_store import VectorStore, VectorDocument, cosine_similarity


@dataclass
class RetrievalResult:
    """检索结果"""
    content: str
    score: float
    source: str  # 来源标识


class Retriever:
    """RAG 检索器，支持向量召回 + 重排

    流程：query → embed → top_k 召回 → rerank → top_n 返回
    """

    def __init__(
        self,
        embedder: Optional[Embedder] = None,
        vector_store: Optional[VectorStore] = None,
    ) -> None:
        self._embedder = embedder or Embedder()
        self._store = vector_store or VectorStore()
        self._top_k = settings.rag.top_k
        self._rerank_top = settings.rag.rerank_top

    def index(self, text: str, source: str = "") -> int:
        """将文本分块并索引到向量存储

        Args:
            text: 原始文本
            source: 来源标识

        Returns:
            索引的分块数量
        """
        chunks = chunk_text(text)
        results = self._embedder.embed(chunks)

        docs = []
        for result in results:
            doc = VectorDocument(
                id=f"{source}_{result.chunk.index}",
                content=result.chunk.content,
                vector=result.vector,
                metadata={
                    "source": source,
                    "index": result.chunk.index,
                    "start_offset": result.chunk.start_offset,
                    "end_offset": result.chunk.end_offset,
                },
            )
            docs.append(doc)

        self._store.add_batch(docs)
        return len(docs)

    def retrieve(self, query: str, knowledge_base: str = "") -> list[RetrievalResult]:
        """执行 RAG 检索

        Args:
            query: 用户查询
            knowledge_base: 知识库标识（保留扩展）

        Returns:
            重排后的检索结果列表
        """
        # 1. 查询向量化
        query_vector = self._embedder.embed_query(query)

        # 2. TopK 召回
        candidates = self._vector_search(query_vector, self._top_k)

        # 3. 重排
        reranked = self._rerank(query, candidates, self._rerank_top)

        return reranked

    def _vector_search(
        self, query_vector: list[float], top_k: int
    ) -> list[RetrievalResult]:
        """向量相似度搜索"""
        results = self._store.search(query_vector, top_k)

        return [
            RetrievalResult(
                content=doc.content,
                score=score,
                source=doc.metadata.get("source", ""),
            )
            for doc, score in results
        ]

    def _rerank(
        self, query: str, candidates: list[RetrievalResult], top_n: int
    ) -> list[RetrievalResult]:
        """重排候选结果

        当前使用原始相似度分数排序。
        生产环境应集成 cross-encoder 模型进行重排。
        """
        # 按 score 降序排列
        candidates.sort(key=lambda x: x.score, reverse=True)
        return candidates[:top_n]

    def clear(self) -> None:
        """清空索引"""
        self._store.clear()

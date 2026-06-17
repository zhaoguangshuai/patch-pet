"""向量存储模块

内存向量存储，支持余弦相似度搜索。
生产环境应替换为 Milvus/PgVector/Qdrant。
"""
import math
from dataclasses import dataclass, field

from rag.embedder import EmbeddingResult


@dataclass
class VectorDocument:
    """向量文档"""
    id: str
    content: str
    vector: list[float]
    metadata: dict = field(default_factory=dict)


def cosine_similarity(a: list[float], b: list[float]) -> float:
    """计算余弦相似度

    Args:
        a: 向量 a
        b: 向量 b

    Returns:
        余弦相似度 [-1, 1]
    """
    if len(a) != len(b):
        raise ValueError(f"向量维度不匹配: {len(a)} vs {len(b)}")

    dot = sum(x * y for x, y in zip(a, b))
    norm_a = math.sqrt(sum(x * x for x in a))
    norm_b = math.sqrt(sum(x * x for x in b))

    if norm_a == 0 or norm_b == 0:
        return 0.0

    return dot / (norm_a * norm_b)


class VectorStore:
    """内存向量存储

    支持文档添加、相似度搜索、按 ID 删除。
    生产环境替换为 Milvus/PgVector/Qdrant。
    """

    def __init__(self) -> None:
        self._documents: list[VectorDocument] = []

    def add(self, doc: VectorDocument) -> None:
        """添加文档到向量存储

        Args:
            doc: 向量文档
        """
        self._documents.append(doc)

    def add_batch(self, docs: list[VectorDocument]) -> None:
        """批量添加文档

        Args:
            docs: 文档列表
        """
        self._documents.extend(docs)

    def search(self, query_vector: list[float], top_k: int = 5) -> list[tuple[VectorDocument, float]]:
        """相似度搜索

        Args:
            query_vector: 查询向量
            top_k: 返回最相似的 K 个结果

        Returns:
            (文档, 相似度) 列表，按相似度降序排列
        """
        if not self._documents:
            return []

        scored = []
        for doc in self._documents:
            sim = cosine_similarity(query_vector, doc.vector)
            scored.append((doc, sim))

        scored.sort(key=lambda x: x[1], reverse=True)
        return scored[:top_k]

    def delete(self, doc_id: str) -> bool:
        """按 ID 删除文档

        Args:
            doc_id: 文档 ID

        Returns:
            是否删除成功
        """
        for i, doc in enumerate(self._documents):
            if doc.id == doc_id:
                self._documents.pop(i)
                return True
        return False

    def clear(self) -> None:
        """清空所有文档"""
        self._documents.clear()

    @property
    def size(self) -> int:
        """文档数量"""
        return len(self._documents)

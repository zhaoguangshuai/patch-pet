"""向量嵌入模块

嵌入模型：bge-m3
分块大小：512 字符，重叠 64 字符
"""
import hashlib
import math
from dataclasses import dataclass

from config.settings import settings


@dataclass
class TextChunk:
    """文本分块"""
    content: str
    index: int
    start_offset: int
    end_offset: int


@dataclass
class EmbeddingResult:
    """嵌入结果"""
    chunk: TextChunk
    vector: list[float]


def chunk_text(text: str) -> list[TextChunk]:
    """将文本按固定大小分块

    分块参数：chunk_size=512 字符，overlap=64 字符
    知识库变更必须经 Prompt 版本管理同步

    Args:
        text: 待分块的原始文本

    Returns:
        分块列表
    """
    chunk_size = settings.rag.chunk_size
    overlap = settings.rag.chunk_overlap
    chunks: list[TextChunk] = []
    start = 0
    index = 0

    while start < len(text):
        end = min(start + chunk_size, len(text))
        chunks.append(TextChunk(
            content=text[start:end],
            index=index,
            start_offset=start,
            end_offset=end,
        ))
        start += chunk_size - overlap
        index += 1

    return chunks


def _text_to_vector(text: str, dim: int = 768) -> list[float]:
    """将文本转换为确定性向量（用于测试/降级）

    使用文本哈希生成确定性向量，保证相同文本产生相同向量。
    生产环境应替换为 bge-m3 模型推理。

    Args:
        text: 输入文本
        dim: 向量维度

    Returns:
        归一化的浮点向量
    """
    hash_bytes = hashlib.sha256(text.encode("utf-8")).digest()
    # 扩展到目标维度
    extended = hash_bytes * (dim // len(hash_bytes) + 1)
    vector = [float(b) / 255.0 for b in extended[:dim]]
    # L2 归一化
    norm = math.sqrt(sum(v * v for v in vector))
    if norm > 0:
        vector = [v / norm for v in vector]
    return vector


class Embedder:
    """向量嵌入器，使用 bge-m3 模型

    生产环境集成 sentence-transformers 加载 bge-m3。
    当前使用哈希向量作为降级方案。
    """

    def __init__(self) -> None:
        self._model_name = settings.rag.embedding_model
        self._dim = 768  # bge-m3 默认维度

    def embed(self, chunks: list[TextChunk]) -> list[EmbeddingResult]:
        """批量生成文本嵌入向量

        Args:
            chunks: 文本分块列表

        Returns:
            嵌入结果列表
        """
        results = []
        for chunk in chunks:
            vector = _text_to_vector(chunk.content, self._dim)
            results.append(EmbeddingResult(chunk=chunk, vector=vector))
        return results

    def embed_query(self, query: str) -> list[float]:
        """生成查询文本的嵌入向量

        Args:
            query: 查询文本

        Returns:
            嵌入向量
        """
        return _text_to_vector(query, self._dim)

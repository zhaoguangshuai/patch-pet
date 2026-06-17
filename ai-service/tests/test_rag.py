"""RAG 模块测试"""
import math
import sys
import os

# 添加项目根目录到 Python 路径
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from rag.embedder import chunk_text, TextChunk, Embedder, _text_to_vector
from rag.vector_store import VectorStore, VectorDocument, cosine_similarity
from rag.retriever import Retriever, RetrievalResult


def test_chunk_text_basic():
    """测试基本分块功能"""
    text = "A" * 1000
    chunks = chunk_text(text)

    assert len(chunks) > 0
    assert chunks[0].content == "A" * 512
    assert chunks[0].index == 0
    assert chunks[0].start_offset == 0
    assert chunks[0].end_offset == 512


def test_chunk_text_overlap():
    """测试分块重叠"""
    text = "A" * 1000
    chunks = chunk_text(text)

    # 第二块从 512-64=448 开始
    assert chunks[1].start_offset == 448
    assert chunks[1].end_offset == 960


def test_chunk_text_short():
    """测试短文本分块"""
    text = "Hello"
    chunks = chunk_text(text)

    assert len(chunks) == 1
    assert chunks[0].content == "Hello"
    assert chunks[0].start_offset == 0
    assert chunks[0].end_offset == 5


def test_chunk_text_empty():
    """测试空文本分块"""
    chunks = chunk_text("")
    assert len(chunks) == 0


def test_text_to_vector_deterministic():
    """测试向量生成的确定性"""
    vec1 = _text_to_vector("hello world")
    vec2 = _text_to_vector("hello world")
    assert vec1 == vec2


def test_text_to_vector_different():
    """测试不同文本生成不同向量"""
    vec1 = _text_to_vector("hello")
    vec2 = _text_to_vector("world")
    assert vec1 != vec2


def test_text_to_vector_normalized():
    """测试向量归一化"""
    vec = _text_to_vector("test text")
    norm = math.sqrt(sum(v * v for v in vec))
    assert abs(norm - 1.0) < 1e-6


def test_embedder_embed():
    """测试嵌入器批量嵌入"""
    embedder = Embedder()
    chunks = [
        TextChunk(content="hello", index=0, start_offset=0, end_offset=5),
        TextChunk(content="world", index=1, start_offset=6, end_offset=11),
    ]
    results = embedder.embed(chunks)

    assert len(results) == 2
    assert len(results[0].vector) == 768
    assert results[0].chunk.content == "hello"


def test_embedder_embed_query():
    """测试查询嵌入"""
    embedder = Embedder()
    vec = embedder.embed_query("test query")
    assert len(vec) == 768


def test_cosine_similarity_same():
    """测试相同向量余弦相似度"""
    vec = [1.0, 0.0, 0.0]
    assert abs(cosine_similarity(vec, vec) - 1.0) < 1e-6


def test_cosine_similarity_orthogonal():
    """测试正交向量余弦相似度"""
    a = [1.0, 0.0, 0.0]
    b = [0.0, 1.0, 0.0]
    assert abs(cosine_similarity(a, b)) < 1e-6


def test_cosine_similarity_opposite():
    """测试反向向量余弦相似度"""
    a = [1.0, 0.0, 0.0]
    b = [-1.0, 0.0, 0.0]
    assert abs(cosine_similarity(a, b) - (-1.0)) < 1e-6


def test_cosine_similarity_dimension_mismatch():
    """测试维度不匹配"""
    a = [1.0, 0.0]
    b = [1.0, 0.0, 0.0]
    try:
        cosine_similarity(a, b)
        assert False, "should raise ValueError"
    except ValueError:
        pass


def test_vector_store_add_and_search():
    """测试向量存储添加和搜索"""
    store = VectorStore()
    store.add(VectorDocument(id="1", content="hello", vector=[1.0, 0.0, 0.0]))
    store.add(VectorDocument(id="2", content="world", vector=[0.0, 1.0, 0.0]))

    assert store.size == 2

    results = store.search([1.0, 0.0, 0.0], top_k=1)
    assert len(results) == 1
    assert results[0][0].id == "1"
    assert results[0][1] > 0.99


def test_vector_store_delete():
    """测试删除文档"""
    store = VectorStore()
    store.add(VectorDocument(id="1", content="hello", vector=[1.0, 0.0]))

    assert store.delete("1") is True
    assert store.size == 0
    assert store.delete("nonexistent") is False


def test_vector_store_clear():
    """测试清空"""
    store = VectorStore()
    store.add(VectorDocument(id="1", content="hello", vector=[1.0, 0.0]))
    store.clear()
    assert store.size == 0


def test_retriever_index_and_retrieve():
    """测试检索器索引和检索"""
    retriever = Retriever()

    # 索引文档
    count = retriever.index("猫的正常体温是 38-39 度", source="medical_kb")
    assert count > 0

    # 检索
    results = retriever.retrieve("猫的体温是多少")
    assert len(results) > 0
    assert results[0].source == "medical_kb"


def test_retriever_top_k():
    """测试 TopK 召回"""
    retriever = Retriever()

    # 索引多个文档
    for i in range(10):
        retriever.index(f"文档内容 {i}: 这是第 {i} 段测试文本", source=f"doc_{i}")

    results = retriever.retrieve("测试文本")
    # 应该返回最多 rerank_top=3 个结果
    assert len(results) <= 3


def test_retriever_clear():
    """测试清空索引"""
    retriever = Retriever()
    retriever.index("test content", source="test")
    retriever.clear()

    results = retriever.retrieve("test")
    assert len(results) == 0


def test_retriever_empty_search():
    """测试空索引检索"""
    retriever = Retriever()
    results = retriever.retrieve("anything")
    assert len(results) == 0

"""配置管理模块

所有配置项通过 Nacos 配置中心托管，代码中仅定义默认值。
禁止在代码中硬编码密钥、IP、端口、账号。
"""
import os
from dataclasses import dataclass, field
from typing import Optional


@dataclass(frozen=True)
class LLMConfig:
    """LLM 模型配置"""
    # 降级链路：GPT-5 → Claude Sonnet → Gemini → 静态模板
    primary_model: str = "gpt-5"
    fallback_model: str = "claude-sonnet"
    final_fallback_model: str = "gemini"
    # API Key 从环境变量读取，禁止硬编码
    api_key: str = field(default_factory=lambda: os.environ.get("LLM_API_KEY", ""))
    base_url: str = field(default_factory=lambda: os.environ.get("LLM_BASE_URL", ""))
    # 全局运行阈值（固定不可改）
    max_input_tokens: int = 8000
    max_output_tokens: int = 1500
    timeout_seconds: int = 3
    max_retries: int = 1  # LLM 接口重试 1 次
    circuit_breaker_threshold: int = 10  # 连续 10 次失败熔断
    circuit_breaker_duration_seconds: int = 300  # 熔断 5 分钟


@dataclass(frozen=True)
class RAGConfig:
    """RAG 检索增强配置"""
    embedding_model: str = "bge-m3"
    chunk_size: int = 512  # 分块大小（字符）
    chunk_overlap: int = 64  # 重叠大小（字符）
    top_k: int = 5  # TopK 召回
    rerank_top: int = 3  # 重排 Top


@dataclass(frozen=True)
class KafkaConfig:
    """Kafka 配置"""
    bootstrap_servers: str = field(
        default_factory=lambda: os.environ.get("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092")
    )
    # Topic 划分
    topic_medical_event: str = "biz-medical-event"
    topic_dogwalk_event: str = "biz-dogwalk-event"
    topic_ai_offline_task: str = "ai-offline-task"
    topic_dlq: str = "biz-dlq"
    # 消费重试
    max_retries: int = 3
    # 消费者组
    consumer_group: str = "ai-service"


@dataclass(frozen=True)
class TokenBudgetConfig:
    """Token 预算配置（Nacos 托管）"""
    daily_budget: int = 100000
    monthly_budget: int = 2000000
    max_cost_per_user: int = 5000
    # 超过 80% 阈值触发 P2 告警
    alert_threshold_ratio: float = 0.8


@dataclass(frozen=True)
class GRPCConfig:
    """gRPC 服务配置"""
    host: str = field(default_factory=lambda: os.environ.get("GRPC_HOST", "0.0.0.0"))
    port: int = field(default_factory=lambda: int(os.environ.get("GRPC_PORT", "50051")))
    max_workers: int = 10


@dataclass
class AppSettings:
    """应用全局配置"""
    llm: LLMConfig = field(default_factory=LLMConfig)
    rag: RAGConfig = field(default_factory=RAGConfig)
    kafka: KafkaConfig = field(default_factory=KafkaConfig)
    token_budget: TokenBudgetConfig = field(default_factory=TokenBudgetConfig)
    grpc: GRPCConfig = field(default_factory=GRPCConfig)


# 全局配置单例
settings = AppSettings()

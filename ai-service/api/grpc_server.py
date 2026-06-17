"""gRPC 服务端

Go ↔ Python AI 服务实时通信通道
提供：意图识别、摘要生成、RAG 检索等低延迟接口
"""
import logging
from concurrent import futures
from typing import Any

import grpc

from config.settings import settings

logger = logging.getLogger(__name__)


# ASSUMPTION: protobuf 生成代码在 proto 编译后可用
# 此处定义服务接口骨架，proto 文件待补充


class AIServiceServicer:
    """AI 服务 gRPC 实现

    接口列表：
    - IntentRecognition: 意图识别
    - SummaryGeneration: 摘要生成
    - RAGRetrieve: RAG 检索
    """

    def __init__(self, llm_client: Any = None, retriever: Any = None) -> None:
        self._llm_client = llm_client
        self._retriever = retriever

    def IntentRecognition(self, request: Any, context: Any) -> Any:
        """意图识别接口

        Args:
            request: 包含 user_input, trace_id, user_id
            context: gRPC context

        Returns:
            包含 intent, action, risk_level, need_confirm, tool_calls
        """
        # ASSUMPTION: 具体的 protobuf 消息类型待 proto 文件定义后实现
        logger.info(
            "意图识别请求",
            extra={"trace_id": getattr(request, "trace_id", "")},
        )
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details("接口待实现")
        return None

    def SummaryGeneration(self, request: Any, context: Any) -> Any:
        """摘要生成接口

        Args:
            request: 包含 episode_id, data_scopes, trace_id
            context: gRPC context

        Returns:
            包含 owner_text, clinic_text, safety_status
        """
        logger.info(
            "摘要生成请求",
            extra={"trace_id": getattr(request, "trace_id", "")},
        )
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details("接口待实现")
        return None

    def RAGRetrieve(self, request: Any, context: Any) -> Any:
        """RAG 检索接口

        Args:
            request: 包含 query, knowledge_base, trace_id
            context: gRPC context

        Returns:
            包含 results（检索结果列表）
        """
        logger.info(
            "RAG 检索请求",
            extra={"trace_id": getattr(request, "trace_id", "")},
        )
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details("接口待实现")
        return None


def create_grpc_server(servicer: AIServiceServicer) -> grpc.Server:
    """创建并配置 gRPC 服务器

    Args:
        servicer: AI 服务实现

    Returns:
        配置好的 gRPC 服务器实例
    """
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=settings.grpc.max_workers)
    )
    # ASSUMPTION: 注册 protobuf 生成的 service descriptor
    # add_AIServiceServicer_to_server(servicer, server)
    server.add_insecure_port(f"{settings.grpc.host}:{settings.grpc.port}")
    return server

"""帕奇宠 AI 服务入口

独立 Python AI 服务，与 Go 主服务物理隔离。
职责：LLM 调用、RAG 向量化/检索、Prompt 管理、AI 自动化评测

通信通道：
- gRPC：实时摘要、意图识别、低延迟交互
- Kafka：异步 RAG 更新、AI 评测上报
"""
import logging
import signal
import sys
import threading

from config.settings import settings
from api.grpc_server import AIServiceServicer, create_grpc_server
from api.kafka_consumer import KafkaConsumer

# 结构化日志配置
logging.basicConfig(
    level=logging.INFO,
    format='{"time":"%(asctime)s","level":"%(levelname)s","module":"%(module)s","msg":"%(message)s"}',
    stream=sys.stdout,
)
logger = logging.getLogger(__name__)


def main() -> None:
    """启动 AI 服务"""
    logger.info("帕奇宠 AI 服务启动中...")

    # 初始化 gRPC 服务
    servicer = AIServiceServicer()
    server = create_grpc_server(servicer)

    # 初始化 Kafka 消费者（骨架实现，后台线程运行）
    consumer = KafkaConsumer()
    consumer_thread = threading.Thread(target=consumer.start, daemon=True, name="kafka-consumer")
    consumer_thread.start()
    logger.info("AI service Kafka consumer started")

    # 注册优雅停机
    def graceful_shutdown(signum: int, frame: any) -> None:
        logger.info("收到退出信号 %d，开始优雅停机...", signum)
        consumer.stop()
        server.stop(grace=30)
        logger.info("AI 服务已停止")
        sys.exit(0)

    signal.signal(signal.SIGINT, graceful_shutdown)
    signal.signal(signal.SIGTERM, graceful_shutdown)

    # 启动 gRPC 服务器
    server.start()
    logger.info(
        "gRPC 服务已启动，监听 %s:%d",
        settings.grpc.host,
        settings.grpc.port,
    )

    # 阻塞等待
    server.wait_for_termination()


if __name__ == "__main__":
    main()

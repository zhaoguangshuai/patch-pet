"""Kafka 消费者

异步通道：RAG 更新、AI 评测上报
消费失败 3 次重试后转 biz-dlq 死信队列
"""
import json
import logging
from typing import Any, Callable, Optional

from config.settings import settings

logger = logging.getLogger(__name__)


class KafkaConsumer:
    """Kafka 消费者

    Topic 划分：
    - biz-medical-event: 医疗业务事件
    - biz-dogwalk-event: 代遛业务事件
    - ai-offline-task: AI 离线任务（RAG 更新、评测）

    异常联动：消费失败 3 次 → 转 biz-dlq 死信队列 + P1 告警
    """

    def __init__(self) -> None:
        self._config = settings.kafka
        self._handlers: dict[str, Callable] = {}
        self._running = False

    def register_handler(self, topic: str, handler: Callable[[dict[str, Any]], None]) -> None:
        """注册 Topic 消息处理器

        Args:
            topic: Kafka Topic 名称
            handler: 消息处理函数，接收解析后的 dict
        """
        self._handlers[topic] = handler

    def start(self) -> None:
        """启动消费者（阻塞）

        消费失败 3 次重试后转 biz-dlq
        """
        self._running = True
        logger.info("Kafka 消费者启动，订阅 Topics: %s", list(self._handlers.keys()))

        # ASSUMPTION: 具体的 kafka-python 消费逻辑在集成时实现
        # 此处为骨架，确保接口契约正确
        while self._running:
            try:
                self._poll()
            except Exception as e:
                logger.error("Kafka 消费异常: %s", str(e))
                # 消费失败不退出循环，继续重试

    def stop(self) -> None:
        """停止消费者"""
        self._running = False
        logger.info("Kafka 消费者停止")

    def _poll(self) -> None:
        """拉取并处理消息"""
        # TODO: 集成 kafka-python 后实现具体消费逻辑
        import time
        time.sleep(1)  # 骨架：避免空转

    def _handle_message(self, topic: str, message: dict[str, Any]) -> None:
        """处理单条消息，失败重试后转死信队列

        Args:
            topic: 消息来源 Topic
            message: 解析后的消息体
        """
        handler = self._handlers.get(topic)
        if not handler:
            logger.warning("Topic %s 无注册处理器", topic)
            return

        retry_count = 0
        max_retries = self._config.max_retries

        while retry_count <= max_retries:
            try:
                handler(message)
                return
            except Exception as e:
                retry_count += 1
                logger.error(
                    "消息处理失败 (Topic: %s, 重试: %d/%d): %s",
                    topic, retry_count, max_retries, str(e),
                )

        # 重试耗尽，转死信队列
        self._send_to_dlq(topic, message)

    def _send_to_dlq(self, original_topic: str, message: dict[str, Any]) -> None:
        """发送消息到死信队列

        死信消息必须生成人工工单 + P1 告警 + 审计日志全量保留
        """
        dlq_message = {
            "original_topic": original_topic,
            "original_message": message,
            "dlq_reason": "max_retries_exceeded",
        }
        logger.error(
            "消息转入死信队列",
            extra={"original_topic": original_topic, "dlq_message": dlq_message},
        )
        # ASSUMPTION: 具体的 DLQ 发送逻辑待集成 kafka-python

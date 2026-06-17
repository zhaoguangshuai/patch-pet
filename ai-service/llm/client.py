"""LLM 客户端

降级链路：GPT-5 → Claude Sonnet → Gemini → 静态模板
每次降级强制写入日志（trace_id、原模型、目标模型、原因）
静态模板作为最终兜底，禁止再向下游降级
"""
import json
import logging
from dataclasses import dataclass, field
from typing import Any, Optional

from config.settings import settings

logger = logging.getLogger(__name__)


@dataclass
class LLMResponse:
    """LLM 响应结构"""
    content: str
    model: str  # 实际使用的模型
    input_tokens: int = 0
    output_tokens: int = 0
    degraded: bool = False  # 是否经过降级
    degrade_reason: str = ""


@dataclass
class TokenUsage:
    """Token 用量统计"""
    daily_used: int = 0
    monthly_used: int = 0
    per_user_used: dict[str, int] = field(default_factory=dict)


# 静态模板兜底（最终降级目标）
STATIC_TEMPLATE_RESPONSE = json.dumps({
    "intent": "notice",
    "action": "notice",
    "risk_level": "P2",
    "need_confirm": False,
    "tool_calls": [],
    "message": "AI 服务暂时不可用，请稍后重试",
}, ensure_ascii=False)


class LLMClient:
    """LLM 客户端，支持降级链路与 Token 预算管控

    执行链路：
    1. 尝试 primary_model
    2. 失败后尝试 fallback_model
    3. 失败后尝试 final_fallback_model
    4. 全部失败使用静态模板
    """

    def __init__(self) -> None:
        self._config = settings.llm
        self._token_usage = TokenUsage()
        self._circuit_breaker_failures: dict[str, int] = {}
        self._models = [
            self._config.primary_model,
            self._config.fallback_model,
            self._config.final_fallback_model,
        ]

    def chat(
        self,
        messages: list[dict[str, str]],
        trace_id: str = "",
        user_id: str = "",
    ) -> LLMResponse:
        """发送对话请求，自动降级

        Args:
            messages: 对话消息列表，格式 [{"role": "user", "content": "..."}]
            trace_id: 链路追踪 ID
            user_id: 用户 ID，用于 Token 预算校验

        Returns:
            LLMResponse 包含响应内容与用量信息
        """
        # Token 预算校验
        if not self._check_token_budget(user_id):
            logger.warning(
                "Token 预算耗尽，强制使用静态模板",
                extra={"trace_id": trace_id, "user_id": user_id},
            )
            return LLMResponse(
                content=STATIC_TEMPLATE_RESPONSE,
                model="static_template",
                degraded=True,
                degrade_reason="token_budget_exhausted",
            )

        last_error: Optional[Exception] = None
        for model in self._models:
            if self._is_circuit_open(model):
                logger.warning(
                    "模型 %s 熔断中，跳过",
                    model,
                    extra={"trace_id": trace_id},
                )
                continue
            try:
                response = self._call_model(model, messages)
                self._record_success(model)
                return response
            except Exception as e:
                last_error = e
                self._record_failure(model)
                logger.error(
                    "模型 %s 调用失败，降级至下一模型",
                    model,
                    extra={"trace_id": trace_id, "error": str(e)},
                )

        # 全链路降级，使用静态模板
        logger.error(
            "全链路降级，使用静态模板",
            extra={"trace_id": trace_id, "last_error": str(last_error)},
        )
        return LLMResponse(
            content=STATIC_TEMPLATE_RESPONSE,
            model="static_template",
            degraded=True,
            degrade_reason="all_models_failed",
        )

    def _call_model(self, model: str, messages: list[dict[str, str]]) -> LLMResponse:
        """调用指定模型（子类或集成时实现具体 HTTP 调用）"""
        # ASSUMPTION: 具体的 HTTP 调用逻辑在集成第三方 SDK 时实现
        # 此处为骨架实现，确保接口契约正确
        raise NotImplementedError(f"模型 {model} 的调用逻辑待集成第三方 SDK 实现")

    def _check_token_budget(self, user_id: str) -> bool:
        """校验 Token 预算是否充足"""
        budget = settings.token_budget
        if self._token_usage.daily_used >= budget.daily_budget:
            return False
        if self._token_usage.monthly_used >= budget.monthly_budget:
            return False
        user_used = self._token_usage.per_user_used.get(user_id, 0)
        if user_used >= budget.max_cost_per_user:
            return False
        return True

    def _is_circuit_open(self, model: str) -> bool:
        """检查模型熔断器是否打开"""
        failures = self._circuit_breaker_failures.get(model, 0)
        return failures >= self._config.circuit_breaker_threshold

    def _record_failure(self, model: str) -> None:
        """记录模型调用失败"""
        self._circuit_breaker_failures[model] = (
            self._circuit_breaker_failures.get(model, 0) + 1
        )

    def _record_success(self, model: str) -> None:
        """记录模型调用成功，重置熔断计数"""
        self._circuit_breaker_failures[model] = 0

    def record_token_usage(
        self, user_id: str, input_tokens: int, output_tokens: int
    ) -> None:
        """记录 Token 消耗"""
        total = input_tokens + output_tokens
        self._token_usage.daily_used += total
        self._token_usage.monthly_used += total
        self._token_usage.per_user_used[user_id] = (
            self._token_usage.per_user_used.get(user_id, 0) + total
        )

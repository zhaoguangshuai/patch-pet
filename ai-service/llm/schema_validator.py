"""LLM 输出 Schema 强校验

LLM 输出必须符合以下结构，不合规输出由安全网关直接拦截：
{
    "intent": "字符串，意图描述",
    "action": "枚举：execute_tool / notice / deny",
    "risk_level": "枚举：P0 / P1 / P2",
    "need_confirm": "布尔值",
    "tool_calls": []
}
"""
import json
from dataclasses import dataclass
from typing import Any, Optional


VALID_ACTIONS = {"execute_tool", "notice", "deny"}
VALID_RISK_LEVELS = {"P0", "P1", "P2"}


@dataclass
class ValidatedIntent:
    """校验后的 LLM 输出意图"""
    intent: str
    action: str
    risk_level: str
    need_confirm: bool
    tool_calls: list[dict[str, Any]]


class SchemaValidationError(Exception):
    """Schema 校验失败异常"""
    def __init__(self, message: str, field: str = "") -> None:
        super().__init__(message)
        self.field = field


def validate_llm_output(raw_output: str) -> ValidatedIntent:
    """校验 LLM 输出是否符合 Schema

    Args:
        raw_output: LLM 原始输出字符串（JSON 格式）

    Returns:
        ValidatedIntent 校验通过的结构化意图

    Raises:
        SchemaValidationError 输出格式不合规
    """
    try:
        data = json.loads(raw_output)
    except json.JSONDecodeError as e:
        raise SchemaValidationError(
            f"LLM 输出非合法 JSON: {e}", field="root"
        ) from e

    if not isinstance(data, dict):
        raise SchemaValidationError("LLM 输出必须为 JSON 对象", field="root")

    # 校验 intent
    intent = data.get("intent")
    if not intent or not isinstance(intent, str):
        raise SchemaValidationError("缺少或无效的 intent 字段", field="intent")

    # 校验 action
    action = data.get("action")
    if action not in VALID_ACTIONS:
        raise SchemaValidationError(
            f"action 不合规: {action}，允许值: {VALID_ACTIONS}", field="action"
        )

    # 校验 risk_level
    risk_level = data.get("risk_level")
    if risk_level not in VALID_RISK_LEVELS:
        raise SchemaValidationError(
            f"risk_level 不合规: {risk_level}，允许值: {VALID_RISK_LEVELS}",
            field="risk_level",
        )

    # 校验 need_confirm
    need_confirm = data.get("need_confirm")
    if not isinstance(need_confirm, bool):
        raise SchemaValidationError(
            "need_confirm 必须为布尔值", field="need_confirm"
        )

    # 校验 tool_calls
    tool_calls = data.get("tool_calls", [])
    if not isinstance(tool_calls, list):
        raise SchemaValidationError(
            "tool_calls 必须为数组", field="tool_calls"
        )

    return ValidatedIntent(
        intent=intent,
        action=action,
        risk_level=risk_level,
        need_confirm=need_confirm,
        tool_calls=tool_calls,
    )

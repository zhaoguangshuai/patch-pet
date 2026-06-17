"""LLM 输出 Schema 校验器测试

覆盖：正常场景 / 边界场景 / 异常场景
"""
import json

import pytest

from llm.schema_validator import (
    SchemaValidationError,
    validate_llm_output,
    ValidatedIntent,
)


class TestValidateLLMOutput:
    """LLM 输出 Schema 校验测试"""

    # ---- 正常场景 ----

    def test应该校验通过_当输出符合Schema时(self):
        """正常场景：LLM 输出完全符合 Schema"""
        output = json.dumps({
            "intent": "查询宠物当前疗程",
            "action": "execute_tool",
            "risk_level": "P2",
            "need_confirm": False,
            "tool_calls": [{"tool_name": "get_medical_episode", "input": {"pet_id": "pet_001"}}],
        })
        result = validate_llm_output(output)
        assert isinstance(result, ValidatedIntent)
        assert result.intent == "查询宠物当前疗程"
        assert result.action == "execute_tool"
        assert result.risk_level == "P2"
        assert result.need_confirm is False
        assert len(result.tool_calls) == 1

    def test应该校验通过_当action为notice时(self):
        """正常场景：action 为 notice"""
        output = json.dumps({
            "intent": "通知用户",
            "action": "notice",
            "risk_level": "P1",
            "need_confirm": True,
            "tool_calls": [],
        })
        result = validate_llm_output(output)
        assert result.action == "notice"
        assert result.need_confirm is True

    def test应该校验通过_当action为deny时(self):
        """正常场景：action 为 deny（红线拦截）"""
        output = json.dumps({
            "intent": "禁止AI诊断",
            "action": "deny",
            "risk_level": "P0",
            "need_confirm": False,
            "tool_calls": [],
        })
        result = validate_llm_output(output)
        assert result.action == "deny"
        assert result.risk_level == "P0"

    # ---- 边界场景 ----

    def test应该校验通过_当tool_calls为空时(self):
        """边界场景：tool_calls 为空列表"""
        output = json.dumps({
            "intent": "通知",
            "action": "notice",
            "risk_level": "P2",
            "need_confirm": False,
            "tool_calls": [],
        })
        result = validate_llm_output(output)
        assert result.tool_calls == []

    # ---- 异常场景 ----

    def test应该抛出异常_当输入非JSON时(self):
        """异常场景：输入非法 JSON"""
        with pytest.raises(SchemaValidationError) as exc_info:
            validate_llm_output("not json")
        assert "非合法 JSON" in str(exc_info.value)

    def test应该抛出异常_当缺少intent时(self):
        """异常场景：缺少 intent 字段"""
        output = json.dumps({
            "action": "notice",
            "risk_level": "P2",
            "need_confirm": False,
            "tool_calls": [],
        })
        with pytest.raises(SchemaValidationError) as exc_info:
            validate_llm_output(output)
        assert "intent" in str(exc_info.value)

    def test应该抛出异常_当action不合规时(self):
        """异常场景：action 值不在允许范围"""
        output = json.dumps({
            "intent": "测试",
            "action": "invalid_action",
            "risk_level": "P2",
            "need_confirm": False,
            "tool_calls": [],
        })
        with pytest.raises(SchemaValidationError) as exc_info:
            validate_llm_output(output)
        assert "action" in str(exc_info.value)

    def test应该抛出异常_当risk_level不合规时(self):
        """异常场景：risk_level 值不在允许范围"""
        output = json.dumps({
            "intent": "测试",
            "action": "notice",
            "risk_level": "P99",
            "need_confirm": False,
            "tool_calls": [],
        })
        with pytest.raises(SchemaValidationError) as exc_info:
            validate_llm_output(output)
        assert "risk_level" in str(exc_info.value)

    def test应该抛出异常_当need_confirm非布尔时(self):
        """异常场景：need_confirm 不是布尔值"""
        output = json.dumps({
            "intent": "测试",
            "action": "notice",
            "risk_level": "P2",
            "need_confirm": "yes",
            "tool_calls": [],
        })
        with pytest.raises(SchemaValidationError) as exc_info:
            validate_llm_output(output)
        assert "need_confirm" in str(exc_info.value)

    def test应该抛出异常_当输出为JSON对象外类型时(self):
        """异常场景：输出为 JSON 数组而非对象"""
        with pytest.raises(SchemaValidationError):
            validate_llm_output("[]")

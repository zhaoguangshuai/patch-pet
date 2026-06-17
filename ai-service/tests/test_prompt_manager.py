"""Prompt 版本管理器测试

覆盖：正常场景 / 边界场景 / 异常场景
"""
import os
import tempfile
from pathlib import Path

import pytest
import yaml

from prompt.manager import PromptManager, MIN_HISTORY_VERSIONS


@pytest.fixture
def prompt_dir(tmp_path: Path) -> Path:
    """创建临时 Prompt 目录"""
    templates_dir = tmp_path / "templates"
    templates_dir.mkdir()

    # 创建多个版本的 prompt 文件
    for version in ["v1.0", "v1.1", "v1.2"]:
        data = {
            "name": "test_prompt",
            "version": version,
            "content": f"这是 {version} 版本的内容",
        }
        file_path = templates_dir / f"test_prompt_{version}.yaml"
        with open(file_path, "w", encoding="utf-8") as f:
            yaml.dump(data, f, allow_unicode=True)

    return templates_dir


class TestPromptManager:
    """Prompt 版本管理器测试"""

    # ---- 正常场景 ----

    def test应该加载所有模板_当目录存在时(self, prompt_dir: Path):
        """正常场景：加载目录下所有 yaml 文件"""
        manager = PromptManager(prompt_dir)
        manager.load_all()
        versions = manager.list_versions("test_prompt")
        assert len(versions) == 3

    def test应该返回激活版本_当未指定版本时(self, prompt_dir: Path):
        """正常场景：不指定版本返回当前激活版本（最新版本）"""
        manager = PromptManager(prompt_dir)
        manager.load_all()
        content = manager.get_prompt("test_prompt")
        assert "v1.2" in content

    def test应该返回指定版本_当指定版本时(self, prompt_dir: Path):
        """正常场景：指定版本返回对应内容"""
        manager = PromptManager(prompt_dir)
        manager.load_all()
        content = manager.get_prompt("test_prompt", version="v1.2")
        assert "v1.2" in content

    def test应该支持回滚_当目标版本存在时(self, prompt_dir: Path):
        """正常场景：回滚到指定版本"""
        manager = PromptManager(prompt_dir)
        manager.load_all()

        result = manager.rollback("test_prompt", "v1.2")
        assert result is True

        content = manager.get_prompt("test_prompt")
        assert "v1.2" in content

    # ---- 边界场景 ----

    def test应该返回空列表_当模板不存在时(self, prompt_dir: Path):
        """边界场景：查询不存在的模板"""
        manager = PromptManager(prompt_dir)
        manager.load_all()
        versions = manager.list_versions("nonexistent")
        assert versions == []

    def test应该返回False_当回滚到不存在版本时(self, prompt_dir: Path):
        """边界场景：回滚到不存在的版本"""
        manager = PromptManager(prompt_dir)
        manager.load_all()
        result = manager.rollback("test_prompt", "v99.0")
        assert result is False

    # ---- 异常场景 ----

    def test应该抛出异常_当模板不存在时(self, prompt_dir: Path):
        """异常场景：获取不存在的模板"""
        manager = PromptManager(prompt_dir)
        manager.load_all()
        with pytest.raises(KeyError):
            manager.get_prompt("nonexistent")

    def test应该抛出异常_当版本不存在时(self, prompt_dir: Path):
        """异常场景：获取不存在的版本"""
        manager = PromptManager(prompt_dir)
        manager.load_all()
        with pytest.raises(KeyError):
            manager.get_prompt("test_prompt", version="v99.0")

    def test应该正常处理_当目录为空时(self, tmp_path: Path):
        """边界场景：空目录"""
        empty_dir = tmp_path / "empty"
        empty_dir.mkdir()
        manager = PromptManager(empty_dir)
        manager.load_all()
        assert manager.list_versions("any") == []

"""Prompt 版本管理器

文件命名：prompt_v1.x.yaml
至少保留 3 份历史版本
支持动态加载 + 一键回滚
"""
import os
import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional

import yaml


PROMPT_DIR = Path(__file__).parent / "templates"
MIN_HISTORY_VERSIONS = 3


@dataclass
class PromptVersion:
    """Prompt 版本"""
    version: str  # 例如 v1.0, v1.1
    content: str
    file_path: str
    is_active: bool = False


@dataclass
class PromptTemplate:
    """Prompt 模板"""
    name: str  # 模板名称，例如 medical_disclaimer, dogwalk_intent
    versions: list[PromptVersion] = field(default_factory=list)

    @property
    def active_version(self) -> Optional[PromptVersion]:
        """获取当前激活版本"""
        for v in self.versions:
            if v.is_active:
                return v
        return self.versions[-1] if self.versions else None


def _parse_version(version_str: str) -> tuple[int, ...]:
    """解析版本号为可比较的元组

    Args:
        version_str: 版本号字符串，例如 "v1.0", "v1.2"

    Returns:
        版本号元组，例如 (1, 0), (1, 2)
    """
    cleaned = version_str.lstrip("v")
    parts = cleaned.split(".")
    return tuple(int(p) for p in parts if p.isdigit())


class PromptManager:
    """Prompt 版本管理器

    职责：
    - 动态加载 prompt_v1.x.yaml 文件
    - 维护至少 3 份历史版本
    - 支持一键回滚到指定版本
    """

    def __init__(self, prompt_dir: Optional[Path] = None) -> None:
        self._prompt_dir = prompt_dir or PROMPT_DIR
        self._templates: dict[str, PromptTemplate] = {}

    def load_all(self) -> None:
        """加载所有 Prompt 模板文件"""
        if not self._prompt_dir.exists():
            return

        for file_path in sorted(self._prompt_dir.glob("*.yaml")):
            self._load_file(file_path)

        # 按版本号排序
        for template in self._templates.values():
            template.versions.sort(key=lambda v: _parse_version(v.version))

        # 设置最新版本为激活版本
        for template in self._templates.values():
            if template.versions and not any(v.is_active for v in template.versions):
                template.versions[-1].is_active = True

    def get_prompt(self, name: str, version: str = "") -> str:
        """获取指定 Prompt 内容

        Args:
            name: 模板名称
            version: 版本号，为空时返回当前激活版本

        Returns:
            Prompt 文本内容

        Raises:
            KeyError 模板不存在
        """
        template = self._templates.get(name)
        if not template:
            raise KeyError(f"Prompt 模板不存在: {name}")

        if version:
            for v in template.versions:
                if v.version == version:
                    return v.content
            raise KeyError(f"Prompt 版本不存在: {name}@{version}")

        active = template.active_version
        if not active:
            raise KeyError(f"Prompt 模板无可用版本: {name}")
        return active.content

    def rollback(self, name: str, target_version: str) -> bool:
        """一键回滚到指定版本

        Args:
            name: 模板名称
            target_version: 目标版本号

        Returns:
            是否回滚成功
        """
        template = self._templates.get(name)
        if not template:
            return False

        found = False
        for v in template.versions:
            if v.version == target_version:
                v.is_active = True
                found = True
            else:
                v.is_active = False

        return found

    def list_versions(self, name: str) -> list[str]:
        """列出模板所有版本号"""
        template = self._templates.get(name)
        if not template:
            return []
        return [v.version for v in template.versions]

    def list_templates(self) -> list[str]:
        """列出所有模板名称"""
        return list(self._templates.keys())

    def get_version_count(self, name: str) -> int:
        """获取模板版本数量"""
        template = self._templates.get(name)
        return len(template.versions) if template else 0

    def has_minimum_history(self, name: str) -> bool:
        """检查是否满足最低历史版本要求（≥3）"""
        return self.get_version_count(name) >= MIN_HISTORY_VERSIONS

    def _load_file(self, file_path: Path) -> None:
        """加载单个 Prompt 文件"""
        try:
            with open(file_path, "r", encoding="utf-8") as f:
                data = yaml.safe_load(f)
            if not isinstance(data, dict):
                return

            name = data.get("name", file_path.stem)
            version = data.get("version", "v1.0")
            content = data.get("content", "")

            if name not in self._templates:
                self._templates[name] = PromptTemplate(name=name)

            self._templates[name].versions.append(PromptVersion(
                version=version,
                content=content,
                file_path=str(file_path),
                is_active=False,
            ))
        except Exception:
            # 加载失败不影响其他模板
            pass

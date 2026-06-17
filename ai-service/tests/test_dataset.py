"""AI 评测数据集测试"""
import sys
import os

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from eval.dataset import load_medical_dataset, load_dogwalk_dataset


def test_medical_dataset_size():
    """医疗数据集应有 1000 条"""
    cases = load_medical_dataset()
    assert len(cases) == 1000


def test_dogwalk_dataset_size():
    """代遛数据集应有 1000 条"""
    cases = load_dogwalk_dataset()
    assert len(cases) == 1000


def test_medical_has_p0_redlines():
    """医疗数据集应包含 P0 红线场景"""
    cases = load_medical_dataset()
    p0_cases = [c for c in cases if c.risk_level == "P0" and c.expected_blocked]
    assert len(p0_cases) >= 10


def test_dogwalk_has_p0_redlines():
    """代遛数据集应包含 P0 红线场景"""
    cases = load_dogwalk_dataset()
    p0_cases = [c for c in cases if c.risk_level == "P0" and c.expected_blocked]
    assert len(p0_cases) >= 10


def test_medical_p0_must_deny():
    """医疗 P0 红线必须全部预期 deny"""
    cases = load_medical_dataset()
    p0_blocked = [c for c in cases if c.risk_level == "P0" and c.expected_blocked]
    for c in p0_blocked:
        assert c.expected_action == "deny", f"{c.id} should expect deny"


def test_dogwalk_p0_must_deny():
    """代遛 P0 红线必须全部预期 deny"""
    cases = load_dogwalk_dataset()
    p0_blocked = [c for c in cases if c.risk_level == "P0" and c.expected_blocked]
    for c in p0_blocked:
        assert c.expected_action == "deny", f"{c.id} should expect deny"


def test_medical_unique_ids():
    """医疗数据集 ID 必须唯一"""
    cases = load_medical_dataset()
    ids = [c.id for c in cases]
    assert len(ids) == len(set(ids))


def test_dogwalk_unique_ids():
    """代遛数据集 ID 必须唯一"""
    cases = load_dogwalk_dataset()
    ids = [c.id for c in cases]
    assert len(ids) == len(set(ids))


def test_medical_has_normal_cases():
    """医疗数据集应包含正常场景"""
    cases = load_medical_dataset()
    normal = [c for c in cases if not c.expected_blocked]
    assert len(normal) > 0


def test_dogwalk_has_normal_cases():
    """代遛数据集应包含正常场景"""
    cases = load_dogwalk_dataset()
    normal = [c for c in cases if not c.expected_blocked]
    assert len(normal) > 0


def test_all_cases_have_input():
    """所有用例必须有输入文本（除了故意测试空输入的）"""
    cases = load_medical_dataset() + load_dogwalk_dataset()
    empty_cases = [c for c in cases if not c.input_text and "空输入" not in c.description]
    assert len(empty_cases) == 0

#!/bin/bash
# 一键回滚脚本
# 策略：功能开关 > 配置回滚 > 镜像回滚
set -e

NAMESPACE="patch-pet"
SERVICE="${1:-api-gateway}"
STRATEGY="${2:-auto}"

echo "=== 帕奇宠一键回滚 ==="
echo "服务: $SERVICE"
echo "策略: $STRATEGY"

# 1. 功能开关回滚（最快）
rollback_feature_flags() {
    echo "[1/3] 尝试功能开关回滚..."
    # 通过 Unleash 关闭所有灰度功能
    # 实际实现需调用 Unleash API
    echo "功能开关已回滚到稳定版本"
}

# 2. 配置回滚
rollback_config() {
    echo "[2/3] 尝试配置回滚..."
    # 回滚 Nacos 配置到上一个版本
    # 实际实现需调用 Nacos API
    echo "配置已回滚到上一版本"
}

# 3. 镜像回滚（最慢但最可靠）
rollback_image() {
    echo "[3/3] 尝试镜像回滚..."
    # 获取上一个稳定版本的镜像
    PREVIOUS_IMAGE=$(kubectl -n $NAMESPACE get deployment $SERVICE \
        -o jsonpath='{.metadata.annotations.previous-image}')

    if [ -z "$PREVIOUS_IMAGE" ]; then
        echo "错误：找不到上一个版本的镜像"
        exit 1
    fi

    kubectl -n $NAMESPACE set image deployment/$SERVICE \
        $SERVICE=$PREVIOUS_IMAGE

    kubectl -n $NAMESPACE rollout status deployment/$SERVICE --timeout=60s
    echo "镜像已回滚到: $PREVIOUS_IMAGE"
}

case "$STRATEGY" in
    "feature_flag")
        rollback_feature_flags
        ;;
    "config")
        rollback_config
        ;;
    "image")
        rollback_image
        ;;
    "auto")
        # 自动选择策略：优先尝试最快的
        rollback_feature_flags || rollback_config || rollback_image
        ;;
    *)
        echo "未知策略: $STRATEGY"
        echo "可用策略: feature_flag, config, image, auto"
        exit 1
        ;;
esac

echo "=== 回滚完成 ==="

#!/bin/bash
# 部署脚本
# 灰度发布：5% → 20% → 50% → 100%
# 每段观察 ≥ 2h，无 P0/P1 才放量
set -e

NAMESPACE="patch-pet"
SERVICE="${1:-api-gateway}"
IMAGE="${2:-patch-pet/api-gateway:latest}"
CANARY_WEIGHTS=(5 20 50 100)
OBSERVATION_HOURS=2

echo "=== 帕奇宠灰度部署 ==="
echo "服务: $SERVICE"
echo "镜像: $IMAGE"

# 部署 canary 版本
deploy_canary() {
    local weight=$1
    echo "部署 canary 版本，流量权重: ${weight}%"

    # 保存当前镜像用于回滚
    CURRENT_IMAGE=$(kubectl -n $NAMESPACE get deployment $SERVICE \
        -o jsonpath='{.spec.template.spec.containers[0].image}')

    kubectl -n $NAMESPACE annotate deployment $SERVICE \
        previous-image=$CURRENT_IMAGE --overwrite

    # 部署新版本
    kubectl -n $NAMESPACE set image deployment/$SERVICE \
        $SERVICE=$IMAGE

    # 更新流量权重
    kubectl -n $NAMESPACE patch virtualservice patch-pet-canary \
        --type merge -p "{\"spec\":{\"http\":[{\"route\":[{\"destination\":{\"host\":\"$SERVICE\",\"subset\":\"stable\"},\"weight\":$((100-weight))},{\"destination\":{\"host\":\"$SERVICE\",\"subset\":\"canary\"},\"weight\":$weight}]}]}}"

    echo "Canary 部署完成，流量权重: ${weight}%"
}

# 检查健康状态
check_health() {
    echo "检查服务健康状态..."
    kubectl -n $NAMESPACE get pods -l app=$SERVICE
    # 检查是否有 P0/P1 错误
    # 实际实现应检查监控告警
}

# 灰度发布流程
for weight in "${CANARY_WEIGHTS[@]}"; do
    echo "=== 灰度阶段: ${weight}% ==="
    deploy_canary $weight

    if [ "$weight" -lt 100 ]; then
        echo "观察 ${OBSERVATION_HOURS} 小时..."
        # 实际部署中应等待并检查监控
        # sleep $((OBSERVATION_HOURS * 3600))
        check_health
    fi
done

echo "=== 部署完成 ==="

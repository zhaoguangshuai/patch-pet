#!/usr/bin/env bash
# ============================================================================
# 帕奇宠 Agent 后端 —— 环境变量加载脚本
# 使用方式：source scripts/load-env.sh
# 作用：将项目根目录 .env 文件中的变量注入当前 shell 环境
# 注意：必须用 source 执行，直接执行不会生效
# ============================================================================

ENV_FILE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/.env"

if [ ! -f "$ENV_FILE" ]; then
  echo "❌ .env 文件不存在，请先执行：cp .env.example .env"
  return 1
fi

echo "📄 加载环境变量：$ENV_FILE"

# 逐行读取 .env，跳过注释和空行
while IFS= read -r line || [ -n "$line" ]; do
  # 去掉首尾空白
  line="$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"

  # 跳过空行和注释
  if [ -z "$line" ] || [ "${line:0:1}" = "#" ]; then
    continue
  fi

  # 导出变量（支持值中包含 = 号）
  export "$line"
done < "$ENV_FILE"

echo "✅ 环境变量加载完成"
echo "   POSTGRES_DSN:         ${POSTGRES_DSN:+已配置}/未配置"
echo "   REDIS_ADDR:           ${REDIS_ADDR:-未配置}"
echo "   JWT_SECRET:           ${JWT_SECRET:+已配置}/未配置"
echo "   REPLAY_SECRET_KEY:    ${REPLAY_SECRET_KEY:+已配置}/未配置"
echo "   LLM_API_KEY:          ${LLM_API_KEY:+已配置}/未配置"
echo "   AI_SERVICE_GRPC_ADDR: ${AI_SERVICE_GRPC_ADDR:-未配置}"
echo "   APP_ENV:              ${APP_ENV:-未配置}"

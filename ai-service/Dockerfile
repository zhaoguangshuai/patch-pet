# 帕奇宠 AI 服务 Dockerfile
# 强制使用多阶段构建优化镜像体积
# 禁止写入明文密钥，运行时容器注入

# ---- 构建阶段 ----
FROM python:3.12-slim AS builder

WORKDIR /build

# 安装依赖（利用缓存层）
COPY requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

# ---- 运行阶段 ----
FROM python:3.12-slim

WORKDIR /app

# 从构建阶段复制已安装的依赖
COPY --from=builder /install /usr/local

# 复制应用代码
COPY . .

# 非 root 用户运行
RUN useradd -r -s /bin/false aiuser
USER aiuser

# 环境变量（运行时注入，禁止在此写入密钥）
ENV PYTHONUNBUFFERED=1
ENV GRPC_HOST=0.0.0.0
ENV GRPC_PORT=50051

EXPOSE 50051

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD python -c "import grpc; channel = grpc.insecure_channel('localhost:50051'); channel.close()" || exit 1

CMD ["python", "main.py"]

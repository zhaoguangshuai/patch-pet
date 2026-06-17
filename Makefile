.PHONY: run dev test docker-up docker-down docker-logs lint clean test-coverage help

help:
	@echo "帕奇宠 Agent 后端 —— 开发命令"
	@echo ""
	@echo "  make docker-up    启动所有依赖服务（PostgreSQL + Redis + Kafka）"
	@echo "  make docker-down  停止依赖服务"
	@echo "  make run          加载 .env + 启动 Go 主服务"
	@echo "  make dev          加载 .env + 同时启动 Go 主服务和 Python AI 服务"
	@echo "  make test         加载 .env + 运行全部测试"
	@echo "  make test-coverage生成覆盖率报告"
	@echo "  make lint         运行代码检查"
	@echo "  make clean        清理构建产物"

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

run:
	export $$(grep -v '^\s*#' .env | grep -v '^\s*$$' | xargs) && \
		go run ./cmd/...

dev:
	export $$(grep -v '^\s*#' .env | grep -v '^\s*$$' | xargs) && \
		(trap 'kill 0' SIGINT; \
		 go run ./cmd/... & \
		 echo "🐍 启动 Python AI 服务..." && \
		 cd ai-service && [ ! -d .venv ] && python -m venv .venv; \
		 [ -f .venv/bin/activate ] && . .venv/bin/activate && python main.py & \
		 wait)

test:
	export $$(grep -v '^\s*#' .env | grep -v '^\s*$$' | xargs) && \
		go test ./... -v -count=1

test-coverage:
	export $$(grep -v '^\s*#' .env | grep -v '^\s*$$' | xargs) && \
		go test ./... -coverprofile=coverage.out -count=1 && \
		go tool cover -html=coverage.out -o coverage.html

lint:
	go vet ./...

clean:
	rm -f coverage.out coverage.html

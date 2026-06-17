# PatchPet — 帕奇宠·AI Agent 后端

帕奇宠是面向多宠家庭的智能照护生命流系统。后端采用 **Go 主服务 + Python AI 服务** 微服务架构，通过 LAKI Agent 编排引擎实现医疗居家任务调度与代遛狗服务闭环。

```
客户端请求/定时任务/第三方回调 → API Gateway → 身份认证 & 权限
                                                     ↓
                                               LAKI 意图路由
                                                   ↓
                          ┌──────────────────────────────────────┐
                          │                                      │
                   医疗居家 Agent                        代遛狗服务 Agent
                          │                                      │
                    ┌─────┴─────┐                         ┌─────┴─────┐
                 状态机      安全网关                   状态机      风险网关
                    │                                      │
               ┌───┼───┐                              ┌───┼───┐
            设备  诊所  通知                         服务商  地图  支付
              │                                      │
              └──────────────┬───────────────────────┘
                             ↓
                   生命流事件服务 · 审计日志 · 可观测
```

---

## 目录

1. [快速开始](#快速开始)（5 分钟跑通本地开发）
2. [服务架构](#服务架构)（Go + Python 双服务分工）
3. [代码框架说明](#代码框架说明)（每个目录的职责）
4. [核心模块](#核心模块)
5. [测试](#测试) · [端口速查](#端口速查) · [技术栈](#技术栈)
6. [设计文档](#设计文档)

---

## 快速开始

### 前置条件

- Go 1.25+
- Python 3.11+
- Docker（运行 PostgreSQL + Redis + Kafka 等依赖服务）
- Kafka（可选，本地开发可用内存模式替代）

### 开发命令速查

项目提供 `Makefile` 封装常用开发操作：

| 命令 | 作用 |
|------|------|
| `make docker-up` | 启动 PostgreSQL + Redis + Kafka 依赖服务 |
| `make docker-down` | 停止依赖服务 |
| `make docker-logs` | 查看依赖服务日志 |
| `make run` | 加载 `.env` + 启动 Go 主服务 |
| `make dev` | 加载 `.env` + 同时启动 Go 和 Python AI 服务 |
| `make test` | 运行全部 Go 测试 |
| `make test-coverage` | 生成覆盖率报告 |
| `make lint` | 代码静态检查（go vet） |
| `make clean` | 清理构建产物 |

### Step 1：克隆 & 安装依赖

```bash
git clone <repo> patch-pet
cd patch-pet

# Go 依赖
go mod download

# Python AI 服务依赖
cd ai-service
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
cd ..
```

### Step 2：配置

```bash
cp .env.example .env
```

编辑 `.env`，需要填写的最少配置：

```env
# 数据库连接串（格式：postgres://用户名:密码@主机:端口/数据库名?sslmode=disable）
POSTGRES_DSN=postgres://patchpet:patchpet_dev@localhost:5432/patchpet?sslmode=disable

# Redis
REDIS_ADDR=localhost:6379

# LLM API Key（无可用 Key 时 AI 功能不可用，不影响服务启动）
LLM_API_KEY=

# 认证密钥（本地开发随意填写，生产环境必须为强密钥）
JWT_SECRET=patchpet-local-dev-jwt-secret
```

### Step 3：启动依赖服务

```bash
# 一键启动 PostgreSQL + Redis + Kafka
make docker-up
```

> 也可使用 `docker compose up -d` 手动启动，所有服务定义在 `docker-compose.yaml` 中。

### Step 4：数据库迁移

```bash
go run ./cmd/main.go migrate
```

### Step 5：启动服务

```bash
# 启动 Go 主服务（开发模式，自动加载 .env）
make run

# 同时启动 Go + Python AI 服务
make dev
```

启动成功后：

```
帕奇宠 Agent 后端服务启动，监听 :8080
AI service gRPC server started on :50051
AI service Kafka consumer started
```

### Step 6：验证

```bash
# 健康检查
curl http://localhost:8080/health

# 创建医疗疗程
curl -X POST http://localhost:8080/api/v1/medical-episodes \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Trace-Id: $(go run ./cmd/ulid)" \
  -H "Content-Type: application/json"

# 查看指标
curl http://localhost:8080/metrics
```

---

## 服务架构

系统采用 **Go 主服务 + Python AI 服务** 双服务架构，资源和部署完全隔离。

### Go 主服务（线上核心）

API 网关、认证、权限、状态机、Saga 事务、数据库、Redis、Kafka、支付、诊所/服务商对接、生命流、审计、配置中心、功能开关。

### Python AI 服务（AI/离线）

LLM 推理、RAG 向量检索、Prompt 管理、AI 自动化评测、数据清洗、离线脚本。

### 通信协议

| 场景 | 协议 | 说明 |
|------|------|------|
| 实时推理（摘要、意图识别） | gRPC | 低延迟、强类型 |
| 异步任务（RAG 更新、评测上报） | Kafka 事件总线 | 异步解耦 |

---

## 代码框架说明

```
patch-pet/
├── Makefile                       #   开发命令（docker-up / run / dev / test / lint）
├── docker-compose.yaml            #   本地依赖服务编排（PostgreSQL + Redis + Kafka）
├── .env.example                   #   环境变量模板（首次使用 cp 为 .env）
├── CLAUDE.md                      #   AI 辅助编码规范（全技术栈）
│
├── cmd/                           # Go 程序入口
│   └── main.go                    #   启动入口：路由装配、中间件、服务初始化
│
├── internal/                      # Go 业务核心（不对外暴露）
│   ├── agent/                     #   LAKI Agent 编排引擎
│   │   ├── agent.go               #     Agent 核心生命周期
│   │   ├── intent_router.go       #     意图路由分发
│   │   ├── llm_chain.go           #     LLM 调用链路
│   │   └── token_budget.go        #     Token 预算控制
│   ├── medical/                   #   医疗居家 Agent 模块 ★
│   │   ├── handler.go             #     HTTP 处理器
│   │   ├── statemachine.go        #     医疗疗程状态机
│   │   ├── model.go               #     领域模型
│   │   ├── repository.go          #     数据访问层
│   │   ├── safety_gateway.go      #     医疗安全网关（红线拦截）
│   │   ├── scheduler.go           #     定时任务调度
│   │   ├── escalation.go          #     异常升级规则
│   │   ├── migration.go           #     数据库迁移
│   │   └── popup_blocker.go       #     医疗弹窗拦截
│   ├── dogwalk/                   #   代遛狗服务 Agent 模块 ★
│   │   ├── handler.go             #     HTTP 处理器
│   │   ├── statemachine.go        #     代遛订单状态机
│   │   ├── model.go               #     领域模型
│   │   ├── repository.go          #     数据访问层
│   │   ├── risk_gateway.go        #     服务风险拦截网关
│   │   └── migration.go           #     数据库迁移
│   ├── runtime/                   #   Agent Runtime 执行引擎
│   │   ├── runtime.go             #     运行时调度
│   │   └── gateway.go             #     工具网关
│   ├── tools/                     #   工具注册中心
│   │   └── registry.go            #     工具注册 & 发现
│   ├── workflow/                  #   Saga 分布式事务 + 通用状态机
│   │   ├── saga.go                #     Saga 编排引擎
│   │   └── statemachine.go        #     通用状态机框架
│   ├── policy/                    #   策略引擎（权限 & 风控）
│   │   └── engine.go              #     策略执行引擎
│   ├── memory/                    #   三层记忆管理
│   │   ├── memory.go              #     记忆类型定义
│   │   └── manager.go             #     记忆读写管理器
│   ├── eventbus/                  #   事件总线（Kafka 生产/消费）
│   │   ├── event.go               #     事件标准结构
│   │   ├── publisher.go           #     事件发布
│   │   └── ai_bridge.go           #     Go ↔ AI 服务桥接
│   ├── auth/                      #   身份认证 & 权限
│   │   ├── auth.go                #     认证核心逻辑
│   │   ├── jwt_auth.go            #     JWT 鉴权
│   │   └── dual_auth.go           #     能力权限双层校验
│   ├── audit/                     #   全链路审计日志
│   │   ├── audit.go               #     审计核心
│   │   └── recorder.go            #     审计记录器
│   ├── lifeflow/                  #   生命流事件服务
│   │   ├── event.go               #     事件定义
│   │   └── writer.go              #     事件写入
│   ├── middleware/                #   HTTP 中间件层
│   │   ├── gateway.go             #     API 网关
│   │   ├── idempotency.go         #     幂等校验
│   │   ├── ratelimit.go           #     限流
│   │   ├── ipblock.go             #     IP 拦截
│   │   ├── trace.go               #     链路追踪
│   │   ├── replay.go              #     防重放
│   │   └── pagination.go          #     分页解析
│   ├── database/                  #   数据库基础设施
│   │   ├── database.go            #     连接管理
│   │   ├── migrate.go             #     迁移工具
│   │   ├── repository.go          #     通用 Repository
│   │   └── timescale.go           #     TimescaleDB 时序
│   ├── health/                    #   健康检查
│   │   └── health.go
│   ├── hitl/                      #   人机介入（HITL）
│   │   └── hitl.go
│   ├── rag/                       #   RAG 客户端（gRPC 调 Python）
│   │   └── client.go
│   └── compliance/                #   合规 & 数据留存
│       └── retention.go
│
├── pkg/                           # Go 公共包
│   ├── constants/                 #   全局常量（error_code, threshold）
│   ├── types/                     #   通用类型（ID、响应、工具接口、时间）
│   ├── utils/                     #   工具函数（脱敏、ULID、重试、校验、安全、时间）
│   ├── logger/                    #   日志封装（zap）
│   └── thirdparty/                #   第三方 SDK 封装
│       ├── redis.go               #     Redis
│       ├── grpc.go                #     gRPC（→ Python AI 服务）
│       ├── nacos.go               #     配置中心
│       ├── unleash.go             #     功能开关
│       ├── client.go              #     HTTP 客户端
│       └── errors.go              #     第三方错误定义
│
├── ai-service/                    # Python AI 独立服务
│   ├── main.py                    #   启动入口
│   ├── Dockerfile                 #   容器构建
│   ├── requirements.txt           #   Python 依赖
│   ├── llm/                       #   LLM 推理（client, schema_validator）
│   ├── rag/                       #   RAG 向量检索（embedder, retriever, vector_store）
│   ├── prompt/                    #   Prompt 管理（manager + YAML 模板）
│   │   └── templates/             #     dogwalk_intent, medical_disclaimer
│   ├── eval/                      #   AI 自动化评测（dataset, runner）
│   ├── api/                       #   gRPC 服务端 + Kafka 消费者
│   ├── config/                    #   配置管理（settings）
│   └── tests/                     #   Python 测试用例
│
├── deploy/                        # 部署配置
│   ├── k8s/                       #   K8s 编排（api-gateway, medical, dogwalk, agent-runtime, canary）
│   └── docker/                    #   Docker 构建（预留）
│
├── scripts/                       # 运维脚本
│   ├── deploy.sh                  #   部署脚本
│   ├── rollback.sh                #   回滚脚本
│   └── load-env.sh                #   .env 加载脚本（source 使用）
│
└── docs/                          # 项目设计文档
    ├── 01-总体架构设计.md          #   架构总览、状态机、Saga、部署
    ├── 02-接口契约文档.md          #   接口协议、数据库表、事件、错误码
    ├── 03-AI编码规范.md            #   编码约束、目录标准、安全规则
    └── 04-测试风险与运维手册.md     #   测试体系、风险评估、SOP
```

---

## 核心模块

### 医疗居家 Agent（16号模块 · P0 高危）

基于诊所医嘱完成护理调度、数据汇总、诊所对接。Agent 仅辅助执行，**不参与医疗决策**。

| 能力 | 说明 |
|------|------|
| 医嘱解析 | 解析诊所医嘱为可执行护理任务 |
| 任务调度 | 状态机管控任务执行、提醒、延后、超时 |
| 数据汇总 | 设备数据汇聚、周期摘要生成 |
| 授权管理 | 医疗数据外发授权审批 |
| 诊所对接 | 摘要推送、复诊数据同步 |

### 代遛狗服务 Agent（19号模块 · P1 中高危）

自动识别遛狗需求、完成服务商编排、订单支付、履约监控。Agent 仅做流程编排，**不干预用户消费决策**。

| 能力 | 说明 |
|------|------|
| 需求识别 | 基于设备数据/时间规律识别遛狗需求 |
| 服务商筛选 | 可用服务商匹配与路线规划 |
| 订单履约 | 支付 → 预约 → 服务 → 报告全链路 |
| 实时监控 | 服务过程轨迹追踪与异常检测 |

### Agent Runtime 执行链路

所有 Agent 工具调用必须遵循固定链路，禁止跳步：

```
LLM 意图生成 → 工具调用意图 → 工具网关 → 策略引擎(权限/风控) → 工具执行
```

**硬性约束**：
- Agent 禁止直接操作数据库、直连第三方接口
- 所有工具调用必须经过工具网关 + 策略引擎双重校验

### 全局运行阈值

| 配置项 | 数值 |
|--------|------|
| 单次会话最大工具调用数 | 5 |
| 工具嵌套最大递归深度 | 3 |
| LLM 输入 Token 上限 | 8000 |
| LLM 输出 Token 上限 | 1500 |

---

## 测试

```bash
# 全量 Go 单元测试
go test ./... -v -count=1

# 特定模块测试
go test ./internal/medical/... -v
go test ./internal/dogwalk/... -v
go test ./internal/agent/... -v

# Python AI 服务测试
cd ai-service && source .venv/bin/activate
pytest tests/ -v

# 覆盖率报告
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## 端口速查

| 端口 | 服务 | 说明 |
|------|------|------|
| 8080 | Go API Gateway | 主服务 HTTP 端点 |
| 50051 | Python AI gRPC | LLM/RAG 推理服务 |
| 5432 | PostgreSQL | 业务数据库 |
| 6379 | Redis | 缓存/分布式锁 |
| 9092 | Kafka | 事件总线 |
| 8848 | Nacos | 配置中心（可选） |

---

## 技术栈

| 组件 | 版本 | 用途 |
|------|------|------|
| Go | 1.25+ | 主服务语言 |
| Gin | latest | Web 框架 |
| GORM | latest | ORM |
| Python | 3.11+ | AI 服务语言 |
| PostgreSQL | 16+ | 核心业务数据库 |
| TimescaleDB | latest | 时序数据（轨迹/设备） |
| Redis | 7+ | 缓存/分布式锁/会话 |
| Kafka | latest | 事件总线 |
| Langfuse | latest | LLM 可观测（可选） |
| Prometheus + Grafana | latest | 指标监控 |
| Sentry | latest | 错误追踪 |

---

## 设计文档

| 文档 | 内容 |
|------|------|
| [docs/01-总体架构设计.md](docs/01-总体架构设计.md) | 项目概述、设计原则、架构分层、状态机、Saga、Memory、中间件、双服务通信、K8s 部署、LLM 降级、HITL、伦理红线 |
| [docs/02-接口契约文档.md](docs/02-接口契约文档.md) | 全局规范、ID 规则、错误码、响应格式、Kafka 事件、LLM Schema、数据库表结构、API 接口列表、工具接口、策略引擎 DSL、RAG 参数、安全规则 |
| [docs/03-AI编码规范.md](docs/03-AI编码规范.md) | 技术栈约束、语言分工、工程目录标准、命名规范、状态机代码规范、安全编码、日志规范 |
| [docs/04-测试风险与运维手册.md](docs/04-测试风险与运维手册.md) | 环境划分、分层测试、AI 评测、风险评估矩阵（P0/P1/P2）、数据生命周期、应急 SOP、灰度发布、巡检清单 |
| [docs/05-联调接口文档.md](docs/05-联调接口文档.md) | 联调接口定义、请求示例、响应格式、错误码速查 |
| [docs/06-技术选型分析.md](docs/06-技术选型分析.md) | 双语言架构选型分析、Agent 框架选型决策、未来迁移路径 |

---

## License

MIT

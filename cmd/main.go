// Package main 帕奇宠 Agent 后端服务入口
// 双服务架构：Go 主服务（线上核心）+ Python AI 服务（LLM/RAG/评测）物理隔离
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/patch-pet/patch-pet/internal/runtime"
	"github.com/patch-pet/patch-pet/pkg/constants"
	"github.com/patch-pet/patch-pet/pkg/thirdparty"
)

func main() {
	// 自动加载 .env 文件（不存在时不阻塞启动）
	if err := godotenv.Load(); err != nil {
		log.Printf("[INFO] .env 文件未找到，使用系统环境变量: %v", err)
	}

	// 初始化 Agent Runtime
	agentRuntime := runtime.New()
	gateway := runtime.NewToolGateway()
	executor := runtime.NewToolExecutor(gateway)

	// 初始化功能开关（Unleash）
	// 连接失败时降级为默认配置（所有 flag 关闭），不阻塞服务启动
	featureFlagClient := thirdparty.NewFeatureFlagClient(thirdparty.UnleashConfig{})
	if err := featureFlagClient.Init(); err != nil {
		log.Printf("[WARN] 功能开关初始化失败: %v", err)
	}
	_ = featureFlagClient

	// 注册工具（Default-Deny：需人工审批后方可启用）
	_ = agentRuntime
	_ = executor

	// HTTP 服务
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"code":0,"message":"ok","data":{"status":"healthy"}}`)
	})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 优雅启停
	go func() {
		log.Printf("帕奇宠 Agent 后端服务启动，监听 :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("收到退出信号，开始优雅停机（最长 %ds）...", constants.GracefulShutdownTimeoutSec)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(constants.GracefulShutdownTimeoutSec)*time.Second,
	)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("服务停机失败: %v", err)
	}
	log.Printf("服务已停止")
}

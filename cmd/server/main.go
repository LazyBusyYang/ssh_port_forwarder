package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ssh-port-forwarder/internal/config"
	"ssh-port-forwarder/internal/handler"
	"ssh-port-forwarder/internal/service"
)

func main() {
	// 1. 命令行参数
	configPath := flag.String("config", "config/config.yaml", "path to config file")
	flag.Parse()

	// 2. 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		// 如果配置文件不存在，使用默认配置
		log.Printf("Warning: failed to load config from %s: %v, using defaults", *configPath, err)
		cfg, err = config.Load("")
		if err != nil {
			log.Fatalf("Failed to load default config: %v", err)
		}
	}

	// 3. 初始化依赖注入容器
	container, err := service.NewContainer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize container: %v", err)
	}

	// 4. 启动服务（SSH Manager, Health Checker, LB Pool, Scheduler, 创建默认 admin）
	if err := container.Start(); err != nil {
		log.Fatalf("Failed to start services: %v", err)
	}
	log.Println("All services started successfully")

	// 5. 设置路由
	router := handler.SetupRouter(container)

	// 6. 启动 HTTP Server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("HTTP server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 7. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Received signal %v, shutting down...", sig)

	// 停止 HTTP Server（带超时）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// 停止所有内部服务
	container.Stop()
	log.Println("Server exited gracefully")
}

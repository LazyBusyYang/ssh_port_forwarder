package scheduler

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"ssh-port-forwarder/internal/config"
	"ssh-port-forwarder/internal/repository"
	"ssh-port-forwarder/internal/service/health"
	"ssh-port-forwarder/internal/service/lb"
	"ssh-port-forwarder/internal/service/ssh_manager"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	mu            sync.Mutex
	healthChecker *health.Checker
	sshManager    *ssh_manager.Manager
	lbPool        *lb.Pool
	auditRepo     repository.AuditLogRepository
	healthRepo    repository.HealthHistoryRepository
	config        *config.Config
	stopCh        chan struct{}
	wg            sync.WaitGroup
	// 用于跟踪正在重连的客户端，避免重复启动重连
	reconnectingClients map[uint64]bool
}

// NewScheduler 创建新的 Scheduler 实例
func NewScheduler(
	healthChecker *health.Checker,
	sshManager *ssh_manager.Manager,
	lbPool *lb.Pool,
	auditRepo repository.AuditLogRepository,
	healthRepo repository.HealthHistoryRepository,
	config *config.Config,
) *Scheduler {
	return &Scheduler{
		healthChecker:       healthChecker,
		sshManager:          sshManager,
		lbPool:              lbPool,
		auditRepo:           auditRepo,
		healthRepo:          healthRepo,
		config:              config,
		stopCh:              make(chan struct{}),
		reconnectingClients: make(map[uint64]bool),
	}
}

// Start 启动所有定时任务
func (s *Scheduler) Start() {
	log.Printf("[Scheduler] Starting all scheduled tasks...")

	// 1. Health Check（每 10s）
	s.runTicker("HealthCheck", 10*time.Second, func() {
		s.healthChecker.RunCheck()
	})

	// 2. Reconnect Loop（每 5s）
	s.runTicker("ReconnectLoop", 5*time.Second, s.runReconnectLoop)

	// 3. Metrics Flush（每 15s）
	s.runTicker("MetricsFlush", 15*time.Second, s.runMetricsFlush)

	// 4. Config Reload（每 30s + SIGHUP 信号）
	s.runTicker("ConfigReload", 30*time.Second, s.runConfigReload)
	s.startSignalListener()

	// 5. Cleanup Dead Connections（每 60s）
	s.runTicker("CleanupDeadConnections", 60*time.Second, s.runCleanupDeadConnections)

	// 6. Cleanup Audit Log（每 1h）
	s.runTicker("CleanupAuditLog", 1*time.Hour, s.runCleanupAuditLog)

	// 7. Cleanup Health History（每 1h）
	s.runTicker("CleanupHealthHistory", 1*time.Hour, s.runCleanupHealthHistory)

	log.Printf("[Scheduler] All scheduled tasks started")
}

// Stop 停止所有定时任务
func (s *Scheduler) Stop() {
	log.Printf("[Scheduler] Stopping all scheduled tasks...")
	close(s.stopCh)
	s.wg.Wait()
	log.Printf("[Scheduler] All scheduled tasks stopped")
}

// runTicker 通用定时任务运行器
func (s *Scheduler) runTicker(name string, interval time.Duration, fn func()) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		log.Printf("[Scheduler] %s started (interval: %v)", name, interval)
		for {
			select {
			case <-ticker.C:
				fn()
			case <-s.stopCh:
				log.Printf("[Scheduler] %s stopped", name)
				return
			}
		}
	}()
}

// runReconnectLoop 执行重连检查
// 遍历 sshManager.GetAllClients()，对状态为 Disconnected 或 Failed 的 Client 调用 StartReconnectLoop()
// 注意避免对同一个 Client 重复启动重连
func (s *Scheduler) runReconnectLoop() {
	clients := s.sshManager.GetAllClients()

	s.mu.Lock()
	defer s.mu.Unlock()

	for hostID, client := range clients {
		state := client.State()

		// 检查是否需要重连：状态为 Disconnected 或 Failed
		if state == ssh_manager.ConnStateDisconnected || state == ssh_manager.ConnStateFailed {
			// 避免重复启动重连
			if s.reconnectingClients[hostID] {
				continue
			}

			s.reconnectingClients[hostID] = true

			// 在 goroutine 中启动重连，避免阻塞调度器
			go func(id uint64, c *ssh_manager.SSHClient) {
				log.Printf("[Scheduler] Starting reconnect loop for host %d", id)
				c.StartReconnectLoop()

				// 重连完成后（无论成功或失败），清除标记
				s.mu.Lock()
				delete(s.reconnectingClients, id)
				s.mu.Unlock()
			}(hostID, client)
		}
	}
}

// runMetricsFlush 执行指标刷新
// 遍历 sshManager.GetAllClients()，将运行时状态写入 HealthHistory
func (s *Scheduler) runMetricsFlush() {
	clients := s.sshManager.GetAllClients()

	for hostID, client := range clients {
		host := client.GetHost()
		if host == nil {
			continue
		}

		// 获取客户端状态信息
		isConnected := client.IsConnected()
		state := client.State()

		// 根据连接状态计算健康分数
		var score float64
		var isHealthy bool
		if isConnected {
			score = 100.0
			isHealthy = true
		} else {
			score = 0.0
			isHealthy = false
		}

		// 记录历史（使用 healthChecker.RecordHistory 方法）
		s.healthChecker.RecordHistory(hostID, score, isHealthy, 0)

		log.Printf("[Scheduler] Metrics flushed for host %d: state=%s, score=%.2f",
			hostID, state.String(), score)
	}
}

// runConfigReload 执行配置重载
// 目前只需 log 一条消息表示配置已重载即可（占位实现）
func (s *Scheduler) runConfigReload() {
	log.Printf("[Scheduler] Config reload triggered (placeholder implementation)")
	// TODO: 实现配置重载逻辑
	// 1. 重新加载配置文件
	// 2. 应用新的配置
	// 3. 通知相关组件配置已变更
}

// startSignalListener 启动信号监听器，监听 SIGHUP 信号触发配置重载
func (s *Scheduler) startSignalListener() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGHUP)
		defer signal.Stop(sigCh)

		log.Printf("[Scheduler] Signal listener started (listening for SIGHUP)")

		for {
			select {
			case sig, ok := <-sigCh:
				if !ok {
					return
				}
				if sig == syscall.SIGHUP {
					log.Printf("[Scheduler] Received SIGHUP signal, triggering config reload")
					s.runConfigReload()
				}
			case <-s.stopCh:
				log.Printf("[Scheduler] Signal listener stopped")
				return
			}
		}
	}()
}

// runCleanupDeadConnections 清理长时间处于 Failed 状态的连接
func (s *Scheduler) runCleanupDeadConnections() {
	clients := s.sshManager.GetAllClients()

	for hostID, client := range clients {
		state := client.State()

		// 清理长时间处于 Failed 状态的连接
		if state == ssh_manager.ConnStateFailed {
			log.Printf("[Scheduler] Cleaning up dead connection for host %d", hostID)
			// 断开并清理该连接
			if err := s.sshManager.DisconnectHost(hostID); err != nil {
				log.Printf("[Scheduler] Failed to disconnect dead host %d: %v", hostID, err)
			}
		}
	}
}

// runCleanupAuditLog 清理 7 天前的审计日志
func (s *Scheduler) runCleanupAuditLog() {
	cutoffTime := time.Now().Add(-7 * 24 * time.Hour).Unix()

	if err := s.auditRepo.DeleteBefore(cutoffTime); err != nil {
		log.Printf("[Scheduler] Failed to cleanup audit logs: %v", err)
	} else {
		log.Printf("[Scheduler] Audit logs cleaned up (before %d)", cutoffTime)
	}
}

// runCleanupHealthHistory 清理超过配置天数的健康历史记录
func (s *Scheduler) runCleanupHealthHistory() {
	retentionDays := s.config.DataRetention.HealthHistoryDays
	if retentionDays <= 0 {
		retentionDays = 7 // 兜底默认值
	}

	cutoffTime := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour).Unix()

	if err := s.healthRepo.DeleteBefore(cutoffTime); err != nil {
		log.Printf("[Scheduler] Failed to cleanup health history: %v", err)
	} else {
		log.Printf("[Scheduler] Health history cleaned up (before %d, retention: %d days)", cutoffTime, retentionDays)
	}
}

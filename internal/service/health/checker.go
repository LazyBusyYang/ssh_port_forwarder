package health

import (
	"log"
	"sync"
	"time"

	"ssh-port-forwarder/internal/model"
	"ssh-port-forwarder/internal/repository"
	"ssh-port-forwarder/internal/service/ssh_manager"
)

type Checker struct {
	mu           sync.RWMutex
	hostRepo     repository.SSHHostRepository
	healthRepo   repository.HealthHistoryRepository
	ruleRepo     repository.ForwardRuleRepository
	sshManager   *ssh_manager.Manager
	eventCh      chan HealthEvent // 状态变更事件通道
	stopCh       chan struct{}
	// 滑动窗口：hostID -> []CheckResult（最近 5 分钟）
	checkWindows map[uint64][]windowEntry
	// WebSocket 订阅者管理
	wsSubsMu     sync.RWMutex
	wsSubs       map[chan HealthEvent]struct{}
}

type windowEntry struct {
	result    CheckResult
	timestamp time.Time
}

// NewChecker 创建新的 Checker 实例
func NewChecker(
	hostRepo repository.SSHHostRepository,
	healthRepo repository.HealthHistoryRepository,
	ruleRepo repository.ForwardRuleRepository,
	sshManager *ssh_manager.Manager,
) *Checker {
	return &Checker{
		hostRepo:     hostRepo,
		healthRepo:   healthRepo,
		ruleRepo:     ruleRepo,
		sshManager:   sshManager,
		eventCh:      make(chan HealthEvent, 100),
		stopCh:       make(chan struct{}),
		checkWindows: make(map[uint64][]windowEntry),
		wsSubs:       make(map[chan HealthEvent]struct{}),
	}
}

// SubscribeWS 为 WebSocket 客户端创建独立通道
func (c *Checker) SubscribeWS() chan HealthEvent {
	ch := make(chan HealthEvent, 100)
	c.wsSubsMu.Lock()
	c.wsSubs[ch] = struct{}{}
	c.wsSubsMu.Unlock()
	return ch
}

// UnsubscribeWS 取消订阅
func (c *Checker) UnsubscribeWS(ch chan HealthEvent) {
	c.wsSubsMu.Lock()
	delete(c.wsSubs, ch)
	close(ch)
	c.wsSubsMu.Unlock()
}

// broadcastToWS 广播事件到所有 WS 订阅者
func (c *Checker) broadcastToWS(event HealthEvent) {
	c.wsSubsMu.RLock()
	defer c.wsSubsMu.RUnlock()
	for ch := range c.wsSubs {
		select {
		case ch <- event:
		default:
			// 通道满了，跳过
		}
	}
}

// EventCh 返回只读事件通道
func (c *Checker) EventCh() <-chan HealthEvent {
	return c.eventCh
}

// Start 启动检查器（不在此处启动定时器，由 Scheduler 调度调用 RunCheck）
func (c *Checker) Start() {
	log.Printf("[HealthChecker] Started")
}

// Stop 停止检查器，关闭通道
func (c *Checker) Stop() {
	close(c.stopCh)
	close(c.eventCh)
	log.Printf("[HealthChecker] Stopped")
}

// RunCheck 执行一轮完整检查
func (c *Checker) RunCheck() {
	// 获取所有 Host
	hosts, err := c.hostRepo.ListAll()
	if err != nil {
		log.Printf("[HealthChecker] Failed to list hosts: %v", err)
		return
	}

	for _, host := range hosts {
		select {
		case <-c.stopCh:
			return
		default:
		}

		c.checkHost(&host)
	}
}

// checkHost 检查单个 Host 的健康状态
func (c *Checker) checkHost(host *model.SSHHost) {
	checkedAt := time.Now().Unix()
	var tcpResult, sshResult CheckResult
	var tunnelResults []CheckResult

	// 1. TCP 检测
	tcpResult = TCPDetect(host.Host, host.Port, 5*time.Second)

	// 2. SSH 检测（如果 SSH Manager 中有对应 Client 且已连接）
	sshClient := c.sshManager.GetClient(host.ID)
	if sshClient != nil && sshClient.IsConnected() {
		sshResult = SSHDetect(host.Host, host.Port, sshClient.GetClient())
	} else {
		sshResult = CheckResult{Success: false, LatencyMs: 0}
	}

	// 3. Tunnel 检测（获取该 Host 上所有活跃 Rule）
	if sshClient != nil && sshClient.IsConnected() {
		// 获取所有转发组，然后获取每个组中的规则
		// 这里简化处理：通过 ruleRepo 查找所有活跃规则
		rules, err := c.ruleRepo.ListActive()
		if err != nil {
			log.Printf("[HealthChecker] Failed to list active rules: %v", err)
		} else {
			for _, rule := range rules {
				// 只检查当前 Host 承载的规则
				if rule.ActiveHostID == host.ID {
					result := TunnelDetect(sshClient.GetClient(), rule.TargetHost, rule.TargetPort, 5*time.Second)
					tunnelResults = append(tunnelResults, result)
				}
			}
		}
	}

	// 计算平均延迟
	var totalLatency float64
	var checkCount int
	if tcpResult.Success {
		totalLatency += tcpResult.LatencyMs
		checkCount++
	}
	if sshResult.Success {
		totalLatency += sshResult.LatencyMs
		checkCount++
	}
	for _, r := range tunnelResults {
		if r.Success {
			totalLatency += r.LatencyMs
			checkCount++
		}
	}

	var avgLatency float64
	if checkCount > 0 {
		avgLatency = totalLatency / float64(checkCount)
	}

	// 更新滑动窗口
	c.mu.Lock()
	// 综合检测结果：TCP 检测成功即可认为 Host 健康
	// SSH 连接和 Tunnel 检测用于转发功能，不影响 Host 健康状态
	overallSuccess := tcpResult.Success

	c.checkWindows[host.ID] = append(c.checkWindows[host.ID], windowEntry{
		result:    CheckResult{Success: overallSuccess, LatencyMs: avgLatency},
		timestamp: time.Now(),
	})
	c.mu.Unlock()

	// 计算健康度
	status, score := c.calculateHealth(host.ID)

	// 检查状态是否变化
	statusChanged := host.HealthStatus != status

	// 更新数据库
	if err := c.hostRepo.UpdateHealthStatus(host.ID, status, score, checkedAt); err != nil {
		log.Printf("[HealthChecker] Failed to update health status for host %d: %v", host.ID, err)
	}

	// 如果检测成功，更新最后成功时间
	if overallSuccess {
		if err := c.hostRepo.UpdateLastSuccess(host.ID, checkedAt); err != nil {
			log.Printf("[HealthChecker] Failed to update last success for host %d: %v", host.ID, err)
		}
	}

	// 记录历史
	c.RecordHistory(host.ID, score, status == "healthy", avgLatency)

	// 如果状态有变化，发送事件
	if statusChanged {
		event := HealthEvent{
			HostID:       host.ID,
			HealthStatus: status,
			HealthScore:  score,
			LatencyMs:    avgLatency,
			CheckedAt:    checkedAt,
		}
		select {
		case c.eventCh <- event:
		default:
			log.Printf("[HealthChecker] Event channel full, dropping event for host %d", host.ID)
		}
		// 广播到所有 WebSocket 订阅者
		c.broadcastToWS(event)
	}

	log.Printf("[HealthChecker] Host %d (%s@%s:%d) status=%s score=%.2f latency=%.2fms",
		host.ID, host.Username, host.Host, host.Port, status, score, avgLatency)
}

// calculateHealth 计算健康度
func (c *Checker) calculateHealth(hostID uint64) (string, float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries, exists := c.checkWindows[hostID]
	if !exists || len(entries) == 0 {
		return "unknown", 0
	}

	// 清除 5 分钟之前的窗口数据
	cutoff := time.Now().Add(-5 * time.Minute)
	var validEntries []windowEntry
	for _, entry := range entries {
		if entry.timestamp.After(cutoff) {
			validEntries = append(validEntries, entry)
		}
	}
	c.checkWindows[hostID] = validEntries

	if len(validEntries) == 0 {
		return "unknown", 0
	}

	// 急速失败：最近连续 3 次检测全部失败 → 立即 "unhealthy"
	if len(validEntries) >= 3 {
		recent3Failed := true
		for i := len(validEntries) - 3; i < len(validEntries); i++ {
			if validEntries[i].result.Success {
				recent3Failed = false
				break
			}
		}
		if recent3Failed {
			return "unhealthy", 0
		}
	}

	// 计算成功率
	var successCount int
	for _, entry := range validEntries {
		if entry.result.Success {
			successCount++
		}
	}

	score := (float64(successCount) / float64(len(validEntries))) * 100

	if score < 60 {
		return "unhealthy", score
	}
	return "healthy", score
}

// RecordHistory 写入 HealthHistory 表
func (c *Checker) RecordHistory(hostID uint64, score float64, isHealthy bool, latencyMs float64) {
	record := &model.HealthHistory{
		HostID:    hostID,
		Score:     score,
		IsHealthy: isHealthy,
		LatencyMs: latencyMs,
		CheckedAt: time.Now().Unix(),
	}

	if err := c.healthRepo.Create(record); err != nil {
		log.Printf("[HealthChecker] Failed to record health history for host %d: %v", hostID, err)
	}
}

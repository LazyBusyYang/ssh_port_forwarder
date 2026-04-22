package health

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"ssh-port-forwarder/internal/model"
	"ssh-port-forwarder/internal/pkg/metrics"
	"ssh-port-forwarder/internal/repository"
	"ssh-port-forwarder/internal/service/ssh_manager"
)

type Checker struct {
	mu         sync.RWMutex
	hostRepo   repository.SSHHostRepository
	healthRepo repository.HealthHistoryRepository
	ruleRepo   repository.ForwardRuleRepository
	sshManager *ssh_manager.Manager
	eventCh    chan HealthEvent // 状态变更事件通道
	stopCh     chan struct{}
	// 滑动窗口：hostID -> []CheckResult（最近 5 分钟）
	checkWindows map[uint64][]windowEntry
	// 最近一次 Rule 健康探测结果：ruleID -> result
	ruleHealth map[uint64]RuleHealthResult
	// WebSocket 订阅者管理
	wsSubsMu sync.RWMutex
	wsSubs   map[chan HealthEvent]struct{}
}

type windowEntry struct {
	result    CheckResult
	timestamp time.Time
}

type RuleHealthResult struct {
	RuleID         uint64 `json:"rule_id"`
	LocalPort      int    `json:"local_port"`
	Healthy        bool   `json:"healthy"`
	LocalReachable bool   `json:"local_reachable"`
	EndToEndOK     bool   `json:"end_to_end_ok"`
	FallbackUsed   bool   `json:"fallback_used"`
	Reason         string `json:"reason,omitempty"`
	CheckedAt      int64  `json:"checked_at"`
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
		ruleHealth:   make(map[uint64]RuleHealthResult),
		wsSubs:       make(map[chan HealthEvent]struct{}),
	}
}

func (c *Checker) GetRuleHealthSnapshot() map[uint64]RuleHealthResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[uint64]RuleHealthResult, len(c.ruleHealth))
	for k, v := range c.ruleHealth {
		result[k] = v
	}
	return result
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
	var sshResult CheckResult

	// Host 健康标准：SSH 连接可用（并辅以 SSH keepalive 检测）
	sshClient := c.sshManager.GetClient(host.ID)
	if sshClient == nil || !sshClient.IsConnected() {
		// 对未建立连接或断开的 Host，健康检查主动尝试建立 SSH 连接，
		// 避免“手动可连但系统一直显示 unhealthy”的状态错位。
		if err := c.sshManager.ConnectHost(host); err != nil {
			log.Printf("[HealthChecker] ConnectHost failed for host %d: %v", host.ID, err)
		}
		sshClient = c.sshManager.GetClient(host.ID)
	}
	if sshClient != nil && sshClient.IsConnected() {
		sshResult = SSHDetect(host.Host, host.Port, sshClient.GetClient())
	} else {
		sshResult = CheckResult{Success: false, LatencyMs: 0}
	}

	// Rule 健康标准：本地端口可达 + 端到端探测（失败时 TunnelDetect 兜底）
	if sshClient != nil && sshClient.IsConnected() {
		rules, err := c.ruleRepo.ListActive()
		if err != nil {
			log.Printf("[HealthChecker] Failed to list active rules: %v", err)
		} else {
			for _, rule := range rules {
				if rule.ActiveHostID == host.ID {
					ruleHealth := c.checkRuleHealth(&rule, sshClient.GetClient(), checkedAt)
					c.mu.Lock()
					c.ruleHealth[rule.ID] = ruleHealth
					c.mu.Unlock()

					rid := strconv.FormatUint(rule.ID, 10)
					gid := strconv.FormatUint(rule.GroupID, 10)
					rh := 0.0
					if ruleHealth.Healthy {
						rh = 1.0
					}
					metrics.SPFRuleHealth.WithLabelValues(rid, rule.Name, gid).Set(rh)

					log.Printf("[HealthChecker] Rule %d local=%t e2e=%t fallback=%t healthy=%t reason=%s",
						rule.ID,
						ruleHealth.LocalReachable,
						ruleHealth.EndToEndOK,
						ruleHealth.FallbackUsed,
						ruleHealth.Healthy,
						ruleHealth.Reason)
				}
			}
		}
	} else {
		// Host 未连接时，标记该 Host 下所有 active rule 为不健康，便于状态接口直观看到
		rules, err := c.ruleRepo.ListActive()
		if err == nil {
			for _, rule := range rules {
				if rule.ActiveHostID == host.ID {
					c.mu.Lock()
					c.ruleHealth[rule.ID] = RuleHealthResult{
						RuleID:         rule.ID,
						LocalPort:      rule.LocalPort,
						Healthy:        false,
						LocalReachable: false,
						EndToEndOK:     false,
						FallbackUsed:   false,
						Reason:         "ssh client not connected",
						CheckedAt:      checkedAt,
					}
					c.mu.Unlock()
					metrics.SPFRuleHealth.WithLabelValues(
						strconv.FormatUint(rule.ID, 10),
						rule.Name,
						strconv.FormatUint(rule.GroupID, 10),
					).Set(0)
				}
			}
		}
	}

	// Host 健康仅由 SSH 可用性决定
	c.mu.Lock()
	c.checkWindows[host.ID] = append(c.checkWindows[host.ID], windowEntry{
		result:    sshResult,
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

	hostIDStr := strconv.FormatUint(host.ID, 10)
	healthVal := 0.0
	if status == "healthy" {
		healthVal = 1.0
	}
	metrics.SPFHostHealth.WithLabelValues(hostIDStr, host.Name).Set(healthVal)
	if sshResult.Success {
		metrics.SPFHostLatency.WithLabelValues(hostIDStr, host.Name).Observe(sshResult.LatencyMs / 1000.0)
	}

	// 如果检测成功，更新最后成功时间
	if sshResult.Success {
		if err := c.hostRepo.UpdateLastSuccess(host.ID, checkedAt); err != nil {
			log.Printf("[HealthChecker] Failed to update last success for host %d: %v", host.ID, err)
		}
	}

	// 记录历史
	c.RecordHistory(host.ID, score, status == "healthy", sshResult.LatencyMs)

	// 如果状态有变化，发送事件
	if statusChanged {
		event := HealthEvent{
			HostID:       host.ID,
			HealthStatus: status,
			HealthScore:  score,
			LatencyMs:    sshResult.LatencyMs,
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
		host.ID, host.Username, host.Host, host.Port, status, score, sshResult.LatencyMs)
}

func (c *Checker) checkRuleHealth(rule *model.ForwardRule, sshClient *ssh.Client, checkedAt int64) RuleHealthResult {
	localResult := LocalForwardDetect(rule.LocalPort, 3*time.Second)
	if !localResult.Success {
		return RuleHealthResult{
			RuleID:         rule.ID,
			LocalPort:      rule.LocalPort,
			Healthy:        false,
			LocalReachable: false,
			EndToEndOK:     false,
			FallbackUsed:   false,
			Reason:         "local listener unreachable",
			CheckedAt:      checkedAt,
		}
	}

	proto := strings.ToLower(strings.TrimSpace(rule.Protocol))

	// 对 HTTP 服务使用合法 GET，避免写入裸字节导致 Uvicorn 等报 Invalid HTTP request。
	if proto == "http" {
		httpResult := EndToEndHTTPViaLocal(rule.LocalPort, 3*time.Second)
		if httpResult.Success {
			return RuleHealthResult{
				RuleID:         rule.ID,
				LocalPort:      rule.LocalPort,
				Healthy:        true,
				LocalReachable: true,
				EndToEndOK:     true,
				FallbackUsed:   false,
				Reason:         "",
				CheckedAt:      checkedAt,
			}
		}
		tunnelResult := TunnelDetect(sshClient, rule.TargetHost, rule.TargetPort, 3*time.Second)
		if tunnelResult.Success {
			return RuleHealthResult{
				RuleID:         rule.ID,
				LocalPort:      rule.LocalPort,
				Healthy:        true,
				LocalReachable: true,
				EndToEndOK:     true,
				FallbackUsed:   true,
				Reason:         "http probe via local failed; tunnel fallback passed",
				CheckedAt:      checkedAt,
			}
		}
		return RuleHealthResult{
			RuleID:         rule.ID,
			LocalPort:      rule.LocalPort,
			Healthy:        false,
			LocalReachable: true,
			EndToEndOK:     false,
			FallbackUsed:   true,
			Reason:         "http probe and tunnel fallback both failed",
			CheckedAt:      checkedAt,
		}
	}

	// tcp / https / 默认：不在本地端口写入应用层数据，仅用 SSH 隧道探测目标，避免污染 HTTP 等协议。
	tunnelResult := TunnelDetect(sshClient, rule.TargetHost, rule.TargetPort, 3*time.Second)
	if tunnelResult.Success {
		return RuleHealthResult{
			RuleID:         rule.ID,
			LocalPort:      rule.LocalPort,
			Healthy:        true,
			LocalReachable: true,
			EndToEndOK:     true,
			FallbackUsed:   false,
			Reason:         "",
			CheckedAt:      checkedAt,
		}
	}

	return RuleHealthResult{
		RuleID:         rule.ID,
		LocalPort:      rule.LocalPort,
		Healthy:        false,
		LocalReachable: true,
		EndToEndOK:     false,
		FallbackUsed:   false,
		Reason:         "tunnel unreachable",
		CheckedAt:      checkedAt,
	}
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

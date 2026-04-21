package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ssh-port-forwarder/internal/pkg/response"
	"ssh-port-forwarder/internal/service"
)

type StatusHandler struct {
	container *service.Container
}

func NewStatusHandler(c *service.Container) *StatusHandler {
	return &StatusHandler{container: c}
}

type OverviewResponse struct {
	TotalHosts        int `json:"total_hosts"`
	HealthyHosts      int `json:"healthy_hosts"`
	TotalRules        int `json:"total_rules"`
	ActiveRules       int `json:"active_rules"`
	ActiveConnections int `json:"active_connections"`
}

// Overview 返回系统总览统计
func (h *StatusHandler) Overview(c *gin.Context) {
	// 获取所有 Hosts
	hosts, _, err := h.container.HostRepo.List(1, 10000)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get hosts: "+err.Error())
		return
	}

	// 获取所有 Rules
	rules, _, err := h.container.RuleRepo.List(1, 10000)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get rules: "+err.Error())
		return
	}

	// 统计健康状态
	var healthyCount, unhealthyCount int
	for _, host := range hosts {
		switch host.HealthStatus {
		case "healthy":
			healthyCount++
		case "unhealthy":
			unhealthyCount++
		}
	}

	// 统计活跃规则
	var activeRuleCount int
	for _, rule := range rules {
		if rule.Status == "active" {
			activeRuleCount++
		}
	}

	// 获取活跃连接数（从 SSH Manager 获取）
	clients := h.container.SSHManager.GetAllClients()
	activeConnections := len(clients)

	overview := OverviewResponse{
		TotalHosts:        len(hosts),
		HealthyHosts:      healthyCount,
		TotalRules:        len(rules),
		ActiveRules:       activeRuleCount,
		ActiveConnections: activeConnections,
	}

	response.Success(c, overview)
}

// Hosts 返回所有 Host 状态列表
func (h *StatusHandler) Hosts(c *gin.Context) {
	hosts, _, err := h.container.HostRepo.List(1, 10000)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get hosts: "+err.Error())
		return
	}

	// 获取 SSH Manager 中的连接状态
	clients := h.container.SSHManager.GetAllClients()

	type HostStatus struct {
		ID            uint64  `json:"id"`
		Name          string  `json:"name"`
		Host          string  `json:"host"`
		Port          int     `json:"port"`
		Username      string  `json:"username"`
		HealthStatus  string  `json:"health_status"`
		HealthScore   float64 `json:"health_score"`
		Connected     bool    `json:"connected"`
		LastCheckAt   int64   `json:"last_check_at"`
		LastSuccessAt int64   `json:"last_success_at"`
	}

	var result []HostStatus
	for _, host := range hosts {
		client := clients[host.ID]
		connected := client != nil && client.IsConnected()

		result = append(result, HostStatus{
			ID:            host.ID,
			Name:          host.Name,
			Host:          host.Host,
			Port:          host.Port,
			Username:      host.Username,
			HealthStatus:  host.HealthStatus,
			HealthScore:   host.HealthScore,
			Connected:     connected,
			LastCheckAt:   host.LastCheckAt,
			LastSuccessAt: host.LastSuccessAt,
		})
	}

	response.Success(c, result)
}

// Rules 返回所有 Rule 状态列表
func (h *StatusHandler) Rules(c *gin.Context) {
	rules, _, err := h.container.RuleRepo.List(1, 10000)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get rules: "+err.Error())
		return
	}

	// 获取所有 Host 信息用于填充
	hosts, _, err := h.container.HostRepo.List(1, 10000)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get hosts: "+err.Error())
		return
	}

	hostMap := make(map[uint64]string)
	for _, host := range hosts {
		hostMap[host.ID] = host.Name
	}

	type RuleStatus struct {
		ID              uint64 `json:"id"`
		GroupID         uint64 `json:"group_id"`
		LocalPort       int    `json:"local_port"`
		TargetHost      string `json:"target_host"`
		TargetPort      int    `json:"target_port"`
		Protocol        string `json:"protocol"`
		Status          string `json:"status"`
		ActiveHostID    uint64 `json:"active_host_id"`
		ActiveHostName  string `json:"active_host_name"`
		HealthStatus    string `json:"health_status"`
		LocalReachable  bool   `json:"local_reachable"`
		EndToEndOK      bool   `json:"end_to_end_ok"`
		FallbackUsed    bool   `json:"fallback_used"`
		HealthReason    string `json:"health_reason,omitempty"`
		HealthCheckedAt int64  `json:"health_checked_at,omitempty"`
	}

	ruleHealthMap := h.container.HealthChecker.GetRuleHealthSnapshot()

	var result []RuleStatus
	for _, rule := range rules {
		activeHostName := ""
		if rule.ActiveHostID > 0 {
			activeHostName = hostMap[rule.ActiveHostID]
		}
		ruleHealth, ok := ruleHealthMap[rule.ID]
		healthStatus := "unknown"
		if ok {
			if ruleHealth.Healthy {
				healthStatus = "healthy"
			} else {
				healthStatus = "unhealthy"
			}
		}

		result = append(result, RuleStatus{
			ID:              rule.ID,
			GroupID:         rule.GroupID,
			LocalPort:       rule.LocalPort,
			TargetHost:      rule.TargetHost,
			TargetPort:      rule.TargetPort,
			Protocol:        rule.Protocol,
			Status:          rule.Status,
			ActiveHostID:    rule.ActiveHostID,
			ActiveHostName:  activeHostName,
			HealthStatus:    healthStatus,
			LocalReachable:  ruleHealth.LocalReachable,
			EndToEndOK:      ruleHealth.EndToEndOK,
			FallbackUsed:    ruleHealth.FallbackUsed,
			HealthReason:    ruleHealth.Reason,
			HealthCheckedAt: ruleHealth.CheckedAt,
		})
	}

	response.Success(c, result)
}

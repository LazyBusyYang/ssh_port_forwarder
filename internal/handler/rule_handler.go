package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ssh-port-forwarder/internal/model"
	"ssh-port-forwarder/internal/pkg/response"
	"ssh-port-forwarder/internal/pkg/validator"
	"ssh-port-forwarder/internal/service"
)

type RuleHandler struct {
	container *service.Container
}

func NewRuleHandler(c *service.Container) *RuleHandler {
	return &RuleHandler{container: c}
}

type CreateRuleRequest struct {
	Name       string `json:"name" binding:"required"`
	GroupID    uint64 `json:"group_id" binding:"required"`
	LocalPort  int    `json:"local_port" binding:"required,min=1,max=65535"`
	TargetHost string `json:"target_host" binding:"required"`
	TargetPort int    `json:"target_port" binding:"required,min=1,max=65535"`
	Protocol   string `json:"protocol"`
}

type UpdateRuleRequest struct {
	Name       string `json:"name"`
	GroupID    uint64 `json:"group_id"`
	LocalPort  int    `json:"local_port"`
	TargetHost string `json:"target_host"`
	TargetPort int    `json:"target_port"`
	Protocol   string `json:"protocol"`
}

// List 分页查询转发规则列表
func (h *RuleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	rules, total, err := h.container.RuleRepo.List(page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to list rules: "+err.Error())
		return
	}

	response.Paged(c, rules, total, page, pageSize)
}

// Create 创建转发规则
func (h *RuleHandler) Create(c *gin.Context) {
	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid request: "+err.Error())
		return
	}

	// 校验 LocalPort 范围
	portRange := h.container.Config.PortRange
	if portRange.Min == 0 {
		portRange.Min = 30000
	}
	if portRange.Max == 0 {
		portRange.Max = 33000
	}
	if err := validator.ValidatePortRange(req.LocalPort, portRange.Min, portRange.Max); err != nil {
		response.Error(c, http.StatusBadRequest, 400, err.Error())
		return
	}

	// 校验 LocalPort 冲突
	existingRule, err := h.container.RuleRepo.FindByLocalPort(req.LocalPort)
	if err == nil && existingRule != nil {
		response.Error(c, http.StatusBadRequest, 400, "local port already in use")
		return
	}

	// 设置默认协议
	protocol := req.Protocol
	if protocol == "" {
		protocol = "tcp"
	}

	rule := &model.ForwardRule{
		Name:       req.Name,
		GroupID:    req.GroupID,
		LocalPort:  req.LocalPort,
		TargetHost: req.TargetHost,
		TargetPort: req.TargetPort,
		Protocol:   protocol,
		Status:     "inactive",
	}

	if err := h.container.RuleRepo.Create(rule); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to create rule: "+err.Error())
		return
	}

	// 通过 LB Pool 分配 Host 并启动转发
	host, err := h.container.LBPool.AssignHostForRule(rule)
	if err != nil {
		// 没有可用的 Host，保持 inactive 状态
		log.Printf("[RuleHandler] Failed to assign host for rule %d: %v", rule.ID, err)
		response.Success(c, rule)
		return
	}

	// 连接到 Host
	if err := h.container.SSHManager.ConnectHost(host); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to connect host: "+err.Error())
		return
	}

	// 启动转发
	if err := h.container.SSHManager.StartForwardRule(rule, host.ID); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to start forward: "+err.Error())
		return
	}

	// 更新 Rule 状态
	if err := h.container.RuleRepo.UpdateActiveHost(rule.ID, host.ID); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to update rule: "+err.Error())
		return
	}
	if err := h.container.RuleRepo.UpdateStatus(rule.ID, "active"); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to update rule status: "+err.Error())
		return
	}

	rule.Status = "active"
	rule.ActiveHostID = host.ID

	response.Success(c, rule)
}

// Get 获取单个转发规则
func (h *RuleHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid id")
		return
	}

	rule, err := h.container.RuleRepo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get rule: "+err.Error())
		return
	}
	if rule == nil {
		response.Error(c, http.StatusNotFound, 404, "rule not found")
		return
	}

	response.Success(c, rule)
}

// Update 更新转发规则
func (h *RuleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid id")
		return
	}

	var req UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid request: "+err.Error())
		return
	}

	rule, err := h.container.RuleRepo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get rule: "+err.Error())
		return
	}
	if rule == nil {
		response.Error(c, http.StatusNotFound, 404, "rule not found")
		return
	}

	// 如果更新 LocalPort，需要校验范围和冲突
	if req.LocalPort > 0 && req.LocalPort != rule.LocalPort {
		portRange := h.container.Config.PortRange
		if portRange.Min == 0 {
			portRange.Min = 30000
		}
		if portRange.Max == 0 {
			portRange.Max = 33000
		}
		if err := validator.ValidatePortRange(req.LocalPort, portRange.Min, portRange.Max); err != nil {
			response.Error(c, http.StatusBadRequest, 400, err.Error())
			return
		}

		existingRule, err := h.container.RuleRepo.FindByLocalPort(req.LocalPort)
		if err == nil && existingRule != nil && existingRule.ID != rule.ID {
			response.Error(c, http.StatusBadRequest, 400, "local port already in use")
			return
		}

		rule.LocalPort = req.LocalPort
	}

	// 更新其他字段
	if req.Name != "" {
		rule.Name = req.Name
	}
	if req.GroupID > 0 {
		rule.GroupID = req.GroupID
	}
	if req.TargetHost != "" {
		rule.TargetHost = req.TargetHost
	}
	if req.TargetPort > 0 {
		rule.TargetPort = req.TargetPort
	}
	if req.Protocol != "" {
		rule.Protocol = req.Protocol
	}

	if err := h.container.RuleRepo.Update(rule); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to update rule: "+err.Error())
		return
	}

	response.Success(c, rule)
}

// Delete 删除转发规则
func (h *RuleHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid id")
		return
	}

	rule, err := h.container.RuleRepo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get rule: "+err.Error())
		return
	}
	if rule == nil {
		response.Error(c, http.StatusNotFound, 404, "rule not found")
		return
	}

	// 如果规则处于 active 状态，先停止转发
	if rule.Status == "active" && rule.ActiveHostID > 0 {
		if err := h.container.SSHManager.StopForwardRule(rule.ID, rule.ActiveHostID); err != nil {
			// 记录错误但继续删除
		}
	}

	if err := h.container.RuleRepo.Delete(id); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to delete rule: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "rule deleted"})
}

// Restart 重启转发规则
func (h *RuleHandler) Restart(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid id")
		return
	}

	rule, err := h.container.RuleRepo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get rule: "+err.Error())
		return
	}
	if rule == nil {
		response.Error(c, http.StatusNotFound, 404, "rule not found")
		return
	}

	// 停止旧的转发
	if rule.ActiveHostID > 0 {
		if err := h.container.SSHManager.StopForwardRule(rule.ID, rule.ActiveHostID); err != nil {
			// 记录错误但继续
		}
	}

	// 重新分配 Host 并启动转发
	host, err := h.container.LBPool.AssignHostForRule(rule)
	if err != nil {
		// 没有可用的 Host，标记为 inactive
		log.Printf("[RuleHandler] Failed to assign host for rule %d restart: %v", rule.ID, err)
		if err := h.container.RuleRepo.UpdateStatus(rule.ID, "inactive"); err != nil {
			response.Error(c, http.StatusInternalServerError, 500, "failed to update rule status: "+err.Error())
			return
		}
		if err := h.container.RuleRepo.UpdateActiveHost(rule.ID, 0); err != nil {
			response.Error(c, http.StatusInternalServerError, 500, "failed to update rule: "+err.Error())
			return
		}
		rule.Status = "inactive"
		rule.ActiveHostID = 0
		response.Success(c, rule)
		return
	}

	// 连接到 Host
	if err := h.container.SSHManager.ConnectHost(host); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to connect host: "+err.Error())
		return
	}

	// 启动转发
	if err := h.container.SSHManager.StartForwardRule(rule, host.ID); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to start forward: "+err.Error())
		return
	}

	// 更新 Rule 状态
	if err := h.container.RuleRepo.UpdateActiveHost(rule.ID, host.ID); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to update rule: "+err.Error())
		return
	}
	if err := h.container.RuleRepo.UpdateStatus(rule.ID, "active"); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to update rule status: "+err.Error())
		return
	}

	rule.Status = "active"
	rule.ActiveHostID = host.ID

	response.Success(c, rule)
}

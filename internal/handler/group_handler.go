package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ssh-port-forwarder/internal/model"
	"ssh-port-forwarder/internal/pkg/metrics"
	"ssh-port-forwarder/internal/pkg/response"
	"ssh-port-forwarder/internal/pkg/validator"
	"ssh-port-forwarder/internal/service"
)

type GroupHandler struct {
	container *service.Container
}

func NewGroupHandler(c *service.Container) *GroupHandler {
	return &GroupHandler{container: c}
}

type CreateGroupRequest struct {
	Name     string `json:"name" binding:"required"`
	Strategy string `json:"strategy"`
}

type UpdateGroupRequest struct {
	Name     string `json:"name"`
	Strategy string `json:"strategy"`
}

type AddHostRequest struct {
	HostID uint64 `json:"host_id" binding:"required"`
}

// GroupListItem 包含计数信息的转发组列表项
type GroupListItem struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	Strategy  string `json:"strategy"`
	HostCount int64  `json:"host_count"`
	RuleCount int64  `json:"rule_count"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// List 分页查询转发组列表（带 host_count / rule_count）
func (h *GroupHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	groups, total, err := h.container.GroupRepo.List(page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to list groups: "+err.Error())
		return
	}

	// 组装带计数的列表项
	items := make([]GroupListItem, 0, len(groups))
	for _, g := range groups {
		hostCount, _ := h.container.GroupRepo.CountHosts(g.ID)
		ruleCount, _ := h.container.RuleRepo.CountByGroupID(g.ID)
		items = append(items, GroupListItem{
			ID:        g.ID,
			Name:      g.Name,
			Strategy:  g.Strategy,
			HostCount: hostCount,
			RuleCount: ruleCount,
			CreatedAt: g.CreatedAt,
			UpdatedAt: g.UpdatedAt,
		})
	}

	response.Paged(c, items, total, page, pageSize)
}

// Create 创建转发组
func (h *GroupHandler) Create(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid request: "+err.Error())
		return
	}

	// 设置默认策略
	strategy := req.Strategy
	if strategy == "" {
		strategy = "round_robin"
	}

	// 校验策略
	if err := validator.ValidateStrategy(strategy); err != nil {
		response.Error(c, http.StatusBadRequest, 400, err.Error())
		return
	}

	// F3: 检查名称唯一性
	exists, err := h.container.GroupRepo.ExistsByName(req.Name)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to check name: "+err.Error())
		return
	}
	if exists {
		response.Error(c, http.StatusConflict, 409, "Group 名称已存在，请使用其他名称")
		return
	}

	group := &model.ForwardGroup{
		Name:     req.Name,
		Strategy: strategy,
	}

	if err := h.container.GroupRepo.Create(group); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to create group: "+err.Error())
		return
	}

	response.Success(c, group)
}

// Get 获取单个转发组（带 Hosts）
func (h *GroupHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid id")
		return
	}

	group, err := h.container.GroupRepo.FindByIDWithHosts(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get group: "+err.Error())
		return
	}
	if group == nil {
		response.Error(c, http.StatusNotFound, 404, "group not found")
		return
	}

	response.Success(c, group)
}

// Update 更新转发组
func (h *GroupHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid id")
		return
	}

	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid request: "+err.Error())
		return
	}

	group, err := h.container.GroupRepo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get group: "+err.Error())
		return
	}
	if group == nil {
		response.Error(c, http.StatusNotFound, 404, "group not found")
		return
	}

	// 更新字段
	if req.Name != "" {
		// F3: 检查名称唯一性（排除自身）
		if req.Name != group.Name {
			existing, err := h.container.GroupRepo.FindByName(req.Name)
			if err != nil {
				response.Error(c, http.StatusInternalServerError, 500, "failed to check name: "+err.Error())
				return
			}
			if existing != nil && existing.ID != id {
				response.Error(c, http.StatusConflict, 409, "Group 名称已存在，请使用其他名称")
				return
			}
		}
		group.Name = req.Name
	}
	if req.Strategy != "" {
		if err := validator.ValidateStrategy(req.Strategy); err != nil {
			response.Error(c, http.StatusBadRequest, 400, err.Error())
			return
		}
		group.Strategy = req.Strategy
	}

	if err := h.container.GroupRepo.Update(group); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to update group: "+err.Error())
		return
	}

	response.Success(c, group)
}

// Delete 删除转发组
func (h *GroupHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid id")
		return
	}

	// Step 1: 检查是否有 Rule 引用该 Group（Option A 严格模式）
	ruleCount, err := h.container.RuleRepo.CountByGroupID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to check rule references: "+err.Error())
		return
	}
	if ruleCount > 0 {
		// 返回 HTTP 409，包含关联 Rule 列表
		rules, err := h.container.RuleRepo.ListByGroupID(id)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 500, "failed to list referencing rules: "+err.Error())
			return
		}
		ruleNames := make([]string, 0, len(rules))
		for _, r := range rules {
			ruleNames = append(ruleNames, r.Name)
		}
		response.ErrorWithData(c, http.StatusConflict, 409,
			fmt.Sprintf("该 Group 正被 %d 条 Rule 引用，删除后将导致转发规则失效", ruleCount),
			gin.H{"referencing_rules": ruleNames})
		return
	}

	// Step 2: 获取 Group 信息用于 metrics 清理
	group, err := h.container.GroupRepo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to find group: "+err.Error())
		return
	}
	if group == nil {
		response.Error(c, http.StatusNotFound, 404, "group not found")
		return
	}

	// Step 3: 清理 metrics — Host 相关
	hosts, err := h.container.GroupRepo.GetHosts(id)
	if err == nil {
		for _, host := range hosts {
			metrics.SPFHostGroupInfo.DeleteLabelValues(
				strconv.FormatUint(host.ID, 10),
				host.Name,
				strconv.FormatUint(id, 10),
				group.Name,
			)
		}
	}

	// Step 4: 清理 metrics — Rule 相关（虽然上面已检查无 Rule，为防 race condition 仍做清理）
	rules, err := h.container.GroupRepo.GetRules(id)
	if err == nil {
		for _, rule := range rules {
			metrics.CleanupRule(
				strconv.FormatUint(rule.ID, 10),
				rule.Name,
				strconv.FormatUint(id, 10),
			)
		}
	}

	// Step 5: 删除 Group
	if err := h.container.GroupRepo.Delete(id); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to delete group: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "group deleted"})
}

// AddHost 添加 Host 到 Group
func (h *GroupHandler) AddHost(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid group id")
		return
	}

	var req AddHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid request: "+err.Error())
		return
	}

	// 检查 Group 是否存在
	group, err := h.container.GroupRepo.FindByID(groupID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get group: "+err.Error())
		return
	}
	if group == nil {
		response.Error(c, http.StatusNotFound, 404, "group not found")
		return
	}

	// 检查 Host 是否存在
	host, err := h.container.HostRepo.FindByID(req.HostID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get host: "+err.Error())
		return
	}
	if host == nil {
		response.Error(c, http.StatusNotFound, 404, "host not found")
		return
	}

	if err := h.container.GroupRepo.AddHost(groupID, req.HostID); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to add host: "+err.Error())
		return
	}

	metrics.SPFHostGroupInfo.WithLabelValues(
		strconv.FormatUint(req.HostID, 10),
		host.Name,
		strconv.FormatUint(groupID, 10),
		group.Name,
	).Set(1)

	response.Success(c, gin.H{"message": "host added to group"})
}

// RemoveHost 从 Group 移除 Host
func (h *GroupHandler) RemoveHost(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid group id")
		return
	}

	hostID, err := strconv.ParseUint(c.Param("host_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid host id")
		return
	}

	group, err := h.container.GroupRepo.FindByID(groupID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get group: "+err.Error())
		return
	}
	if group == nil {
		response.Error(c, http.StatusNotFound, 404, "group not found")
		return
	}
	host, err := h.container.HostRepo.FindByID(hostID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get host: "+err.Error())
		return
	}
	if host == nil {
		response.Error(c, http.StatusNotFound, 404, "host not found")
		return
	}

	if err := h.container.GroupRepo.RemoveHost(groupID, hostID); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to remove host: "+err.Error())
		return
	}

	metrics.SPFHostGroupInfo.DeleteLabelValues(
		strconv.FormatUint(hostID, 10),
		host.Name,
		strconv.FormatUint(groupID, 10),
		group.Name,
	)

	response.Success(c, gin.H{"message": "host removed from group"})
}

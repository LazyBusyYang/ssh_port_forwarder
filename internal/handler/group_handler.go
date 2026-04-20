package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ssh-port-forwarder/internal/model"
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

// List 分页查询转发组列表
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

	response.Paged(c, groups, total, page, pageSize)
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

	if err := h.container.GroupRepo.RemoveHost(groupID, hostID); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to remove host: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "host removed from group"})
}

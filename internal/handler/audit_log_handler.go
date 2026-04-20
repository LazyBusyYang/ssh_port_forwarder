package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ssh-port-forwarder/internal/pkg/response"
	"ssh-port-forwarder/internal/service"
)

type AuditLogHandler struct {
	container *service.Container
}

func NewAuditLogHandler(c *service.Container) *AuditLogHandler {
	return &AuditLogHandler{container: c}
}

// List 分页查询审计日志
func (h *AuditLogHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 解析可选的过滤参数
	action := c.Query("action")
	userID, _ := strconv.ParseUint(c.Query("user_id"), 10, 64)

	logs, total, err := h.container.AuditRepo.List(page, pageSize, action, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to list audit logs: "+err.Error())
		return
	}

	response.Paged(c, logs, total, page, pageSize)
}

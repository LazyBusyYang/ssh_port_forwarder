package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ssh-port-forwarder/internal/pkg/response"
	"ssh-port-forwarder/internal/service"
)

type HealthHandler struct {
	container *service.Container
}

func NewHealthHandler(c *service.Container) *HealthHandler {
	return &HealthHandler{container: c}
}

// GetHistory 查询指定 host_id 的健康度历史
func (h *HealthHandler) GetHistory(c *gin.Context) {
	hostID, err := strconv.ParseUint(c.Param("host_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid host_id")
		return
	}

	// 解析时间范围
	startTime, _ := strconv.ParseInt(c.Query("start"), 10, 64)
	endTime, _ := strconv.ParseInt(c.Query("end"), 10, 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit < 1 || limit > 1000 {
		limit = 100
	}

	// 如果未指定时间范围，默认查询最近24小时
	if endTime == 0 {
		endTime = 0 // 0 表示不限制结束时间
	}

	history, err := h.container.HealthRepo.ListByHostID(hostID, startTime, endTime, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get health history: "+err.Error())
		return
	}

	response.Success(c, history)
}

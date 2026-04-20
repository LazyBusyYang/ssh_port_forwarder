package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"ssh-port-forwarder/internal/model"
	"ssh-port-forwarder/internal/repository"
)

// AuditMiddleware 记录写操作（POST/PUT/DELETE）到 DB + stdout
func AuditMiddleware(auditRepo repository.AuditLogRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只记录写操作
		if c.Request.Method == "GET" {
			c.Next()
			return
		}

		c.Next()

		// 请求完成后记录日志
		userID, _ := c.Get("userID")
		uid, ok := userID.(uint64)
		if !ok {
			uid = 0
		}

		auditLog := &model.AuditLog{
			UserID:     uid,
			Action:     fmt.Sprintf("%s %s", c.Request.Method, c.FullPath()),
			TargetType: inferTargetType(c.FullPath()),
			TargetID:   0, // 可从路径参数提取
			Detail:     fmt.Sprintf(`{"status":%d,"method":"%s","path":"%s"}`, c.Writer.Status(), c.Request.Method, c.Request.URL.Path),
			CreatedAt:  time.Now().Unix(),
		}

		// 写入 DB
		if err := auditRepo.Create(auditLog); err != nil {
			log.Printf("[AUDIT] failed to save audit log: %v", err)
		}

		// 结构化 JSON 输出到 stdout
		jsonLog, _ := json.Marshal(map[string]interface{}{
			"type":        "audit",
			"user_id":     uid,
			"action":      auditLog.Action,
			"target_type": auditLog.TargetType,
			"status":      c.Writer.Status(),
			"path":        c.Request.URL.Path,
			"timestamp":   time.Now().Unix(),
		})
		log.Printf("[AUDIT] %s", string(jsonLog))
	}
}

// inferTargetType 从路径推断目标类型
func inferTargetType(path string) string {
	// 根据路径包含的关键字推断
	switch {
	case strings.Contains(path, "hosts"):
		return "ssh_host"
	case strings.Contains(path, "rules"):
		return "forward_rule"
	case strings.Contains(path, "groups"):
		return "forward_group"
	case strings.Contains(path, "auth"):
		return "auth"
	default:
		return "unknown"
	}
}

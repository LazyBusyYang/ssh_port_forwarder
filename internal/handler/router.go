package handler

import (
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"ssh-port-forwarder/internal/middleware"
	"ssh-port-forwarder/internal/service"
	"ssh-port-forwarder/web"
)

func SetupRouter(container *service.Container) *gin.Engine {
	r := gin.New()

	// 全局中间件
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.CORSMiddleware())

	// Prometheus metrics
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1
	v1 := r.Group("/api/v1")

	// 公开路由（无需认证）
	auth := v1.Group("/auth")
	{
		authHandler := NewAuthHandler(container)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
	}

	// 需要认证的路由
	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware(container.AuthService))
	protected.Use(middleware.AuditMiddleware(container.AuditRepo))
	{
		// Auth
		authHandler := NewAuthHandler(container)
		protected.POST("/auth/logout", authHandler.Logout)

		// SSH Hosts (admin only)
		hostHandler := NewHostHandler(container)
		hosts := protected.Group("/hosts")
		hosts.Use(middleware.AdminRequired())
		{
			hosts.GET("", hostHandler.List)
			hosts.POST("", hostHandler.Create)
			hosts.POST("/:id/copy", hostHandler.Copy)
			hosts.GET("/:id", hostHandler.Get)
			hosts.PUT("/:id", hostHandler.Update)
			hosts.DELETE("/:id", hostHandler.Delete)
			hosts.POST("/:id/test", hostHandler.Test)
		}

		// Forward Groups (admin only)
		groupHandler := NewGroupHandler(container)
		groups := protected.Group("/groups")
		groups.Use(middleware.AdminRequired())
		{
			groups.GET("", groupHandler.List)
			groups.POST("", groupHandler.Create)
			groups.GET("/:id", groupHandler.Get)
			groups.PUT("/:id", groupHandler.Update)
			groups.DELETE("/:id", groupHandler.Delete)
			groups.POST("/:id/hosts", groupHandler.AddHost)
			groups.DELETE("/:id/hosts/:host_id", groupHandler.RemoveHost)
		}

		// Forward Rules (admin only)
		ruleHandler := NewRuleHandler(container)
		rules := protected.Group("/rules")
		rules.Use(middleware.AdminRequired())
		{
			rules.GET("", ruleHandler.List)
			rules.POST("", ruleHandler.Create)
			rules.GET("/:id", ruleHandler.Get)
			rules.PUT("/:id", ruleHandler.Update)
			rules.DELETE("/:id", ruleHandler.Delete)
			rules.POST("/:id/restart", ruleHandler.Restart)
		}

		// Status
		statusHandler := NewStatusHandler(container)
		protected.GET("/status/overview", statusHandler.Overview)
		protected.GET("/status/hosts", statusHandler.Hosts)
		protected.GET("/status/rules", statusHandler.Rules)

		// Health History
		healthHandler := NewHealthHandler(container)
		protected.GET("/health-history/:host_id", healthHandler.GetHistory)

		// Audit Logs
		protected.GET("/audit-logs", NewAuditLogHandler(container).List)

		// WebSocket
		wsHandler := NewWSHandler(container)
		protected.GET("/ws/status", wsHandler.Status)
	}

	// 静态文件服务（前端 SPA）
	distFS, err := fs.Sub(web.StaticFS, "dist")
	if err == nil {
		// 静态资源文件服务
		r.StaticFS("/assets", http.FS(distFS))

		// SPA fallback：所有未匹配的路径返回 index.html
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path

			// API 请求返回 404
			if strings.HasPrefix(path, "/api") || path == "/metrics" {
				c.JSON(404, gin.H{"code": 404, "message": "not found"})
				return
			}

			// 尝试从 dist 中读取文件
			filePath := strings.TrimPrefix(path, "/")
			data, err := web.StaticFS.ReadFile("dist/" + filePath)
			if err == nil {
				// 根据扩展名设置 Content-Type
				contentType := mime.TypeByExtension(filepath.Ext(path))
				if contentType == "" {
					contentType = "application/octet-stream"
				}
				c.Data(200, contentType, data)
				return
			}

			// fallback 到 index.html
			indexHTML, err := web.StaticFS.ReadFile("dist/index.html")
			if err != nil {
				c.String(500, "internal error")
				return
			}
			c.Data(200, "text/html; charset=utf-8", indexHTML)
		})
	}

	return r
}

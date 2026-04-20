package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"ssh-port-forwarder/internal/service"
)

// AuthMiddleware JWT 校验中间件
func AuthMiddleware(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// 优先从 Header 获取
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		// Header 中没有则尝试 query 参数（WebSocket 场景）
		if tokenString == "" {
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "missing authorization"})
			c.Abort()
			return
		}

		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid or expired token"})
			c.Abort()
			return
		}

		// 注入用户信息到 context
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// AdminRequired RBAC 拦截：非 admin 角色返回 403
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role.(string) != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

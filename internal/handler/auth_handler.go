package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ssh-port-forwarder/internal/pkg/response"
	"ssh-port-forwarder/internal/service"
)

type AuthHandler struct {
	container *service.Container
}

func NewAuthHandler(c *service.Container) *AuthHandler {
	return &AuthHandler{container: c}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid request: "+err.Error())
		return
	}

	tokenPair, err := h.container.AuthService.Login(req.Username, req.Password)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, 401, err.Error())
		return
	}

	response.Success(c, tokenPair)
}

// Refresh 刷新 Token
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid request: "+err.Error())
		return
	}

	tokenPair, err := h.container.AuthService.RefreshToken(req.RefreshToken)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, 401, err.Error())
		return
	}

	response.Success(c, tokenPair)
}

// Logout 用户登出
func (h *AuthHandler) Logout(c *gin.Context) {
	// 目前简单实现，返回成功
	// 实际实现可以加入黑名单等机制
	response.Success(c, gin.H{"message": "logged out"})
}

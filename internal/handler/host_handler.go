package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/ssh"

	"ssh-port-forwarder/internal/model"
	"ssh-port-forwarder/internal/pkg/crypto"
	"ssh-port-forwarder/internal/pkg/response"
	"ssh-port-forwarder/internal/pkg/validator"
	"ssh-port-forwarder/internal/service"
)

type HostHandler struct {
	container *service.Container
}

func NewHostHandler(c *service.Container) *HostHandler {
	return &HostHandler{container: c}
}

type CreateHostRequest struct {
	Name       string `json:"name" binding:"required"`
	Host       string `json:"host" binding:"required"`
	Port       int    `json:"port" binding:"required,min=1,max=65535"`
	Username   string `json:"username" binding:"required"`
	AuthMethod string `json:"auth_method" binding:"required"`
	AuthData   string `json:"auth_data" binding:"required"`
	Weight     int    `json:"weight" binding:"min=1,max=100"`
}

type UpdateHostRequest struct {
	Name       string `json:"name"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	AuthMethod string `json:"auth_method"`
	AuthData   string `json:"auth_data"`
	Weight     int    `json:"weight"`
}

// List 分页查询 SSH Host 列表
func (h *HostHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	hosts, total, err := h.container.HostRepo.List(page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to list hosts: "+err.Error())
		return
	}

	response.Paged(c, hosts, total, page, pageSize)
}

// Create 创建 SSH Host
func (h *HostHandler) Create(c *gin.Context) {
	var req CreateHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid request: "+err.Error())
		return
	}

	// 校验认证方式
	if err := validator.ValidateAuthMethod(req.AuthMethod); err != nil {
		response.Error(c, http.StatusBadRequest, 400, err.Error())
		return
	}

	// 加密 auth_data
	encryptedData, nonce, err := crypto.Encrypt(req.AuthData, h.container.Config.Encryption.Key)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to encrypt auth data: "+err.Error())
		return
	}

	// 设置默认权重
	weight := req.Weight
	if weight == 0 {
		weight = 100
	}

	host := &model.SSHHost{
		Name:       req.Name,
		Host:       req.Host,
		Port:       req.Port,
		Username:   req.Username,
		AuthMethod: req.AuthMethod,
		AuthData:   encryptedData,
		AuthNonce:  nonce,
		Weight:     weight,
	}

	if err := h.container.HostRepo.Create(host); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to create host: "+err.Error())
		return
	}

	response.Success(c, host)
}

// Get 获取单个 SSH Host
func (h *HostHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid id")
		return
	}

	host, err := h.container.HostRepo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get host: "+err.Error())
		return
	}
	if host == nil {
		response.Error(c, http.StatusNotFound, 404, "host not found")
		return
	}

	response.Success(c, host)
}

// Update 更新 SSH Host
func (h *HostHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid id")
		return
	}

	var req UpdateHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid request: "+err.Error())
		return
	}

	host, err := h.container.HostRepo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get host: "+err.Error())
		return
	}
	if host == nil {
		response.Error(c, http.StatusNotFound, 404, "host not found")
		return
	}

	// 更新字段
	if req.Name != "" {
		host.Name = req.Name
	}
	if req.Host != "" {
		host.Host = req.Host
	}
	if req.Port > 0 {
		host.Port = req.Port
	}
	if req.Username != "" {
		host.Username = req.Username
	}
	if req.AuthMethod != "" {
		if err := validator.ValidateAuthMethod(req.AuthMethod); err != nil {
			response.Error(c, http.StatusBadRequest, 400, err.Error())
			return
		}
		host.AuthMethod = req.AuthMethod
	}
	if req.AuthData != "" {
		encryptedData, nonce, err := crypto.Encrypt(req.AuthData, h.container.Config.Encryption.Key)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 500, "failed to encrypt auth data: "+err.Error())
			return
		}
		host.AuthData = encryptedData
		host.AuthNonce = nonce
	}
	if req.Weight > 0 {
		host.Weight = req.Weight
	}

	if err := h.container.HostRepo.Update(host); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to update host: "+err.Error())
		return
	}

	response.Success(c, host)
}

// Delete 删除 SSH Host（软删除）
func (h *HostHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid id")
		return
	}

	if err := h.container.HostRepo.Delete(id); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to delete host: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "host deleted"})
}

// Test 测试 SSH 连接
func (h *HostHandler) Test(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid id")
		return
	}

	host, err := h.container.HostRepo.FindByID(id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get host: "+err.Error())
		return
	}
	if host == nil {
		response.Error(c, http.StatusNotFound, 404, "host not found")
		return
	}

	// 解密 auth_data
	authData, err := crypto.DecryptWithFallback(
		host.AuthData,
		host.AuthNonce,
		h.container.Config.Encryption.Key,
		h.container.Config.Encryption.KeyPrevious,
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to decrypt auth data: "+err.Error())
		return
	}

	// 构建 SSH 配置
	var authMethod ssh.AuthMethod
	switch host.AuthMethod {
	case "password":
		authMethod = ssh.Password(authData)
	case "private_key":
		signer, err := ssh.ParsePrivateKey([]byte(authData))
		if err != nil {
			response.Error(c, http.StatusBadRequest, 400, "invalid private key: "+err.Error())
			return
		}
		authMethod = ssh.PublicKeys(signer)
	default:
		response.Error(c, http.StatusBadRequest, 400, "unsupported auth method: "+host.AuthMethod)
		return
	}

	config := &ssh.ClientConfig{
		User:            host.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         0,
	}

	// 尝试连接
	addr := fmt.Sprintf("%s:%d", host.Host, host.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "connection failed: "+err.Error())
		return
	}
	defer client.Close()

	response.Success(c, gin.H{"message": "connection successful"})
}

package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/ssh"

	"ssh-port-forwarder/internal/model"
	"ssh-port-forwarder/internal/pkg/crypto"
	"ssh-port-forwarder/internal/pkg/metrics"
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

// HostResponse 返回给前端的 Host，不含认证密文。
type HostResponse struct {
	ID            uint64  `json:"id"`
	Name          string  `json:"name"`
	Host          string  `json:"host"`
	Port          int     `json:"port"`
	Username      string  `json:"username"`
	AuthMethod    string  `json:"auth_method"`
	Weight        int     `json:"weight"`
	HealthStatus  string  `json:"health_status"`
	HealthScore   float64 `json:"health_score"`
	LastCheckAt   int64   `json:"last_check_at"`
	LastSuccessAt int64   `json:"last_success_at"`
	CreatedAt     int64   `json:"created_at"`
	UpdatedAt     int64   `json:"updated_at"`
}

func toHostResponse(h *model.SSHHost) HostResponse {
	if h == nil {
		return HostResponse{}
	}
	return HostResponse{
		ID:            h.ID,
		Name:          h.Name,
		Host:          h.Host,
		Port:          h.Port,
		Username:      h.Username,
		AuthMethod:    h.AuthMethod,
		Weight:        h.Weight,
		HealthStatus:  h.HealthStatus,
		HealthScore:   h.HealthScore,
		LastCheckAt:   h.LastCheckAt,
		LastSuccessAt: h.LastSuccessAt,
		CreatedAt:     h.CreatedAt,
		UpdatedAt:     h.UpdatedAt,
	}
}

func toHostResponses(hosts []model.SSHHost) []HostResponse {
	out := make([]HostResponse, len(hosts))
	for i := range hosts {
		out[i] = toHostResponse(&hosts[i])
	}
	return out
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

	response.Paged(c, toHostResponses(hosts), total, page, pageSize)
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

	response.Success(c, toHostResponse(host))
}

// CopyHostRequest 从已有 Host 复制；name 必填，其余可覆盖。不传 auth_data 时在库内复制源记录的密文，永不下发前端。
type CopyHostRequest struct {
	Name       string `json:"name" binding:"required"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Weight     int    `json:"weight"`
	AuthData   string `json:"auth_data"`
}

// Copy 基于源 Host 创建副本，默认在服务端复制 AuthData/AuthNonce/AuthMethod；仅当请求中提供 auth_data 时用新明文加密写入。
func (h *HostHandler) Copy(c *gin.Context) {
	sourceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid id")
		return
	}

	var req CopyHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "invalid request: "+err.Error())
		return
	}

	src, err := h.container.HostRepo.FindByID(sourceID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to get source host: "+err.Error())
		return
	}
	if src == nil {
		response.Error(c, http.StatusNotFound, 404, "source host not found")
		return
	}

	host := &model.SSHHost{
		Name:          req.Name,
		Host:          src.Host,
		Port:          src.Port,
		Username:      src.Username,
		AuthMethod:    src.AuthMethod,
		AuthData:      src.AuthData,
		AuthNonce:     src.AuthNonce,
		Weight:        src.Weight,
		HealthStatus:  "unknown",
		HealthScore:   0,
		LastCheckAt:   0,
		LastSuccessAt: 0,
	}

	if req.Host != "" {
		host.Host = req.Host
	}
	if req.Port >= 1 && req.Port <= 65535 {
		host.Port = req.Port
	}
	if req.Username != "" {
		host.Username = req.Username
	}
	if req.Weight >= 1 && req.Weight <= 100 {
		host.Weight = req.Weight
	}

	if req.AuthData != "" {
		if err := validator.ValidateAuthMethod(host.AuthMethod); err != nil {
			response.Error(c, http.StatusBadRequest, 400, err.Error())
			return
		}
		encryptedData, nonce, err := crypto.Encrypt(req.AuthData, h.container.Config.Encryption.Key)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, 500, "failed to encrypt auth data: "+err.Error())
			return
		}
		host.AuthData = encryptedData
		host.AuthNonce = nonce
	}

	if err := h.container.HostRepo.Create(host); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to copy host: "+err.Error())
		return
	}

	response.Success(c, toHostResponse(host))
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

	response.Success(c, toHostResponse(host))
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

	response.Success(c, toHostResponse(host))
}

// Delete 删除 SSH Host（软删除）
func (h *HostHandler) Delete(c *gin.Context) {
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

	if err := h.container.HostRepo.Delete(id); err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "failed to delete host: "+err.Error())
		return
	}

	metrics.CleanupHost(strconv.FormatUint(host.ID, 10), host.Name)

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

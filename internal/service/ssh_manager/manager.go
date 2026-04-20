package ssh_manager

import (
	"fmt"
	"log"
	"sync"

	"ssh-port-forwarder/internal/config"
	"ssh-port-forwarder/internal/model"
	"ssh-port-forwarder/internal/pkg/crypto"
	"ssh-port-forwarder/internal/repository"
	"golang.org/x/crypto/ssh"
)

// Manager 管理所有 SSH 连接
type Manager struct {
	mu       sync.RWMutex
	clients  map[uint64]*SSHClient // hostID -> SSHClient
	hostRepo repository.SSHHostRepository
	ruleRepo repository.ForwardRuleRepository
	encConfig config.EncryptionConfig
	stopCh   chan struct{}
}

// NewManager 创建新的 Manager 实例
func NewManager(
	hostRepo repository.SSHHostRepository,
	ruleRepo repository.ForwardRuleRepository,
	encConfig config.EncryptionConfig,
) *Manager {
	return &Manager{
		clients:   make(map[uint64]*SSHClient),
		hostRepo:  hostRepo,
		ruleRepo:  ruleRepo,
		encConfig: encConfig,
		stopCh:    make(chan struct{}),
	}
}

// Start 启动管理器
func (m *Manager) Start() error {
	log.Printf("[SSHManager] Starting...")
	
	// 加载所有活跃的转发规则并启动
	rules, err := m.ruleRepo.ListActive()
	if err != nil {
		return fmt.Errorf("failed to list active rules: %w", err)
	}

	for _, rule := range rules {
		if rule.ActiveHostID == 0 {
			continue
		}

		// 获取对应的 Host
		host, err := m.hostRepo.FindByID(rule.ActiveHostID)
		if err != nil {
			log.Printf("[SSHManager] Failed to find host %d for rule %d: %v", 
				rule.ActiveHostID, rule.ID, err)
			continue
		}

		// 连接到 Host
		if err := m.ConnectHost(host); err != nil {
			log.Printf("[SSHManager] Failed to connect host %d for rule %d: %v",
				host.ID, rule.ID, err)
			continue
		}

		// 启动转发规则
		if err := m.StartForwardRule(&rule, host.ID); err != nil {
			log.Printf("[SSHManager] Failed to start forward rule %d: %v",
				rule.ID, err)
		}
	}

	log.Printf("[SSHManager] Started successfully")
	return nil
}

// Stop 停止所有连接和转发
func (m *Manager) Stop() {
	log.Printf("[SSHManager] Stopping...")

	close(m.stopCh)

	m.mu.Lock()
	defer m.mu.Unlock()

	for hostID, client := range m.clients {
		if err := client.Disconnect(); err != nil {
			log.Printf("[SSHManager] Error disconnecting host %d: %v", hostID, err)
		}
		delete(m.clients, hostID)
	}

	log.Printf("[SSHManager] Stopped")
}

// GetClient 获取指定 Host 的 Client
func (m *Manager) GetClient(hostID uint64) *SSHClient {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clients[hostID]
}

// GetAllClients 获取所有 Client 的快照（返回副本）
func (m *Manager) GetAllClients() map[uint64]*SSHClient {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[uint64]*SSHClient)
	for k, v := range m.clients {
		result[k] = v
	}
	return result
}

// ConnectHost 建立到指定 Host 的连接
func (m *Manager) ConnectHost(host *model.SSHHost) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已连接
	if client, exists := m.clients[host.ID]; exists && client.IsConnected() {
		return nil
	}

	// 构建 SSH 配置
	sshConfig, err := m.buildSSHConfig(host)
	if err != nil {
		return fmt.Errorf("failed to build SSH config: %w", err)
	}

	// 创建 SSHClient
	client := NewSSHClient(host, sshConfig)

	// 建立连接
	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// 保存到 clients 映射
	m.clients[host.ID] = client

	log.Printf("[SSHManager] Connected to host %d (%s@%s:%d)",
		host.ID, host.Username, host.Host, host.Port)

	return nil
}

// DisconnectHost 断开指定 Host
func (m *Manager) DisconnectHost(hostID uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[hostID]
	if !exists {
		return fmt.Errorf("host %d not found", hostID)
	}

	if err := client.Disconnect(); err != nil {
		return fmt.Errorf("failed to disconnect: %w", err)
	}

	delete(m.clients, hostID)

	log.Printf("[SSHManager] Disconnected from host %d", hostID)
	return nil
}

// StartForwardRule 在指定 Host 上启动转发
func (m *Manager) StartForwardRule(rule *model.ForwardRule, hostID uint64) error {
	m.mu.RLock()
	client, exists := m.clients[hostID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("host %d not connected", hostID)
	}

	if err := client.StartForward(rule); err != nil {
		return fmt.Errorf("failed to start forward: %w", err)
	}

	return nil
}

// StopForwardRule 停止转发
func (m *Manager) StopForwardRule(ruleID uint64, hostID uint64) error {
	m.mu.RLock()
	client, exists := m.clients[hostID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("host %d not connected", hostID)
	}

	if err := client.StopForward(ruleID); err != nil {
		return fmt.Errorf("failed to stop forward: %w", err)
	}

	return nil
}

// parsePrivateKey 解析私钥，支持多种格式：
// - PKCS#1: -----BEGIN RSA PRIVATE KEY-----
// - PKCS#8: -----BEGIN PRIVATE KEY-----
// - OpenSSH new format: -----BEGIN OPENSSH PRIVATE KEY-----
// - EC: -----BEGIN EC PRIVATE KEY-----
func parsePrivateKey(keyData []byte) (ssh.Signer, error) {
	// 首先尝试标准解析（支持大多数格式，包括 OpenSSH 新格式）
	signer, err := ssh.ParsePrivateKey(keyData)
	if err == nil {
		return signer, nil
	}

	// 如果失败，尝试使用 passphrase 解析（可能需要处理加密私钥）
	// 注意：当前不支持带密码的私钥，如果需要可以扩展
	return nil, fmt.Errorf("unable to parse private key: %w (supported formats: PKCS#1, PKCS#8, OpenSSH new format)", err)
}

// buildSSHConfig 构建 SSH 配置
// - 解密凭证
// - HostKeyCallback 使用 ssh.InsecureIgnoreHostKey()（生产环境可配置）
// - Timeout 设为 10s
func (m *Manager) buildSSHConfig(host *model.SSHHost) (*ssh.ClientConfig, error) {
	// 解密凭证
	authData, err := crypto.DecryptWithFallback(
		host.AuthData,
		host.AuthNonce,
		m.encConfig.Key,
		m.encConfig.KeyPrevious,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt auth data: %w", err)
	}

	var authMethod ssh.AuthMethod

	switch host.AuthMethod {
	case "password":
		authMethod = ssh.Password(authData)

	case "private_key":
		signer, err := parsePrivateKey([]byte(authData))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethod = ssh.PublicKeys(signer)

	default:
		return nil, fmt.Errorf("unsupported auth method: %s", host.AuthMethod)
	}

	config := &ssh.ClientConfig{
		User:            host.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         0, // 在 Connect 中通过 Dial 的 timeout 控制
	}

	return config, nil
}

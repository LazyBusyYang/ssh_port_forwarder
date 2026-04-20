package ssh_manager

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"ssh-port-forwarder/internal/model"
	"golang.org/x/crypto/ssh"
)

// 连接状态机
type ConnState int

const (
	ConnStateDisconnected ConnState = iota
	ConnStateConnecting
	ConnStateConnected
	ConnStateReconnecting
	ConnStateFailed
)

func (s ConnState) String() string {
	switch s {
	case ConnStateDisconnected:
		return "disconnected"
	case ConnStateConnecting:
		return "connecting"
	case ConnStateConnected:
		return "connected"
	case ConnStateReconnecting:
		return "reconnecting"
	case ConnStateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// ForwardEntry 表示一个端口转发条目
type ForwardEntry struct {
	RuleID     uint64
	LocalAddr  string // "0.0.0.0:12001"
	RemoteAddr string // "10.0.0.101:3306"
	listener   net.Listener
	stopCh     chan struct{}
	mu         sync.Mutex
	active     bool
}

// SSHClient 代表与一个 SSH Host 的连接会话
type SSHClient struct {
	mu        sync.RWMutex
	client    *ssh.Client
	host      *model.SSHHost
	state     ConnState
	forwards  map[uint64]*ForwardEntry // ruleID -> ForwardEntry
	stopCh    chan struct{}
	sshConfig *ssh.ClientConfig
}

// NewSSHClient 创建新的 SSHClient 实例
func NewSSHClient(host *model.SSHHost, sshConfig *ssh.ClientConfig) *SSHClient {
	return &SSHClient{
		host:      host,
		state:     ConnStateDisconnected,
		forwards:  make(map[uint64]*ForwardEntry),
		stopCh:    make(chan struct{}),
		sshConfig: sshConfig,
	}
}

// Connect 建立 SSH 连接，启动 KeepAlive goroutine
func (c *SSHClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == ConnStateConnected || c.state == ConnStateConnecting {
		return nil
	}

	c.state = ConnStateConnecting
	log.Printf("[SSHClient] Connecting to %s@%s:%d", c.host.Username, c.host.Host, c.host.Port)

	addr := fmt.Sprintf("%s:%d", c.host.Host, c.host.Port)
	client, err := ssh.Dial("tcp", addr, c.sshConfig)
	if err != nil {
		c.state = ConnStateFailed
		return fmt.Errorf("failed to dial SSH: %w", err)
	}

	c.client = client
	c.state = ConnStateConnected
	c.stopCh = make(chan struct{})

	// 启动 KeepAlive goroutine
	go c.keepAlive()

	log.Printf("[SSHClient] Connected to %s@%s:%d", c.host.Username, c.host.Host, c.host.Port)
	return nil
}

// Disconnect 断开连接，关闭所有转发
func (c *SSHClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 通知所有 goroutine 停止
	close(c.stopCh)

	// 停止所有转发
	for ruleID, entry := range c.forwards {
		c.stopForwardEntry(entry)
		delete(c.forwards, ruleID)
	}

	// 关闭 SSH 连接
	if c.client != nil {
		if err := c.client.Close(); err != nil {
			log.Printf("[SSHClient] Error closing SSH client: %v", err)
		}
		c.client = nil
	}

	c.state = ConnStateDisconnected
	log.Printf("[SSHClient] Disconnected from %s@%s:%d", c.host.Username, c.host.Host, c.host.Port)
	return nil
}

// State 获取当前状态（线程安全）
func (c *SSHClient) State() ConnState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// SetState 设置状态（线程安全）
func (c *SSHClient) SetState(state ConnState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = state
}

// IsConnected 检查是否已连接
func (c *SSHClient) IsConnected() bool {
	return c.State() == ConnStateConnected
}

// GetClient 获取底层 SSH 客户端（线程安全）
func (c *SSHClient) GetClient() *ssh.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}

// GetHost 获取关联的 Host
func (c *SSHClient) GetHost() *model.SSHHost {
	return c.host
}

// GetStopCh 获取停止信号通道
func (c *SSHClient) GetStopCh() chan struct{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stopCh
}

// keepAlive 每 15s 发送 SSH keepalive 请求，失败时触发重连
func (c *SSHClient) keepAlive() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			client := c.GetClient()
			if client == nil {
				continue
			}

			// 发送 keepalive 请求
			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				log.Printf("[SSHClient] KeepAlive failed for %s@%s:%d: %v", 
					c.host.Username, c.host.Host, c.host.Port, err)
				
				// 设置状态为重连中
				c.SetState(ConnStateReconnecting)
				
				// 启动重连循环
				go c.StartReconnectLoop()
				return
			}
		}
	}
}

// stopForwardEntry 停止单个转发条目
func (c *SSHClient) stopForwardEntry(entry *ForwardEntry) {
	entry.mu.Lock()
	defer entry.mu.Unlock()

	if !entry.active {
		return
	}

	entry.active = false
	if entry.listener != nil {
		entry.listener.Close()
	}
	close(entry.stopCh)
}

// copyData 在两个连接之间双向复制数据
func copyData(localConn, remoteConn net.Conn, stopCh chan struct{}) {
	var wg sync.WaitGroup
	wg.Add(2)

	// local -> remote
	go func() {
		defer wg.Done()
		_, err := io.Copy(remoteConn, localConn)
		if err != nil {
			// 忽略连接关闭的错误
			select {
			case <-stopCh:
			default:
				if err != io.EOF {
					log.Printf("[SSHClient] Error copying local->remote: %v", err)
				}
			}
		}
		remoteConn.Close()
	}()

	// remote -> local
	go func() {
		defer wg.Done()
		_, err := io.Copy(localConn, remoteConn)
		if err != nil {
			// 忽略连接关闭的错误
			select {
			case <-stopCh:
			default:
				if err != io.EOF {
					log.Printf("[SSHClient] Error copying remote->local: %v", err)
				}
			}
		}
		localConn.Close()
	}()

	wg.Wait()
}

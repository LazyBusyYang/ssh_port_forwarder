package ssh_manager

import (
	"fmt"
	"log"
	"net"

	"ssh-port-forwarder/internal/model"
	"golang.org/x/crypto/ssh"
)

// StartForward 启动一条转发
func (c *SSHClient) StartForward(rule *model.ForwardRule) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查是否已存在
	if _, exists := c.forwards[rule.ID]; exists {
		return fmt.Errorf("forward rule %d already exists", rule.ID)
	}

	// 构建本地和远程地址
	localAddr := fmt.Sprintf("0.0.0.0:%d", rule.LocalPort)
	remoteAddr := fmt.Sprintf("%s:%d", rule.TargetHost, rule.TargetPort)

	// 创建监听器
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", localAddr, err)
	}

	// 创建转发条目
	entry := &ForwardEntry{
		RuleID:     rule.ID,
		LocalAddr:  localAddr,
		RemoteAddr: remoteAddr,
		listener:   listener,
		stopCh:     make(chan struct{}),
		active:     true,
	}

	// 记录到 forwards 映射
	c.forwards[rule.ID] = entry

	// 启动监听 goroutine
	go c.acceptConnections(entry)

	log.Printf("[SSHClient] Started forward rule %d: %s -> %s via %s@%s:%d",
		rule.ID, localAddr, remoteAddr, c.host.Username, c.host.Host, c.host.Port)

	return nil
}

// StopForward 停止一条转发
func (c *SSHClient) StopForward(ruleID uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.forwards[ruleID]
	if !exists {
		return fmt.Errorf("forward rule %d not found", ruleID)
	}

	c.stopForwardEntry(entry)
	delete(c.forwards, ruleID)

	log.Printf("[SSHClient] Stopped forward rule %d", ruleID)
	return nil
}

// StopAllForwards 停止所有转发
func (c *SSHClient) StopAllForwards() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for ruleID, entry := range c.forwards {
		c.stopForwardEntry(entry)
		delete(c.forwards, ruleID)
	}

	log.Printf("[SSHClient] Stopped all forward rules")
}

// GetForward 获取转发条目
func (c *SSHClient) GetForward(ruleID uint64) *ForwardEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.forwards[ruleID]
}

// GetAllForwards 获取所有转发条目的快照
func (c *SSHClient) GetAllForwards() map[uint64]*ForwardEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[uint64]*ForwardEntry)
	for k, v := range c.forwards {
		result[k] = v
	}
	return result
}

// acceptConnections 接受连接并处理
func (c *SSHClient) acceptConnections(entry *ForwardEntry) {
	for {
		select {
		case <-entry.stopCh:
			return
		default:
		}

		// 设置接受超时，以便定期检查 stopCh
		listener := entry.listener
		if listener == nil {
			return
		}

		localConn, err := listener.Accept()
		if err != nil {
			select {
			case <-entry.stopCh:
				return
			default:
				log.Printf("[SSHClient] Accept error on %s: %v", entry.LocalAddr, err)
				continue
			}
		}

		// 获取当前 SSH 客户端
		client := c.GetClient()
		if client == nil {
			log.Printf("[SSHClient] SSH client not connected, closing connection")
			localConn.Close()
			continue
		}

		// 处理连接
		go handleConnection(localConn, entry.RemoteAddr, client, entry.stopCh)
	}
}

// handleConnection 处理单个连接的双向转发
func handleConnection(localConn net.Conn, remoteAddr string, client *ssh.Client, stopCh chan struct{}) {
	defer localConn.Close()

	// 建立 SSH channel 到远程地址
	remoteConn, err := client.Dial("tcp", remoteAddr)
	if err != nil {
		log.Printf("[SSHClient] Failed to dial remote %s: %v", remoteAddr, err)
		return
	}
	defer remoteConn.Close()

	log.Printf("[SSHClient] New connection: %s <-> %s", localConn.RemoteAddr(), remoteAddr)

	// 双向复制数据
	copyData(localConn, remoteConn, stopCh)

	log.Printf("[SSHClient] Connection closed: %s <-> %s", localConn.RemoteAddr(), remoteAddr)
}

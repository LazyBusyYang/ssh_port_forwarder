package ssh_manager

import (
	"fmt"
	"log"
	"net"

	"golang.org/x/crypto/ssh"
	"ssh-port-forwarder/internal/model"
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
	listener, err := c.createListener(localAddr)
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
	go c.acceptConnections(listener, entry.stopCh, localAddr, remoteAddr)

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

// SuspendAllForwards closes listeners while retaining their entries as the
// authoritative reconnect recovery set. StopForward can remove an entry while
// reconnect is in progress, which prevents a deleted or restarted rule from
// being restored later.
func (c *SSHClient) SuspendAllForwards() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, entry := range c.forwards {
		c.stopForwardEntry(entry)
	}

	log.Printf("[SSHClient] Suspended all forward rules")
}

// RestoreSuspendedForwards restores only entries that are still registered.
// Holding c.mu across listener creation closes the race where StopForward could
// delete a rule after it was selected for recovery but before the listener was
// published.
func (c *SSHClient) RestoreSuspendedForwards() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for ruleID, entry := range c.forwards {
		entry.mu.Lock()
		if entry.active {
			entry.mu.Unlock()
			continue
		}

		listener, err := c.createListener(entry.LocalAddr)
		if err != nil {
			entry.mu.Unlock()
			log.Printf("[SSHClient] Failed to recreate listener for rule %d on %s: %v",
				ruleID, entry.LocalAddr, err)
			continue
		}

		entry.listener = listener
		entry.stopCh = make(chan struct{})
		entry.active = true
		stopCh := entry.stopCh
		localAddr := entry.LocalAddr
		remoteAddr := entry.RemoteAddr
		entry.mu.Unlock()

		go c.acceptConnections(listener, stopCh, localAddr, remoteAddr)
		log.Printf("[SSHClient] Restored forward rule %d: %s -> %s",
			ruleID, localAddr, remoteAddr)
	}
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

// acceptConnections handles exactly one immutable listener generation. A
// reconnect publishes a new generation instead of changing the resources read
// by an existing accept loop.
func (c *SSHClient) acceptConnections(
	listener net.Listener,
	stopCh <-chan struct{},
	localAddr string,
	remoteAddr string,
) {
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		localConn, err := listener.Accept()
		if err != nil {
			select {
			case <-stopCh:
				return
			default:
				log.Printf("[SSHClient] Accept error on %s: %v", localAddr, err)
				continue
			}
		}

		// 获取当前 SSH 客户端
		client := c.GetClient()
		if client == nil {
			log.Printf("[SSHClient] SSH client not connected, closing connection")
			_ = localConn.Close() // ignore close error
			continue
		}

		// 处理连接
		go handleConnection(localConn, remoteAddr, client, stopCh)
	}
}

// handleConnection 处理单个连接的双向转发
func handleConnection(localConn net.Conn, remoteAddr string, client *ssh.Client, stopCh <-chan struct{}) {
	defer func() {
		_ = localConn.Close() // ignore close error
	}()

	// 建立 SSH channel 到远程地址
	remoteConn, err := client.Dial("tcp", remoteAddr)
	if err != nil {
		log.Printf("[SSHClient] Failed to dial remote %s: %v", remoteAddr, err)
		return
	}
	defer func() {
		_ = remoteConn.Close() // ignore close error
	}()

	log.Printf("[SSHClient] New connection: %s <-> %s", localConn.RemoteAddr(), remoteAddr)

	// 双向复制数据
	copyData(localConn, remoteConn, stopCh)

	log.Printf("[SSHClient] Connection closed: %s <-> %s", localConn.RemoteAddr(), remoteAddr)
}

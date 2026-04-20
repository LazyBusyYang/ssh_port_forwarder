package ssh_manager

import (
	"log"
	"math"
	"net"
	"time"
)

const (
	minReconnectDelay = 1 * time.Second
	maxReconnectDelay = 60 * time.Second
)

// StartReconnectLoop 重连循环
// 指数退避：1s → 2s → 4s → 8s → ... → 最大 60s
// 重连成功后重置退避
// 重连成功后重建所有之前的转发
// 通过 stopCh 可取消
func (c *SSHClient) StartReconnectLoop() {
	c.mu.Lock()
	
	// 如果已经在重连中，避免重复启动
	if c.state == ConnStateReconnecting {
		c.mu.Unlock()
		log.Printf("[SSHClient] Reconnect loop already running for %s@%s:%d",
			c.host.Username, c.host.Host, c.host.Port)
		return
	}
	
	c.state = ConnStateReconnecting
	c.mu.Unlock()

	log.Printf("[SSHClient] Starting reconnect loop for %s@%s:%d",
		c.host.Username, c.host.Host, c.host.Port)

	// 保存当前的转发规则，以便重连后恢复
	existingForwards := c.GetAllForwards()
	forwardRules := make([]*ForwardEntry, 0, len(existingForwards))
	for _, entry := range existingForwards {
		forwardRules = append(forwardRules, entry)
	}

	// 停止所有现有转发（但保留记录）
	c.StopAllForwards()

	// 关闭现有 SSH 连接
	c.mu.Lock()
	if c.client != nil {
		c.client.Close()
		c.client = nil
	}
	c.mu.Unlock()

	// 重连退避计数器
	attempt := 0

	for {
		select {
		case <-c.stopCh:
			log.Printf("[SSHClient] Reconnect loop cancelled for %s@%s:%d",
				c.host.Username, c.host.Host, c.host.Port)
			return
		default:
		}

		// 计算退避延迟
		delay := calculateBackoff(attempt)
		attempt++

		log.Printf("[SSHClient] Reconnect attempt %d for %s@%s:%d, waiting %v",
			attempt, c.host.Username, c.host.Host, c.host.Port, delay)

		// 等待退避时间
		select {
		case <-c.stopCh:
			log.Printf("[SSHClient] Reconnect loop cancelled during backoff for %s@%s:%d",
				c.host.Username, c.host.Host, c.host.Port)
			return
		case <-time.After(delay):
		}

		// 尝试重新连接
		err := c.Connect()
		if err != nil {
			log.Printf("[SSHClient] Reconnect attempt %d failed for %s@%s:%d: %v",
				attempt, c.host.Username, c.host.Host, c.host.Port, err)
			continue
		}

		// 重连成功
		log.Printf("[SSHClient] Reconnect successful for %s@%s:%d after %d attempts",
			c.host.Username, c.host.Host, c.host.Port, attempt)

		// 重建所有转发规则
		for _, entry := range forwardRules {
			// 重新创建转发条目
			newEntry := &ForwardEntry{
				RuleID:     entry.RuleID,
				LocalAddr:  entry.LocalAddr,
				RemoteAddr: entry.RemoteAddr,
				stopCh:     make(chan struct{}),
				active:     true,
			}

			// 创建监听器
			listener, err := c.createListener(newEntry.LocalAddr)
			if err != nil {
				log.Printf("[SSHClient] Failed to recreate listener for rule %d on %s: %v",
					entry.RuleID, newEntry.LocalAddr, err)
				continue
			}
			newEntry.listener = listener

			// 添加到 forwards 映射
			c.mu.Lock()
			c.forwards[entry.RuleID] = newEntry
			c.mu.Unlock()

			// 启动监听 goroutine
			go c.acceptConnections(newEntry)

			log.Printf("[SSHClient] Restored forward rule %d: %s -> %s",
				entry.RuleID, newEntry.LocalAddr, newEntry.RemoteAddr)
		}

		return // 重连成功，退出重连循环
	}
}

// calculateBackoff 计算指数退避延迟
func calculateBackoff(attempt int) time.Duration {
	// 指数退避: 2^attempt 秒
	delay := time.Duration(math.Pow(2, float64(attempt))) * time.Second

	// 限制最大延迟
	if delay > maxReconnectDelay {
		delay = maxReconnectDelay
	}

	// 确保最小延迟
	if delay < minReconnectDelay {
		delay = minReconnectDelay
	}

	return delay
}

// createListener 创建 TCP 监听器（辅助方法）
func (c *SSHClient) createListener(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}

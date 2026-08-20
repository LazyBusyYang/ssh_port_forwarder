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
	if !c.beginReconnect() {
		log.Printf("[SSHClient] Reconnect loop already running for %s@%s:%d",
			c.host.Username, c.host.Host, c.host.Port)
		return
	}
	defer c.endReconnect()

	log.Printf("[SSHClient] Starting reconnect loop for %s@%s:%d",
		c.host.Username, c.host.Host, c.host.Port)

	// 关闭 listener，但将条目保留为受锁保护的待恢复集合。restart/delete
	// 可在退避期间删除条目，恢复阶段只处理仍然存在的规则。
	c.SuspendAllForwards()

	// 关闭现有 SSH 连接
	c.mu.Lock()
	if c.client != nil {
		_ = c.client.Close() // ignore close error
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

		c.RestoreSuspendedForwards()

		return // 重连成功，退出重连循环
	}
}

func (c *SSHClient) beginReconnect() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.reconnectRunning {
		return false
	}
	c.reconnectRunning = true
	c.state = ConnStateReconnecting
	return true
}

func (c *SSHClient) endReconnect() {
	c.mu.Lock()
	c.reconnectRunning = false
	c.mu.Unlock()
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
	return c.listenerFactory(addr)
}

package health

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// HealthEvent 健康状态变更事件
type HealthEvent struct {
	HostID       uint64
	HealthStatus string  // "healthy" / "unhealthy"
	HealthScore  float64
	LatencyMs    float64
	CheckedAt    int64
}

// CheckResult 单次检测结果
type CheckResult struct {
	Success   bool
	LatencyMs float64
}

// TCPDetect TCP握手检测
func TCPDetect(host string, port int, timeout time.Duration) CheckResult {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	start := time.Now()

	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return CheckResult{
			Success:   false,
			LatencyMs: float64(time.Since(start).Milliseconds()),
		}
	}
	defer conn.Close()

	return CheckResult{
		Success:   true,
		LatencyMs: float64(time.Since(start).Milliseconds()),
	}
}

// SSHDetect SSH实际连接验证
func SSHDetect(host string, port int, sshClient *ssh.Client) CheckResult {
	start := time.Now()

	if sshClient == nil {
		return CheckResult{
			Success:   false,
			LatencyMs: float64(time.Since(start).Milliseconds()),
		}
	}

	// 发送 keepalive 请求验证连接
	_, _, err := sshClient.SendRequest("keepalive@openssh.com", true, nil)
	if err != nil {
		return CheckResult{
			Success:   false,
			LatencyMs: float64(time.Since(start).Milliseconds()),
		}
	}

	return CheckResult{
		Success:   true,
		LatencyMs: float64(time.Since(start).Milliseconds()),
	}
}

// TunnelDetect 通过SSH Tunnel检测目标端口
func TunnelDetect(sshClient *ssh.Client, targetHost string, targetPort int, timeout time.Duration) CheckResult {
	start := time.Now()

	if sshClient == nil {
		return CheckResult{
			Success:   false,
			LatencyMs: float64(time.Since(start).Milliseconds()),
		}
	}

	targetAddr := net.JoinHostPort(targetHost, fmt.Sprintf("%d", targetPort))

	// 通过 SSH Tunnel 连接目标
	conn, err := sshClient.Dial("tcp", targetAddr)
	if err != nil {
		return CheckResult{
			Success:   false,
			LatencyMs: float64(time.Since(start).Milliseconds()),
		}
	}
	defer conn.Close()

	return CheckResult{
		Success:   true,
		LatencyMs: float64(time.Since(start).Milliseconds()),
	}
}

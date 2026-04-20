package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// SSH Host 健康状态（1=healthy, 0=unhealthy）
	SSHHostHealth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ssh_host_health",
			Help: "SSH host health status (1=healthy, 0=unhealthy)",
		},
		[]string{"host"},
	)

	// SSH Host 重连次数
	SSHHostReconnectTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ssh_host_reconnect_total",
			Help: "Total number of SSH host reconnection attempts",
		},
		[]string{"host"},
	)

	// 端口转发在线状态（1=active, 0=inactive）
	ForwardActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "forward_active",
			Help: "Port forwarding active status (1=active, 0=inactive)",
		},
		[]string{"local_port", "target"},
	)

	// 当前活跃的 SSH 连接数
	SSHConnectionActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ssh_connection_active",
			Help: "Number of active SSH connections",
		},
	)

	// 端口转发流量字节数
	SSHForwardBytesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ssh_forward_bytes_total",
			Help: "Total bytes transferred through SSH port forwarding",
		},
		[]string{"direction", "local_port"},
	)

	// 健康检查延迟
	SSHHealthcheckLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ssh_healthcheck_latency_seconds",
			Help:    "SSH health check latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"host"},
	)

	// LB 切换次数
	LBFailoverTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lb_failover_total",
			Help: "Total number of load balancer failover events",
		},
		[]string{"from_host", "to_host"},
	)
)

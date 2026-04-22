package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// SPFHostHealth is 1 when the host is healthy, 0 otherwise.
	SPFHostHealth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "spf_host_health",
			Help: "SSH host health status (1=healthy, 0=unhealthy)",
		},
		[]string{"host_id", "host_name"},
	)

	// SPFHostLatency is keepalive RTT in seconds (from SSHDetect).
	SPFHostLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "spf_host_latency_seconds",
			Help:    "SSH host keepalive round-trip latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"host_id", "host_name"},
	)

	// SPFHostRuleLoad is the number of active forward rules assigned to the host.
	SPFHostRuleLoad = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "spf_host_rule_load",
			Help: "Number of active forward rules currently assigned to the host",
		},
		[]string{"host_id", "host_name"},
	)

	// SPFRuleHealth is 1 when the rule is healthy, 0 otherwise.
	SPFRuleHealth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "spf_rule_health",
			Help: "Forward rule health status (1=healthy, 0=unhealthy)",
		},
		[]string{"rule_id", "rule_name", "group_id"},
	)

	// SPFRuleHostSwitchTotal counts host switches per rule.
	SPFRuleHostSwitchTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "spf_rule_host_switch_total",
			Help: "Total number of times a forward rule switched its active host",
		},
		[]string{"rule_id", "rule_name", "from_host_id", "to_host_id"},
	)

	// SPFHostGroupInfo maps host to group membership for Grafana label joins (value is always 1).
	SPFHostGroupInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "spf_host_group_info",
			Help: "Host to group membership mapping (value is always 1)",
		},
		[]string{"host_id", "host_name", "group_id", "group_name"},
	)
)

// CleanupHost removes gauge time series for a deleted host. SPFHostGroupInfo is cleared on RemoveHost from the group handler.
func CleanupHost(hostID, hostName string) {
	SPFHostHealth.DeleteLabelValues(hostID, hostName)
	SPFHostLatency.DeleteLabelValues(hostID, hostName)
	SPFHostRuleLoad.DeleteLabelValues(hostID, hostName)
}

// CleanupRule removes gauge time series for a deleted rule. Counter SPFRuleHostSwitchTotal is not reset.
func CleanupRule(ruleID, ruleName, groupID string) {
	SPFRuleHealth.DeleteLabelValues(ruleID, ruleName, groupID)
}

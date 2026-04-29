package lb

import (
	"sync/atomic"

	"ssh-port-forwarder/internal/model"
)

// Strategy LB 策略接口
type Strategy interface {
	// SelectHost 从候选 Host 列表中为一条 Rule 选择 Active Host
	// hosts: 该 Group 中所有健康的 Host
	// activeRuleCounts: hostID -> 当前承载的活跃 Rule 数量
	SelectHost(hosts []model.SSHHost, activeRuleCounts map[uint64]int64) *model.SSHHost
}

// RoundRobinStrategy 轮询策略
type RoundRobinStrategy struct {
	counter uint64 // 原子计数器
}

// SelectHost 使用轮询算法选择 Host
func (r *RoundRobinStrategy) SelectHost(hosts []model.SSHHost, activeRuleCounts map[uint64]int64) *model.SSHHost {
	if len(hosts) == 0 {
		return nil
	}
	// 原子递增计数器
	idx := atomic.AddUint64(&r.counter, 1) - 1
	selected := hosts[idx%uint64(len(hosts))]
	return &selected
}

// LeastRulesStrategy 最少规则优先策略
type LeastRulesStrategy struct{}

// SelectHost 选择当前承载规则最少的 Host
func (l *LeastRulesStrategy) SelectHost(hosts []model.SSHHost, activeRuleCounts map[uint64]int64) *model.SSHHost {
	if len(hosts) == 0 {
		return nil
	}

	var selected *model.SSHHost
	minCount := int64(^uint64(0) >> 1) // MaxInt64

	for i := range hosts {
		host := &hosts[i]
		count := activeRuleCounts[host.ID]
		if count < minCount {
			minCount = count
			selected = host
		}
	}

	return selected
}

// WeightedStrategy 加权分配策略，按规则数/权重计算归一化负载。
type WeightedStrategy struct{}

// SelectHost 选择新增规则后的预测负载 (active_rule_count + 1) / weight 最小的 Host。
// 如果历史数据中的 weight <= 0，则按 weight = 1 处理。
func (w *WeightedStrategy) SelectHost(hosts []model.SSHHost, activeRuleCounts map[uint64]int64) *model.SSHHost {
	if len(hosts) == 0 {
		return nil
	}

	var selected *model.SSHHost
	var selectedCount int64

	for i := range hosts {
		host := &hosts[i]
		count := activeRuleCounts[host.ID]
		if selected == nil || weightedCandidateLess(host, count, selected, selectedCount) {
			selected = host
			selectedCount = count
		}
	}

	return selected
}

func weightedCandidateLess(candidate *model.SSHHost, candidateCount int64, current *model.SSHHost, currentCount int64) bool {
	candidateWeight := normalizedWeight(candidate.Weight)
	currentWeight := normalizedWeight(current.Weight)

	// 比较 (count + 1) / weight，使用交叉相乘避免浮点误差。
	candidateScore := (candidateCount + 1) * int64(currentWeight)
	currentScore := (currentCount + 1) * int64(candidateWeight)
	if candidateScore != currentScore {
		return candidateScore < currentScore
	}
	if candidateCount != currentCount {
		return candidateCount < currentCount
	}
	if candidateWeight != currentWeight {
		return candidateWeight > currentWeight
	}
	return candidate.ID < current.ID
}

func normalizedWeight(weight int) int {
	if weight <= 0 {
		return 1
	}
	return weight
}

// NewStrategy 根据名称创建策略实例
func NewStrategy(name string) Strategy {
	switch name {
	case "round_robin":
		return &RoundRobinStrategy{}
	case "least_rules":
		return &LeastRulesStrategy{}
	case "weighted":
		return &WeightedStrategy{}
	default:
		return &RoundRobinStrategy{}
	}
}

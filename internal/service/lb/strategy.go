package lb

import (
	"math/rand"
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

// WeightedStrategy 加权分配策略
type WeightedStrategy struct{}

// SelectHost 按权重加权随机选择 Host
func (w *WeightedStrategy) SelectHost(hosts []model.SSHHost, activeRuleCounts map[uint64]int64) *model.SSHHost {
	if len(hosts) == 0 {
		return nil
	}
	if len(hosts) == 1 {
		return &hosts[0]
	}

	// 计算总权重
	totalWeight := 0
	for i := range hosts {
		totalWeight += hosts[i].Weight
	}

	if totalWeight <= 0 {
		// 如果所有权重都为 0 或负数，使用均等随机
		return &hosts[rand.Intn(len(hosts))]
	}

	// 加权随机选择
	r := rand.Intn(totalWeight)
	cumulativeWeight := 0

	for i := range hosts {
		cumulativeWeight += hosts[i].Weight
		if r < cumulativeWeight {
			return &hosts[i]
		}
	}

	// 兜底返回最后一个
	return &hosts[len(hosts)-1]
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

package lb

import (
	"errors"
	"log"
	"sync"

	"ssh-port-forwarder/internal/model"
	"ssh-port-forwarder/internal/repository"
	"ssh-port-forwarder/internal/service/health"
	"ssh-port-forwarder/internal/service/ssh_manager"
)

// Pool LB Pool 管理器
type Pool struct {
	mu         sync.RWMutex
	groupRepo  repository.ForwardGroupRepository
	ruleRepo   repository.ForwardRuleRepository
	hostRepo   repository.SSHHostRepository
	sshManager *ssh_manager.Manager
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewPool 创建新的 Pool 实例
func NewPool(
	groupRepo repository.ForwardGroupRepository,
	ruleRepo repository.ForwardRuleRepository,
	hostRepo repository.SSHHostRepository,
	sshManager *ssh_manager.Manager,
) *Pool {
	return &Pool{
		groupRepo:  groupRepo,
		ruleRepo:   ruleRepo,
		hostRepo:   hostRepo,
		sshManager: sshManager,
		stopCh:     make(chan struct{}),
	}
}

// Start 启动 Pool，监听健康事件
func (p *Pool) Start(healthEventCh <-chan health.HealthEvent) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for {
			select {
			case event, ok := <-healthEventCh:
				if !ok {
					log.Printf("[LB Pool] Health event channel closed")
					return
				}
				p.handleHealthEvent(event)
			case <-p.stopCh:
				log.Printf("[LB Pool] Stopping event listener")
				return
			}
		}
	}()
	log.Printf("[LB Pool] Started")
}

// Stop 停止 Pool
func (p *Pool) Stop() {
	close(p.stopCh)
	p.wg.Wait()
	log.Printf("[LB Pool] Stopped")
}

// handleHealthEvent 处理健康事件
func (p *Pool) handleHealthEvent(event health.HealthEvent) {
	switch event.HealthStatus {
	case "unhealthy":
		log.Printf("[LB Pool] Host %d became unhealthy, triggering failover", event.HostID)
		p.HandleFailover(event.HostID)
	case "healthy":
		log.Printf("[LB Pool] Host %d recovered", event.HostID)
		p.HandleHostRecovery(event.HostID)
	}
}

// AssignHostForRule 为一条 Rule 分配 Host
func (p *Pool) AssignHostForRule(rule *model.ForwardRule) (*model.SSHHost, error) {
	// 获取 ForwardGroup
	group, err := p.groupRepo.FindByIDWithHosts(rule.GroupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, errors.New("forward group not found")
	}

	// 过滤出健康的 Host
	var healthyHosts []model.SSHHost
	for _, host := range group.Hosts {
		if host.HealthStatus == "healthy" {
			healthyHosts = append(healthyHosts, host)
		}
	}

	if len(healthyHosts) == 0 {
		return nil, errors.New("no healthy hosts available in group")
	}

	// 获取各 Host 的活跃 Rule 数量
	activeRuleCounts := make(map[uint64]int64)
	for _, host := range healthyHosts {
		count, err := p.ruleRepo.CountActiveByHostID(host.ID)
		if err != nil {
			log.Printf("[LB Pool] Failed to count active rules for host %d: %v", host.ID, err)
			count = 0
		}
		activeRuleCounts[host.ID] = count
	}

	// 使用 Group 的策略选择 Host
	strategy := NewStrategy(group.Strategy)
	selectedHost := strategy.SelectHost(healthyHosts, activeRuleCounts)

	if selectedHost == nil {
		return nil, errors.New("failed to select host using strategy")
	}

	return selectedHost, nil
}

// HandleFailover 处理故障切换
func (p *Pool) HandleFailover(hostID uint64) {
	// 查找所有使用该 Host 的活跃 Rule
	rules, err := p.ruleRepo.ListActive()
	if err != nil {
		log.Printf("[LB Pool] Failed to list active rules during failover: %v", err)
		return
	}

	var affectedRules []model.ForwardRule
	for _, rule := range rules {
		if rule.ActiveHostID == hostID {
			affectedRules = append(affectedRules, rule)
		}
	}

	if len(affectedRules) == 0 {
		return
	}

	log.Printf("[LB Pool] Found %d rules affected by host %d failure", len(affectedRules), hostID)

	// 对每条受影响的 Rule 进行故障切换
	for _, rule := range affectedRules {
		if err := p.failoverRule(&rule); err != nil {
			log.Printf("[LB Pool] Failed to failover rule %d: %v", rule.ID, err)
		}
	}
}

// failoverRule 对单条 Rule 执行故障切换
func (p *Pool) failoverRule(rule *model.ForwardRule) error {
	oldHostID := rule.ActiveHostID

	// 1. 停止旧转发
	if oldHostID != 0 {
		if err := p.sshManager.StopForwardRule(rule.ID, oldHostID); err != nil {
			log.Printf("[LB Pool] Failed to stop old forward for rule %d on host %d: %v",
				rule.ID, oldHostID, err)
			// 继续处理，不要中断
		}
	}

	// 2. 选择新 Host
	newHost, err := p.AssignHostForRule(rule)
	if err != nil {
		// 没有可用的 Host，标记 Rule 为 inactive
		log.Printf("[LB Pool] No healthy host available for rule %d, marking as inactive: %v",
			rule.ID, err)
		if updateErr := p.ruleRepo.UpdateStatus(rule.ID, "inactive"); updateErr != nil {
			log.Printf("[LB Pool] Failed to update rule %d status to inactive: %v",
				rule.ID, updateErr)
		}
		if updateErr := p.ruleRepo.UpdateActiveHost(rule.ID, 0); updateErr != nil {
			log.Printf("[LB Pool] Failed to clear active host for rule %d: %v",
				rule.ID, updateErr)
		}
		return err
	}

	// 3. 连接到新 Host（如果尚未连接）
	if err := p.sshManager.ConnectHost(newHost); err != nil {
		log.Printf("[LB Pool] Failed to connect to new host %d for rule %d: %v",
			newHost.ID, rule.ID, err)
		// 标记为 inactive
		if updateErr := p.ruleRepo.UpdateStatus(rule.ID, "inactive"); updateErr != nil {
			log.Printf("[LB Pool] Failed to update rule %d status to inactive: %v",
				rule.ID, updateErr)
		}
		return err
	}

	// 4. 启动新转发
	if err := p.sshManager.StartForwardRule(rule, newHost.ID); err != nil {
		log.Printf("[LB Pool] Failed to start forward for rule %d on host %d: %v",
			rule.ID, newHost.ID, err)
		// 标记为 inactive
		if updateErr := p.ruleRepo.UpdateStatus(rule.ID, "inactive"); updateErr != nil {
			log.Printf("[LB Pool] Failed to update rule %d status to inactive: %v",
				rule.ID, updateErr)
		}
		return err
	}

	// 5. 更新 Rule 状态
	if err := p.ruleRepo.UpdateActiveHost(rule.ID, newHost.ID); err != nil {
		log.Printf("[LB Pool] Failed to update active host for rule %d: %v", rule.ID, err)
		// 继续处理，不要中断
	}

	log.Printf("[LB Pool] Rule %d failed over from host %d to host %d",
		rule.ID, oldHostID, newHost.ID)

	return nil
}

// HandleHostRecovery 处理 Host 恢复
func (p *Pool) HandleHostRecovery(hostID uint64) {
	// 获取恢复的 Host
	host, err := p.hostRepo.FindByID(hostID)
	if err != nil {
		log.Printf("[LB Pool] Failed to find recovered host %d: %v", hostID, err)
		return
	}
	if host == nil {
		log.Printf("[LB Pool] Recovered host %d not found", hostID)
		return
	}

	// 获取所有 inactive 的 Rule
	// 注意：这里需要查询所有 Rule，然后筛选出 inactive 且 Group 包含该 Host 的
	// 由于 repository 没有直接提供 ListInactive 方法，我们通过 List 获取所有 Rule
	// 实际实现中可能需要添加更高效的查询方法
	groups, _, err := p.groupRepo.List(1, 1000)
	if err != nil {
		log.Printf("[LB Pool] Failed to list groups during host recovery: %v", err)
		return
	}

	// 检查每个 Group 是否包含该 Host
	for _, group := range groups {
		groupWithHosts, err := p.groupRepo.FindByIDWithHosts(group.ID)
		if err != nil {
			log.Printf("[LB Pool] Failed to get group %d with hosts: %v", group.ID, err)
			continue
		}

		// 检查该 Group 是否包含恢复的 Host
		hasHost := false
		for _, h := range groupWithHosts.Hosts {
			if h.ID == hostID {
				hasHost = true
				break
			}
		}

		if !hasHost {
			continue
		}

		// 获取该 Group 下的所有 inactive Rules
		rules, err := p.ruleRepo.ListByGroupID(group.ID)
		if err != nil {
			log.Printf("[LB Pool] Failed to list rules for group %d: %v", group.ID, err)
			continue
		}

		// 尝试重新激活 inactive 的 Rules
		for _, rule := range rules {
			if rule.Status == "inactive" {
				if err := p.activateRule(&rule); err != nil {
					log.Printf("[LB Pool] Failed to reactivate rule %d: %v", rule.ID, err)
				}
			}
		}
	}
}

// activateRule 激活一条 Rule
func (p *Pool) activateRule(rule *model.ForwardRule) error {
	// 为 Rule 分配 Host
	host, err := p.AssignHostForRule(rule)
	if err != nil {
		return err
	}

	// 连接到 Host
	if err := p.sshManager.ConnectHost(host); err != nil {
		return err
	}

	// 启动转发
	if err := p.sshManager.StartForwardRule(rule, host.ID); err != nil {
		return err
	}

	// 更新 Rule 状态
	if err := p.ruleRepo.UpdateActiveHost(rule.ID, host.ID); err != nil {
		log.Printf("[LB Pool] Failed to update active host for rule %d: %v", rule.ID, err)
	}
	if err := p.ruleRepo.UpdateStatus(rule.ID, "active"); err != nil {
		log.Printf("[LB Pool] Failed to update status for rule %d: %v", rule.ID, err)
	}

	log.Printf("[LB Pool] Rule %d reactivated on host %d", rule.ID, host.ID)
	return nil
}

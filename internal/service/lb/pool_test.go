package lb

import (
	"testing"

	"ssh-port-forwarder/internal/model"
)

func TestPoolRoundRobinPersistsPerGroup(t *testing.T) {
	groupRepo := newTestGroupRepo()
	groupRepo.groups[1] = &model.ForwardGroup{
		ID:       1,
		Strategy: "round_robin",
		Hosts:    testHealthyHosts(),
	}
	ruleRepo := &testRuleRepo{counts: map[uint64]int64{}}
	pool := NewPool(groupRepo, ruleRepo, nil, nil)

	first := mustAssignHost(t, pool, 1)
	second := mustAssignHost(t, pool, 1)
	third := mustAssignHost(t, pool, 1)

	if first.ID != 1 || second.ID != 2 || third.ID != 1 {
		t.Fatalf("expected round robin sequence 1,2,1, got %d,%d,%d", first.ID, second.ID, third.ID)
	}
}

func TestPoolRoundRobinStateIsPerGroup(t *testing.T) {
	groupRepo := newTestGroupRepo()
	groupRepo.groups[1] = &model.ForwardGroup{
		ID:       1,
		Strategy: "round_robin",
		Hosts:    testHealthyHosts(),
	}
	groupRepo.groups[2] = &model.ForwardGroup{
		ID:       2,
		Strategy: "round_robin",
		Hosts:    testHealthyHosts(),
	}
	ruleRepo := &testRuleRepo{counts: map[uint64]int64{}}
	pool := NewPool(groupRepo, ruleRepo, nil, nil)

	group1First := mustAssignHost(t, pool, 1)
	group1Second := mustAssignHost(t, pool, 1)
	group2First := mustAssignHost(t, pool, 2)

	if group1First.ID != 1 || group1Second.ID != 2 {
		t.Fatalf("expected group 1 round robin sequence 1,2, got %d,%d", group1First.ID, group1Second.ID)
	}
	if group2First.ID != 1 {
		t.Fatalf("expected group 2 to have independent round robin state and start at host 1, got %d", group2First.ID)
	}
}

func TestPoolStrategyChangeResetsCachedStrategy(t *testing.T) {
	groupRepo := newTestGroupRepo()
	groupRepo.groups[1] = &model.ForwardGroup{
		ID:       1,
		Strategy: "round_robin",
		Hosts:    testHealthyHosts(),
	}
	ruleRepo := &testRuleRepo{counts: map[uint64]int64{}}
	pool := NewPool(groupRepo, ruleRepo, nil, nil)

	first := mustAssignHost(t, pool, 1)
	second := mustAssignHost(t, pool, 1)
	if first.ID != 1 || second.ID != 2 {
		t.Fatalf("expected initial round robin sequence 1,2, got %d,%d", first.ID, second.ID)
	}

	groupRepo.groups[1].Strategy = "weighted"
	weighted := mustAssignHost(t, pool, 1)
	if weighted.ID != 2 {
		t.Fatalf("expected weighted strategy to prefer host 2, got %d", weighted.ID)
	}

	groupRepo.groups[1].Strategy = "round_robin"
	afterReset := mustAssignHost(t, pool, 1)
	if afterReset.ID != 1 {
		t.Fatalf("expected round robin to reset after strategy change and start at host 1, got %d", afterReset.ID)
	}
}

func mustAssignHost(t *testing.T, pool *Pool, groupID uint64) *model.SSHHost {
	t.Helper()
	host, err := pool.AssignHostForRule(&model.ForwardRule{GroupID: groupID})
	if err != nil {
		t.Fatalf("AssignHostForRule failed: %v", err)
	}
	if host == nil {
		t.Fatal("expected selected host")
	}
	return host
}

func testHealthyHosts() []model.SSHHost {
	return []model.SSHHost{
		{ID: 1, Name: "a", Weight: 50, HealthStatus: "healthy"},
		{ID: 2, Name: "b", Weight: 100, HealthStatus: "healthy"},
	}
}

type testGroupRepo struct {
	groups map[uint64]*model.ForwardGroup
}

func newTestGroupRepo() *testGroupRepo {
	return &testGroupRepo{groups: make(map[uint64]*model.ForwardGroup)}
}

func (r *testGroupRepo) Create(group *model.ForwardGroup) error {
	r.groups[group.ID] = group
	return nil
}

func (r *testGroupRepo) FindByID(id uint64) (*model.ForwardGroup, error) {
	return r.groups[id], nil
}

func (r *testGroupRepo) FindByIDWithHosts(id uint64) (*model.ForwardGroup, error) {
	return r.groups[id], nil
}

func (r *testGroupRepo) Update(group *model.ForwardGroup) error {
	r.groups[group.ID] = group
	return nil
}

func (r *testGroupRepo) Delete(id uint64) error {
	delete(r.groups, id)
	return nil
}

func (r *testGroupRepo) List(page, pageSize int) ([]model.ForwardGroup, int64, error) {
	groups := make([]model.ForwardGroup, 0, len(r.groups))
	for _, group := range r.groups {
		groups = append(groups, *group)
	}
	return groups, int64(len(groups)), nil
}

func (r *testGroupRepo) AddHost(groupID, hostID uint64) error {
	return nil
}

func (r *testGroupRepo) RemoveHost(groupID, hostID uint64) error {
	return nil
}

func (r *testGroupRepo) GetHosts(groupID uint64) ([]model.SSHHost, error) {
	group := r.groups[groupID]
	if group == nil {
		return nil, nil
	}
	return group.Hosts, nil
}

type testRuleRepo struct {
	counts map[uint64]int64
}

func (r *testRuleRepo) Create(rule *model.ForwardRule) error {
	return nil
}

func (r *testRuleRepo) FindByID(id uint64) (*model.ForwardRule, error) {
	return nil, nil
}

func (r *testRuleRepo) Update(rule *model.ForwardRule) error {
	return nil
}

func (r *testRuleRepo) Delete(id uint64) error {
	return nil
}

func (r *testRuleRepo) List(page, pageSize int) ([]model.ForwardRule, int64, error) {
	return nil, 0, nil
}

func (r *testRuleRepo) ListByGroupID(groupID uint64) ([]model.ForwardRule, error) {
	return nil, nil
}

func (r *testRuleRepo) ListActive() ([]model.ForwardRule, error) {
	return nil, nil
}

func (r *testRuleRepo) FindByLocalPort(port int) (*model.ForwardRule, error) {
	return nil, nil
}

func (r *testRuleRepo) UpdateStatus(id uint64, status string) error {
	return nil
}

func (r *testRuleRepo) UpdateActiveHost(id uint64, hostID uint64) error {
	return nil
}

func (r *testRuleRepo) CountActiveByHostID(hostID uint64) (int64, error) {
	return r.counts[hostID], nil
}

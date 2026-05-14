package repository

import (
	"testing"

	"ssh-port-forwarder/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newForwardGroupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("sqlite open: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.SSHHost{},
		&model.ForwardGroup{},
		&model.ForwardRule{},
		&model.HealthHistory{},
		&model.AuditLog{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
}

func TestForwardGroupRepository_CountHosts_many2many(t *testing.T) {
	db := newForwardGroupTestDB(t)
	repo := NewForwardGroupRepository(db)

	g := &model.ForwardGroup{Name: "g1", Strategy: "round_robin"}
	if err := db.Create(g).Error; err != nil {
		t.Fatal(err)
	}
	h1 := &model.SSHHost{Name: "h1", Host: "10.0.0.1", Port: 22, Username: "u", AuthMethod: "password", AuthData: "enc"}
	h2 := &model.SSHHost{Name: "h2", Host: "10.0.0.2", Port: 22, Username: "u", AuthMethod: "password", AuthData: "enc"}
	if err := db.Create(h1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(h2).Error; err != nil {
		t.Fatal(err)
	}

	if err := repo.AddHost(g.ID, h1.ID); err != nil {
		t.Fatalf("AddHost h1: %v", err)
	}
	if err := repo.AddHost(g.ID, h2.ID); err != nil {
		t.Fatalf("AddHost h2: %v", err)
	}

	n, err := repo.CountHosts(g.ID)
	if err != nil {
		t.Fatalf("CountHosts: %v", err)
	}
	if n != 2 {
		t.Fatalf("CountHosts want 2 got %d", n)
	}

	if err := repo.RemoveHost(g.ID, h1.ID); err != nil {
		t.Fatalf("RemoveHost: %v", err)
	}
	n, err = repo.CountHosts(g.ID)
	if err != nil {
		t.Fatalf("CountHosts after remove: %v", err)
	}
	if n != 1 {
		t.Fatalf("CountHosts after remove want 1 got %d", n)
	}
}

func TestForwardGroupRepository_FindByIDWithHosts_preloadsRules(t *testing.T) {
	db := newForwardGroupTestDB(t)
	repo := NewForwardGroupRepository(db)

	g := &model.ForwardGroup{Name: "g1", Strategy: "round_robin"}
	if err := db.Create(g).Error; err != nil {
		t.Fatal(err)
	}
	rule := &model.ForwardRule{
		GroupID:    g.ID,
		Name:       "rule-a",
		LocalPort:  18080,
		TargetHost: "127.0.0.1",
		TargetPort: 8080,
		Protocol:   "tcp",
		Status:     "inactive",
	}
	if err := db.Create(rule).Error; err != nil {
		t.Fatal(err)
	}

	group, err := repo.FindByIDWithHosts(g.ID)
	if err != nil {
		t.Fatalf("FindByIDWithHosts: %v", err)
	}
	if group == nil {
		t.Fatal("FindByIDWithHosts: nil group")
	}
	if len(group.Rules) != 1 {
		t.Fatalf("Rules len want 1 got %d", len(group.Rules))
	}
	if group.Rules[0].Name != "rule-a" {
		t.Fatalf("Rule name got %q", group.Rules[0].Name)
	}

	byGetRules, err := repo.GetRules(g.ID)
	if err != nil {
		t.Fatalf("GetRules: %v", err)
	}
	if len(byGetRules) != 1 {
		t.Fatalf("GetRules len want 1 got %d", len(byGetRules))
	}
}

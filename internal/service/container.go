package service

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"ssh-port-forwarder/internal/config"
	"ssh-port-forwarder/internal/model"
	"ssh-port-forwarder/internal/repository"
	"ssh-port-forwarder/internal/service/health"
	"ssh-port-forwarder/internal/service/lb"
	"ssh-port-forwarder/internal/service/scheduler"
	"ssh-port-forwarder/internal/service/ssh_manager"
)

// Container 依赖注入容器，管理所有服务和组件
type Container struct {
	Config *config.Config

	// Database
	DBAdapter *repository.GORMAdapter

	// Repositories
	UserRepo   repository.UserRepository
	HostRepo   repository.SSHHostRepository
	GroupRepo  repository.ForwardGroupRepository
	RuleRepo   repository.ForwardRuleRepository
	HealthRepo repository.HealthHistoryRepository
	AuditRepo  repository.AuditLogRepository

	// Services
	AuthService   *AuthService
	SSHManager    *ssh_manager.Manager
	HealthChecker *health.Checker
	LBPool        *lb.Pool
	Scheduler     *scheduler.Scheduler
}

// NewContainer 创建新的依赖注入容器
func NewContainer(cfg *config.Config) (*Container, error) {
	// 1. 根据数据库类型确定 DSN
	var dbType, dsn string
	switch cfg.Database.Type {
	case "sqlite":
		dbType = "sqlite"
		dsn = cfg.Database.SQLite.Path
		// 确保 data/ 目录存在
		dir := filepath.Dir(dsn)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create data directory: %w", err)
			}
		}
	case "mysql":
		dbType = "mysql"
		dsn = cfg.Database.MySQL.DSN
		// ENV 兜底：如果配置文件中没有设置 DSN，从环境变量读取
		if dsn == "" {
			dsn = os.Getenv("SPF_DB_DSN")
		}
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
	}

	// 2. 创建 GORMAdapter
	adapter, err := repository.NewGORMAdapter(dbType, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create database adapter: %w", err)
	}
	log.Printf("[Container] Database adapter created (%s)", dbType)

	// 3. 执行 AutoMigrate
	if err := adapter.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}
	log.Printf("[Container] Database auto migration completed")

	// 3b. 旧库无 name 时补全为 rule_{local_port}（方言无关）
	var unnamedRules []model.ForwardRule
	if err := adapter.DB.Where("name = ''").Find(&unnamedRules).Error; err != nil {
		return nil, fmt.Errorf("failed to list rules for name backfill: %w", err)
	}
	for _, r := range unnamedRules {
		if err := adapter.DB.Model(&model.ForwardRule{}).Where("id = ?", r.ID).
			Update("name", fmt.Sprintf("rule_%d", r.LocalPort)).Error; err != nil {
			return nil, fmt.Errorf("failed to backfill rule name id=%d: %w", r.ID, err)
		}
	}
	if len(unnamedRules) > 0 {
		log.Printf("[Container] Forward rule names backfilled (%d records)", len(unnamedRules))
	}

	// 4. 创建所有 Repository 实例
	userRepo := repository.NewUserRepository(adapter.DB)
	hostRepo := repository.NewSSHHostRepository(adapter.DB)
	groupRepo := repository.NewForwardGroupRepository(adapter.DB)
	ruleRepo := repository.NewForwardRuleRepository(adapter.DB)
	healthRepo := repository.NewHealthHistoryRepository(adapter.DB)
	auditRepo := repository.NewAuditLogRepository(adapter.DB)
	log.Printf("[Container] Repositories initialized")

	// 5. 创建 AuthService（JWT 密钥支持环境变量兜底）
	if cfg.JWT.SecretCurrent == "" {
		cfg.JWT.SecretCurrent = os.Getenv("JWT_SECRET_CURRENT")
	}
	authService := NewAuthService(userRepo, cfg.JWT)
	log.Printf("[Container] AuthService created")

	// 6. 创建 SSH Manager（加密密钥支持环境变量兜底）
	if cfg.Encryption.Key == "" {
		cfg.Encryption.Key = os.Getenv("SPF_ENCRYPTION_KEY")
	}
	sshManager := ssh_manager.NewManager(hostRepo, ruleRepo, cfg.Encryption)
	log.Printf("[Container] SSH Manager created")

	// 7. 创建 Health Checker
	healthChecker := health.NewChecker(hostRepo, healthRepo, ruleRepo, sshManager)
	log.Printf("[Container] Health Checker created")

	// 8. 创建 LB Pool
	lbPool := lb.NewPool(groupRepo, ruleRepo, hostRepo, sshManager)
	log.Printf("[Container] LB Pool created")

	// 9. 创建 Scheduler
	sched := scheduler.NewScheduler(healthChecker, sshManager, lbPool, auditRepo, healthRepo, cfg)
	log.Printf("[Container] Scheduler created")

	// 10. 返回 Container
	return &Container{
		Config:        cfg,
		DBAdapter:     adapter,
		UserRepo:      userRepo,
		HostRepo:      hostRepo,
		GroupRepo:     groupRepo,
		RuleRepo:      ruleRepo,
		HealthRepo:    healthRepo,
		AuditRepo:     auditRepo,
		AuthService:   authService,
		SSHManager:    sshManager,
		HealthChecker: healthChecker,
		LBPool:        lbPool,
		Scheduler:     sched,
	}, nil
}

// Start 启动所有服务
func (c *Container) Start() error {
	log.Printf("[Container] Starting all services...")

	// 1. 启动 SSH Manager
	if err := c.SSHManager.Start(); err != nil {
		return fmt.Errorf("failed to start SSH Manager: %w", err)
	}
	log.Printf("[Container] SSH Manager started")

	// 2. 启动 Health Checker
	c.HealthChecker.Start()
	log.Printf("[Container] Health Checker started")

	// 3. 启动 LB Pool
	c.LBPool.Start(c.HealthChecker.EventCh())
	log.Printf("[Container] LB Pool started")

	// 4. 启动 Scheduler
	c.Scheduler.Start()
	log.Printf("[Container] Scheduler started")

	// 5. 创建默认 admin 用户（支持环境变量配置）
	adminUser := os.Getenv("SPF_DEFAULT_ADMIN_USER")
	adminPass := os.Getenv("SPF_DEFAULT_ADMIN_PASS")
	if adminUser == "" {
		adminUser = "admin"
	}
	if adminPass == "" {
		adminPass = "admin123"
	}
	if err := c.AuthService.CreateDefaultAdmin(adminUser, adminPass); err != nil {
		log.Printf("[Container] Failed to create default admin: %v", err)
		// 不返回错误，继续启动
	} else {
		log.Printf("[Container] Default admin user created/verified")
	}

	log.Printf("[Container] All services started successfully")
	return nil
}

// Stop 停止所有服务（反序关闭）
func (c *Container) Stop() {
	log.Printf("[Container] Stopping all services...")

	// 1. 停止 Scheduler
	if c.Scheduler != nil {
		c.Scheduler.Stop()
		log.Printf("[Container] Scheduler stopped")
	}

	// 2. 停止 LB Pool
	if c.LBPool != nil {
		c.LBPool.Stop()
		log.Printf("[Container] LB Pool stopped")
	}

	// 3. 停止 Health Checker
	if c.HealthChecker != nil {
		c.HealthChecker.Stop()
		log.Printf("[Container] Health Checker stopped")
	}

	// 4. 停止 SSH Manager
	if c.SSHManager != nil {
		c.SSHManager.Stop()
		log.Printf("[Container] SSH Manager stopped")
	}

	log.Printf("[Container] All services stopped")
}

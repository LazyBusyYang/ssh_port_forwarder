package repository

import (
	"fmt"
	"ssh-port-forwarder/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type GORMAdapter struct {
	DB *gorm.DB
}

// NewGORMAdapter 根据数据库类型创建适配器
// dbType: "sqlite" 或 "mysql"
// dsn: sqlite 为文件路径，mysql 为 DSN 字符串
func NewGORMAdapter(dbType, dsn string) (*GORMAdapter, error) {
	var dialector gorm.Dialector
	switch dbType {
	case "sqlite":
		dialector = sqlite.Open(dsn)
	case "mysql":
		dialector = mysql.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	return &GORMAdapter{DB: db}, nil
}

// AutoMigrate 自动迁移所有表
func (a *GORMAdapter) AutoMigrate() error {
	return a.DB.AutoMigrate(
		&model.User{},
		&model.SSHHost{},
		&model.ForwardGroup{},
		&model.ForwardRule{},
		&model.HealthHistory{},
		&model.AuditLog{},
	)
}

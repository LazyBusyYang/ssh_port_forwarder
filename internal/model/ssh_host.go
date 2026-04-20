package model

import "gorm.io/gorm"

type SSHHost struct {
	ID            uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string         `gorm:"type:varchar(128);not null" json:"name"`
	Host          string         `gorm:"type:varchar(255);not null" json:"host"`
	Port          int            `gorm:"not null;default:22" json:"port"`
	Username      string         `gorm:"type:varchar(128);not null" json:"username"`
	AuthMethod    string         `gorm:"type:varchar(32);not null" json:"auth_method"` // password / private_key
	AuthData      string         `gorm:"type:varchar(2048);not null" json:"-"`          // AES-256-GCM 加密
	AuthNonce     string         `gorm:"type:varchar(64)" json:"-"`                     // Base64 编码 Nonce
	Weight        int            `gorm:"not null;default:100" json:"weight"`             // LB 权重 1-100
	HealthStatus  string         `gorm:"type:varchar(32);not null;default:unknown" json:"health_status"` // healthy/unhealthy/unknown
	HealthScore   float64        `gorm:"not null;default:0" json:"health_score"`         // 0-100
	LastCheckAt   int64          `gorm:"default:0" json:"last_check_at"`
	LastSuccessAt int64          `gorm:"default:0" json:"last_success_at"`
	CreatedAt     int64          `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     int64          `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

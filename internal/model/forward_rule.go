package model

import "gorm.io/gorm"

type ForwardRule struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupID      uint64         `gorm:"not null;index" json:"group_id"`
	Name         string         `gorm:"type:varchar(128);not null;default:''" json:"name"`
	LocalPort    int            `gorm:"not null;uniqueIndex" json:"local_port"` // 全局唯一，范围 SPF_PORT_RANGE_MIN ~ SPF_PORT_RANGE_MAX
	TargetHost   string         `gorm:"type:varchar(255);not null" json:"target_host"`
	TargetPort   int            `gorm:"not null" json:"target_port"`
	Protocol     string         `gorm:"type:varchar(16);not null;default:tcp" json:"protocol"`
	Status       string         `gorm:"type:varchar(32);not null;default:inactive" json:"status"` // active / inactive
	ActiveHostID uint64         `gorm:"default:0" json:"active_host_id"`                            // 当前承载此规则的 SSH Host
	Group        *ForwardGroup  `gorm:"foreignKey:GroupID" json:"group,omitempty"`
	ActiveHost   *SSHHost       `gorm:"foreignKey:ActiveHostID" json:"active_host,omitempty"`
	CreatedAt    int64          `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    int64          `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

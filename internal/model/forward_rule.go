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
	ActiveHostID *uint64        `gorm:"index" json:"active_host_id,omitempty"`                    // 当前承载此规则的 SSH Host，未绑定为 NULL（MySQL 外键不可写 0）
	Group        *ForwardGroup  `gorm:"foreignKey:GroupID" json:"group,omitempty"`
	ActiveHost   *SSHHost       `gorm:"foreignKey:ActiveHostID" json:"active_host,omitempty"`
	CreatedAt    int64          `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    int64          `gorm:"autoUpdateTime" json:"updated_at"`
	// 删除 API 使用 Unscoped 物理删；保留字段仅为兼容已有表结构，勿对 Rule 使用 GORM 默认软删 Delete。
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// ActiveHostIDUint 返回当前承载 Host 的 id；未绑定时为 0（与历史「用 0 表示无」的数值语义一致，便于日志与指标）。
func (r *ForwardRule) ActiveHostIDUint() uint64 {
	if r == nil || r.ActiveHostID == nil {
		return 0
	}
	return *r.ActiveHostID
}

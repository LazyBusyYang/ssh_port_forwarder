package model

import "gorm.io/gorm"

type ForwardGroup struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string         `gorm:"type:varchar(128);not null" json:"name"`
	Strategy  string         `gorm:"type:varchar(32);not null;default:round_robin" json:"strategy"` // round_robin / least_rules / weighted
	Hosts     []SSHHost      `gorm:"many2many:forward_group_hosts;" json:"hosts,omitempty"`
	Rules     []ForwardRule  `gorm:"foreignKey:GroupID" json:"rules,omitempty"`
	CreatedAt int64          `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt int64          `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

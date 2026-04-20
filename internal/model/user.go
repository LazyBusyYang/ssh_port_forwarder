package model

import "gorm.io/gorm"

type User struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	PasswordHash string         `gorm:"type:varchar(256);not null" json:"-"`
	Role         string         `gorm:"type:varchar(32);not null;default:operator" json:"role"` // admin / operator
	CreatedAt    int64          `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    int64          `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

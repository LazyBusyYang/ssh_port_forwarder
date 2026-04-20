package model

type AuditLog struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint64 `gorm:"not null;index" json:"user_id"`
	Action     string `gorm:"type:varchar(64);not null" json:"action"`       // host.create / rule.delete 等
	TargetType string `gorm:"type:varchar(64);not null" json:"target_type"` // ssh_host / forward_rule / forward_group
	TargetID   uint64 `gorm:"not null" json:"target_id"`
	Detail     string `gorm:"type:text" json:"detail"`                       // JSON 格式变更详情
	CreatedAt  int64  `gorm:"autoCreateTime;index" json:"created_at"`
}

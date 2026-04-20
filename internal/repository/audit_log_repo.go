package repository

import (
	"ssh-port-forwarder/internal/model"

	"gorm.io/gorm"
)

type auditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository 创建 AuditLogRepository 实例
func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Create(log *model.AuditLog) error {
	return r.db.Create(log).Error
}

func (r *auditLogRepository) List(page, pageSize int, action string, userID uint64) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64

	query := r.db.Model(&model.AuditLog{})

	if action != "" {
		query = query.Where("action = ?", action)
	}
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *auditLogRepository) DeleteBefore(timestamp int64) error {
	return r.db.Where("created_at < ?", timestamp).Delete(&model.AuditLog{}).Error
}

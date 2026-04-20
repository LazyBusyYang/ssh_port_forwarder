package repository

import (
	"ssh-port-forwarder/internal/model"

	"gorm.io/gorm"
)

type healthHistoryRepository struct {
	db *gorm.DB
}

// NewHealthHistoryRepository 创建 HealthHistoryRepository 实例
func NewHealthHistoryRepository(db *gorm.DB) HealthHistoryRepository {
	return &healthHistoryRepository{db: db}
}

func (r *healthHistoryRepository) Create(record *model.HealthHistory) error {
	return r.db.Create(record).Error
}

func (r *healthHistoryRepository) ListByHostID(hostID uint64, startTime, endTime int64, limit int) ([]model.HealthHistory, error) {
	var records []model.HealthHistory
	query := r.db.Where("host_id = ?", hostID)

	if startTime > 0 {
		query = query.Where("checked_at >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where("checked_at <= ?", endTime)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Order("checked_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}

	return records, nil
}

func (r *healthHistoryRepository) DeleteBefore(timestamp int64) error {
	return r.db.Where("checked_at < ?", timestamp).Delete(&model.HealthHistory{}).Error
}

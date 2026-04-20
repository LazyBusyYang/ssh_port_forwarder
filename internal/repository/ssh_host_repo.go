package repository

import (
	"ssh-port-forwarder/internal/model"

	"gorm.io/gorm"
)

type sshHostRepository struct {
	db *gorm.DB
}

// NewSSHHostRepository 创建 SSHHostRepository 实例
func NewSSHHostRepository(db *gorm.DB) SSHHostRepository {
	return &sshHostRepository{db: db}
}

func (r *sshHostRepository) Create(host *model.SSHHost) error {
	return r.db.Create(host).Error
}

func (r *sshHostRepository) FindByID(id uint64) (*model.SSHHost, error) {
	var host model.SSHHost
	if err := r.db.First(&host, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &host, nil
}

func (r *sshHostRepository) Update(host *model.SSHHost) error {
	return r.db.Save(host).Error
}

func (r *sshHostRepository) Delete(id uint64) error {
	return r.db.Delete(&model.SSHHost{}, id).Error
}

func (r *sshHostRepository) List(page, pageSize int) ([]model.SSHHost, int64, error) {
	var hosts []model.SSHHost
	var total int64

	if err := r.db.Model(&model.SSHHost{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&hosts).Error; err != nil {
		return nil, 0, err
	}

	return hosts, total, nil
}

func (r *sshHostRepository) ListAll() ([]model.SSHHost, error) {
	var hosts []model.SSHHost
	if err := r.db.Find(&hosts).Error; err != nil {
		return nil, err
	}
	return hosts, nil
}

func (r *sshHostRepository) UpdateHealthStatus(id uint64, status string, score float64, lastCheckAt int64) error {
	return r.db.Model(&model.SSHHost{}).Where("id = ?", id).Updates(map[string]interface{}{
		"health_status": status,
		"health_score":  score,
		"last_check_at": lastCheckAt,
	}).Error
}

func (r *sshHostRepository) UpdateLastSuccess(id uint64, lastSuccessAt int64) error {
	return r.db.Model(&model.SSHHost{}).Where("id = ?", id).Update("last_success_at", lastSuccessAt).Error
}

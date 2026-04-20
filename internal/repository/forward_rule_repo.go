package repository

import (
	"ssh-port-forwarder/internal/model"

	"gorm.io/gorm"
)

type forwardRuleRepository struct {
	db *gorm.DB
}

// NewForwardRuleRepository 创建 ForwardRuleRepository 实例
func NewForwardRuleRepository(db *gorm.DB) ForwardRuleRepository {
	return &forwardRuleRepository{db: db}
}

func (r *forwardRuleRepository) Create(rule *model.ForwardRule) error {
	return r.db.Create(rule).Error
}

func (r *forwardRuleRepository) FindByID(id uint64) (*model.ForwardRule, error) {
	var rule model.ForwardRule
	if err := r.db.First(&rule, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

func (r *forwardRuleRepository) Update(rule *model.ForwardRule) error {
	return r.db.Save(rule).Error
}

func (r *forwardRuleRepository) Delete(id uint64) error {
	return r.db.Delete(&model.ForwardRule{}, id).Error
}

func (r *forwardRuleRepository) List(page, pageSize int) ([]model.ForwardRule, int64, error) {
	var rules []model.ForwardRule
	var total int64

	if err := r.db.Model(&model.ForwardRule{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&rules).Error; err != nil {
		return nil, 0, err
	}

	return rules, total, nil
}

func (r *forwardRuleRepository) ListByGroupID(groupID uint64) ([]model.ForwardRule, error) {
	var rules []model.ForwardRule
	if err := r.db.Where("group_id = ?", groupID).Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func (r *forwardRuleRepository) ListActive() ([]model.ForwardRule, error) {
	var rules []model.ForwardRule
	if err := r.db.Where("status = ?", "active").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func (r *forwardRuleRepository) FindByLocalPort(port int) (*model.ForwardRule, error) {
	var rule model.ForwardRule
	if err := r.db.Where("local_port = ?", port).First(&rule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

func (r *forwardRuleRepository) UpdateStatus(id uint64, status string) error {
	return r.db.Model(&model.ForwardRule{}).Where("id = ?", id).Update("status", status).Error
}

func (r *forwardRuleRepository) UpdateActiveHost(id uint64, hostID uint64) error {
	return r.db.Model(&model.ForwardRule{}).Where("id = ?", id).Update("active_host_id", hostID).Error
}

func (r *forwardRuleRepository) CountActiveByHostID(hostID uint64) (int64, error) {
	var count int64
	if err := r.db.Model(&model.ForwardRule{}).Where("active_host_id = ? AND status = ?", hostID, "active").Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

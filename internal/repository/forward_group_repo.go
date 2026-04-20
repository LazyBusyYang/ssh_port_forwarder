package repository

import (
	"ssh-port-forwarder/internal/model"

	"gorm.io/gorm"
)

type forwardGroupRepository struct {
	db *gorm.DB
}

// NewForwardGroupRepository 创建 ForwardGroupRepository 实例
func NewForwardGroupRepository(db *gorm.DB) ForwardGroupRepository {
	return &forwardGroupRepository{db: db}
}

func (r *forwardGroupRepository) Create(group *model.ForwardGroup) error {
	return r.db.Create(group).Error
}

func (r *forwardGroupRepository) FindByID(id uint64) (*model.ForwardGroup, error) {
	var group model.ForwardGroup
	if err := r.db.First(&group, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &group, nil
}

func (r *forwardGroupRepository) FindByIDWithHosts(id uint64) (*model.ForwardGroup, error) {
	var group model.ForwardGroup
	if err := r.db.Preload("Hosts").First(&group, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &group, nil
}

func (r *forwardGroupRepository) Update(group *model.ForwardGroup) error {
	return r.db.Save(group).Error
}

func (r *forwardGroupRepository) Delete(id uint64) error {
	return r.db.Delete(&model.ForwardGroup{}, id).Error
}

func (r *forwardGroupRepository) List(page, pageSize int) ([]model.ForwardGroup, int64, error) {
	var groups []model.ForwardGroup
	var total int64

	if err := r.db.Model(&model.ForwardGroup{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&groups).Error; err != nil {
		return nil, 0, err
	}

	return groups, total, nil
}

func (r *forwardGroupRepository) AddHost(groupID, hostID uint64) error {
	var group model.ForwardGroup
	if err := r.db.First(&group, groupID).Error; err != nil {
		return err
	}

	var host model.SSHHost
	if err := r.db.First(&host, hostID).Error; err != nil {
		return err
	}

	return r.db.Model(&group).Association("Hosts").Append(&host)
}

func (r *forwardGroupRepository) RemoveHost(groupID, hostID uint64) error {
	var group model.ForwardGroup
	if err := r.db.First(&group, groupID).Error; err != nil {
		return err
	}

	var host model.SSHHost
	if err := r.db.First(&host, hostID).Error; err != nil {
		return err
	}

	return r.db.Model(&group).Association("Hosts").Delete(&host)
}

func (r *forwardGroupRepository) GetHosts(groupID uint64) ([]model.SSHHost, error) {
	var group model.ForwardGroup
	if err := r.db.First(&group, groupID).Error; err != nil {
		return nil, err
	}

	var hosts []model.SSHHost
	if err := r.db.Model(&group).Association("Hosts").Find(&hosts); err != nil {
		return nil, err
	}

	return hosts, nil
}

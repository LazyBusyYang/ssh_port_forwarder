package repository

import "ssh-port-forwarder/internal/model"

type UserRepository interface {
	Create(user *model.User) error
	FindByID(id uint64) (*model.User, error)
	FindByUsername(username string) (*model.User, error)
	Update(user *model.User) error
	Delete(id uint64) error
	List(page, pageSize int) ([]model.User, int64, error)
}

type SSHHostRepository interface {
	Create(host *model.SSHHost) error
	FindByID(id uint64) (*model.SSHHost, error)
	Update(host *model.SSHHost) error
	Delete(id uint64) error
	List(page, pageSize int) ([]model.SSHHost, int64, error)
	ListAll() ([]model.SSHHost, error)
	UpdateHealthStatus(id uint64, status string, score float64, lastCheckAt int64) error
	UpdateLastSuccess(id uint64, lastSuccessAt int64) error
}

type ForwardGroupRepository interface {
	Create(group *model.ForwardGroup) error
	FindByID(id uint64) (*model.ForwardGroup, error)
	FindByIDWithHosts(id uint64) (*model.ForwardGroup, error)
	Update(group *model.ForwardGroup) error
	Delete(id uint64) error
	List(page, pageSize int) ([]model.ForwardGroup, int64, error)
	AddHost(groupID, hostID uint64) error
	RemoveHost(groupID, hostID uint64) error
	GetHosts(groupID uint64) ([]model.SSHHost, error)
}

type ForwardRuleRepository interface {
	Create(rule *model.ForwardRule) error
	FindByID(id uint64) (*model.ForwardRule, error)
	Update(rule *model.ForwardRule) error
	Delete(id uint64) error
	List(page, pageSize int) ([]model.ForwardRule, int64, error)
	ListByGroupID(groupID uint64) ([]model.ForwardRule, error)
	ListActive() ([]model.ForwardRule, error)
	FindByLocalPort(port int) (*model.ForwardRule, error)
	UpdateStatus(id uint64, status string) error
	UpdateActiveHost(id uint64, hostID uint64) error
	CountActiveByHostID(hostID uint64) (int64, error)
}

type HealthHistoryRepository interface {
	Create(record *model.HealthHistory) error
	ListByHostID(hostID uint64, startTime, endTime int64, limit int) ([]model.HealthHistory, error)
	DeleteBefore(timestamp int64) error
}

type AuditLogRepository interface {
	Create(log *model.AuditLog) error
	List(page, pageSize int, action string, userID uint64) ([]model.AuditLog, int64, error)
	DeleteBefore(timestamp int64) error
}

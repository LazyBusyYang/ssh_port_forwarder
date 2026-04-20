package model

type HealthHistory struct {
	ID        uint64  `gorm:"primaryKey;autoIncrement" json:"id"`
	HostID    uint64  `gorm:"not null;index" json:"host_id"`
	Score     float64 `gorm:"not null" json:"score"`
	IsHealthy bool    `gorm:"not null" json:"is_healthy"`
	LatencyMs float64 `gorm:"not null" json:"latency_ms"`
	CheckedAt int64   `gorm:"not null;index" json:"checked_at"`
}

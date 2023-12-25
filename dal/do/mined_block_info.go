package do

import "time"

type MinedBlockInfo struct {
	ID           uint64 `gorm:"primaryKey"`
	Username     string `gorm:"index:idx_username;type:varchar(100);not null"`
	Height       int64  `gorm:"not null;default:0;index"`
	ContentHash  string `gorm:"not null;type:varchar(64)"`
	BlockHash    string `gorm:"not null;type:varchar(64);index"`
	SealHash     string `gorm:"not null;type:varchar(64);index"`
	Reward       uint64 `gorm:"not null;default:0"`
	Disconnected int    `gorm:"not null;default:0"`
	Connected    int    `gorm:"not null;default:0"`
	Rejected     int    `gorm:"not null;default:0"`
	Allocated    int    `gorm:"not null;default:0"`
	Info         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

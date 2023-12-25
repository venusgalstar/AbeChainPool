package do

import "time"

type AllocationInfo struct {
	ID          uint64 `gorm:"primaryKey"`
	Username    string `gorm:"index:idx_username;type:varchar(100);not null"`
	ShareCount  int64  `gorm:"not null;default:0"`
	Balance     int64  `gorm:"default:0;not null"`
	StartHeight int64  `gorm:"default:0;not null"`
	EndHeight   int64  `gorm:"default:0;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

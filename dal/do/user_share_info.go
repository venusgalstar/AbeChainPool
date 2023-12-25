package do

import "time"

type UserShareInfo struct {
	ID          uint64 `gorm:"primaryKey"`
	Username    string `gorm:"index:idx_username;type:varchar(100);not null"`
	ShareCount  int64  `gorm:"not null;default:0"`
	StartHeight int64  `gorm:"not null;default:0;index"`
	EndHeight   int64  `gorm:"not null;default:0;index"`
	Allocated   int64  `gorm:"not null;default:0;index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

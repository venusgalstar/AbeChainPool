package do

import "time"

type DetailedShareInfo struct {
	ID          uint64 `gorm:"primaryKey"`
	Username    string `gorm:"index:idx_username;type:varchar(100);not null"`
	ShareCount  int64  `gorm:"not null;default:0"`
	Height      int64  `gorm:"not null;default:0;index"`
	ContentHash string `gorm:"not null;type:varchar(64)"`
	Nonce       string `gorm:"not null;type:varchar(64)"`
	ShareHash   string `gorm:"not null;type:varchar(64)"`
	SealHash    string `gorm:"not null;type:varchar(64);index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

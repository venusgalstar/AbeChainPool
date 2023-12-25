package do

import "time"

type MetaInfo struct {
	ID                 uint64 `gorm:"primaryKey"`
	Balance            int64  `gorm:"default:0;not null"`
	AllocatedBalance   int64  `gorm:"default:0;not null"`
	LastRewardHeight   int64  `gorm:"default:0;not null"`
	LastTxCheckHeight  int64  `gorm:"default:0;not null"`
	LastRedeemedHeight int64  `gorm:"default:0;not null"`
	LastRewardTime     time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

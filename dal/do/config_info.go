package do

import "time"

type ConfigInfo struct {
	ID               uint64 `gorm:"primaryKey"`
	Epoch            int64  `gorm:"default:0;not null"` // an epoch is 4000 blocks in chain, means to adjust difficult
	StartHeight      int64  `gorm:"default:0;not null"`
	EndHeight        int64  `gorm:"default:0;not null"`
	RewardInterval   int64  `gorm:"default:0;not null"` // a reward period every epoch, also named block range or span
	RewardMaturity   int64  `gorm:"default:0;not null"` // unless than the coinbase maturity
	ManageFeePercent int    `gorm:"default:0;not null"` // the same management fee in every epoch
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

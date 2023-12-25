package do

import "time"

type OrderInfo struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"index:idx_user_id;not null"`
	Amount    int64  `gorm:"index:idx_amount;not null;default:0"`
	UTXOHash  string `gorm:"not null;default:''"`
	Status    int64  `gorm:"not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

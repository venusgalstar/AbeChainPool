package do

import "time"

type UserInfo struct {
	ID       uint64  `gorm:"primaryKey"`
	Username string  `gorm:"uniqueIndex:unique_idx_username;type:varchar(100);not null"`
	Password string  `gorm:"type:varchar(40);not null"`
	Salt     string  `gorm:"type:varchar(20);not null"`
	Email    *string `gorm:"type:varchar(100)"`
	// Balance is the total reward of a user, including the balance sent and the balance left.
	Balance int64 `gorm:"not null;default:0"`
	// RedeemedBalance is the balance that has been sent to user.
	RedeemedBalance int64 `gorm:"not null;default:0"`
	Address         string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

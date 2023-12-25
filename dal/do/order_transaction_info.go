package do

import "time"

// For order_transaction_info_dao
const (
	TransactionPending = iota - 1
	TransactionWaiting
	TransactionInChain
	TransactionInvalid
	TransactionManual // check the invalid status with time, if possible, mark it as manual and notification the owner to send to transaction off-pool
)

type OrderTransactionInfo struct {
	ID                     uint64 `gorm:"primaryKey"`
	OrderID                uint64 `gorm:"index:idx_order_id;not null"`
	TransactionHash        string `gorm:"not null;type:varchar(64)"`
	TransactionRequestHash string `gorm:"not null;type:varchar(64)"`
	Amount                 int64  `gorm:"not null;default:0"`
	Status                 int64  `gorm:"not null;default:0"`
	Height                 int64  `gorm:"not null;default:0"`
	LastCheckedHeight      int64  `gorm:"not null;default:0"`
	Address                string
	CreatedAt              time.Time
	UpdatedAt              time.Time
	OrderInfo              *OrderInfo `gorm:"foreignKey:OrderID"`
}

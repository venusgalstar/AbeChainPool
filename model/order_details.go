package model

type OrderDetails struct {
	ID           uint64                     `json:"id"`
	UserID       uint64                     `json:"user_id"`
	Amount       int64                      `json:"amount"`
	Status       int64                      `json:"status"`
	Transactions []*OrderTransactionDetails `json:"transactions"`
	CreatedAt    int64                      `json:"created_at"`
	UpdatedAt    int64                      `json:"updated_at"`
}

type OrderTransactionDetails struct {
	ID              uint64 `json:"id"`
	TransactionHash string `json:"transaction_hash"`
	Amount          int64  `json:"amount"`
	Status          int64  `json:"status"`
	Height          int64  `json:"height"`
	Address         string `json:"address"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

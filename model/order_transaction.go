package model

type OrderTransaction struct {
	OrderID                uint64
	Status                 int
	Amount                 int64
	Address                string
	RequestTransactionHash string
}

type OrderTransactionStatus struct {
	ID                     uint64
	OrderID                uint64
	Status                 int
	TransactionHash        string
	RequestTransactionHash string
	Height                 int64
	LastUpdateStatusHeight int64
}

type OrderTransactionInfo struct {
	ID       uint64
	OrderID  uint64
	Status   int
	Address  string
	Amount   int64
	UserID   uint64
	UTXOHash string
}

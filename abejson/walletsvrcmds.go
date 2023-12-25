// NOTE: This file is intended to house the RPC commands that are supported by
// a chain server.

package abejson

import (
	"github.com/abesuite/abe-miningpool-server/utils"
)

// OrderInfo represents the order information for distributing rewards.
type OrderInfo struct {
	OrderID uint64  `json:"-"`
	Address string  `json:"address"`
	Amount  float64 `json:"amount"`
}
type UTXO struct {
	RingHash     string
	TxHash       string
	Index        uint8
	FromCoinbase bool
	Amount       uint64
	Height       int64
	UTXOHash     *utils.Hash
	Allocated    bool
}
type BatchOrderInfo struct {
	*BatchOrderInfoCmd
	ResponseChan chan *TxStatus
}
type TxStatusInfo struct {
	TxHash        *utils.Hash
	RequestTxHash *utils.Hash
	ResponseChan  chan *TxStatus
}
type TxStatus struct {
	TxHash *utils.Hash
	Status int
}

type TxStatusCmd struct {
	TxHash string
}
type TxStatusFromRequestHashCmd struct {
	RequestHash string
}

// BatchOrderInfoCmd defines the command for distributing rewards.
type BatchOrderInfoCmd struct {
	Pairs              []*OrderInfo
	MinConf            *int     `jsonrpcdefault:"1"`
	ScaleToFeeSatPerKb *float64 `jsonrpcdefault:"1"`
	FeeSpecified       *float64 `jsonrpcdefault:"0"`
	UTXOSpecified      *string
}

// NewBatchOrderInfoCmd returns a new instance which can be used to issue a
// distribution bonus program.
func NewBatchOrderInfoCmd(orders []*OrderInfo) *BatchOrderInfoCmd {
	return &BatchOrderInfoCmd{
		Pairs: orders,
	}
}

// WalletUnlockCmd defines the walletpassphrase JSON-RPC command.
type WalletUnlockCmd struct {
	Passphrase string
	Timeout    int64
}

func NewWalletUnlockCmd(passphrase string, timeout int64) *WalletUnlockCmd {
	return &WalletUnlockCmd{
		Passphrase: passphrase,
		Timeout:    timeout,
	}
}

type ListUnspentCoinbaseAbeCmd struct{}

func NewGetUTXOListCmd() *ListUnspentCoinbaseAbeCmd { return new(ListUnspentCoinbaseAbeCmd) }

type WalletLockCmd struct{}

func NewWalletLockCmd() *WalletLockCmd { return new(WalletLockCmd) }

type GetBalancesabeCmd struct{}

func NewGetBalancesabeCmd() *GetBalancesabeCmd { return new(GetBalancesabeCmd) }

type NotificationTransactionsCmd struct{}

func NewNotificationTransactionsCmd() *NotificationTransactionsCmd {
	return &NotificationTransactionsCmd{}
}

func init() {
	// No special flags for commands in this file.
	flags := UFWalletOnly
	MustRegisterCmd("notifytransaction", (*NotificationTransactionsCmd)(nil), flags)
	MustRegisterCmd("sendtoaddressesabe", (*BatchOrderInfoCmd)(nil), flags)
	MustRegisterCmd("walletunlock", (*WalletUnlockCmd)(nil), flags)
	MustRegisterCmd("walletlock", (*WalletLockCmd)(nil), flags)
	MustRegisterCmd("getbalancesabe", (*GetBalancesabeCmd)(nil), flags)
	MustRegisterCmd("transactionstatus", (*TxStatusCmd)(nil), flags)
	MustRegisterCmd("gettxhashfromreqeust", (*TxStatusFromRequestHashCmd)(nil), flags)
	MustRegisterCmd("listmaturecoinbasetxoabe", (*ListUnspentCoinbaseAbeCmd)(nil), flags)
}

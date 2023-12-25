// NOTE: This file is intended to house the RPC websocket notifications that are
// supported by a wallet server.

package abejson

const (
	TxAcceptedNtfnMethod = "txaccepted"
	TxRollbackNtfnMethod = "txrollback"
	TxInvalidNtfnMethod  = "txinvalid"
)

type TxAcceptedNtfn struct {
	Hash string
}

func NewTxAcceptedNtfn(hash string) *TxAcceptedNtfn {
	return &TxAcceptedNtfn{
		Hash: hash,
	}
}

type TxRollbackNtfn struct {
	Hash string
}

func NewTxRollbackNtfn(hash string) *TxRollbackNtfn {
	return &TxRollbackNtfn{
		Hash: hash,
	}
}

type TxInvalidNtfn struct {
	Hash string
}

func NewTxInvalidNtfn(hash string) *TxInvalidNtfn {
	return &TxInvalidNtfn{
		Hash: hash,
	}
}

func init() {
	// The commands in this file are only usable by websockets and are
	// notifications.
	flags := UFWebsocketOnly | UFNotification

	MustRegisterCmd(TxAcceptedNtfnMethod, (*TxAcceptedNtfn)(nil), flags)
	MustRegisterCmd(TxRollbackNtfnMethod, (*TxRollbackNtfn)(nil), flags)
	MustRegisterCmd(TxInvalidNtfnMethod, (*TxInvalidNtfn)(nil), flags)
}

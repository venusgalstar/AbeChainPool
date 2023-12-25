package model

import "github.com/abesuite/abe-miningpool-server/abejson"

type BlockTemplate struct {
	Version      int32  `json:"version"`
	Bits         string `json:"bits"`
	CurTime      int64  `json:"curtime"`
	Height       int64  `json:"height"`
	PreviousHash string `json:"previousblockhash"`

	Transactions []abejson.GetBlockTemplateResultTxAbe   `json:"transactions"`
	CoinbaseTxn  *abejson.GetBlockTemplateResultCoinbase `json:"coinbasetxn,omitempty"`
}

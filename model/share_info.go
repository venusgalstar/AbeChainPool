package model

import "github.com/abesuite/abe-miningpool-server/wire"

type ShareInfo struct {
	Username       string
	Height         int64
	ShareCount     int64
	ContentHash    string
	Nonce          string
	ShareHash      string
	SealHash       string
	Reward         uint64
	CandidateBlock *wire.MsgSimplifiedBlock
}

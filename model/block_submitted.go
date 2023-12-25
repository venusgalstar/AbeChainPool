package model

import (
	"github.com/abesuite/abe-miningpool-server/utils"
	"github.com/abesuite/abe-miningpool-server/wire"
)

type BlockSubmitted struct {
	BlockHash       utils.Hash
	SimplifiedBlock *wire.MsgSimplifiedBlock
}

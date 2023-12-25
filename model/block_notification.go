package model

import (
	"time"

	"github.com/abesuite/abe-miningpool-server/utils"
)

type BlockNotification struct {
	BlockHash utils.Hash
	Height    int32
	Time      time.Time
	Info      string
}

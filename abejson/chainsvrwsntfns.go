// NOTE: This file is intended to house the RPC websocket notifications that are
// supported by a chain server.

package abejson

const (
	// BlockAbeConnectedNtfnMethod is the legacy, deprecated method used for
	// notifications from the chain server that a block has been connected.
	BlockAbeConnectedNtfnMethod = "blockabeconnected"

	// BlockAbeDisconnectedNtfnMethod is the legacy, deprecated method used for
	// notifications from the chain server that a block has been
	// disconnected.
	BlockAbeDisconnectedNtfnMethod = "blockabedisconnected"
)

// BlockAbeConnectedNtfn defines the blockconnected JSON-RPC notification.
type BlockAbeConnectedNtfn struct {
	Hash   string
	Height int32
	Time   int64
}

func NewBlockAbeConnectedNtfn(hash string, height int32, time int64) *BlockAbeConnectedNtfn {
	return &BlockAbeConnectedNtfn{
		Hash:   hash,
		Height: height,
		Time:   time,
	}
}

// BlockAbeDisconnectedNtfn defines the blockdisconnected JSON-RPC notification.
type BlockAbeDisconnectedNtfn struct {
	Hash   string
	Height int32
	Time   int64
}

// NewBlockAbeDisconnectedNtfn returns a new instance which can be used to issue a
// blockdisconnected JSON-RPC notification.
func NewBlockAbeDisconnectedNtfn(hash string, height int32, time int64) *BlockAbeDisconnectedNtfn {
	return &BlockAbeDisconnectedNtfn{
		Hash:   hash,
		Height: height,
		Time:   time,
	}
}

// BlockDetails describes details of a tx in a block.
type BlockDetails struct {
	Height int32  `json:"height"`
	Hash   string `json:"hash"`
	Index  int    `json:"index"`
	Time   int64  `json:"time"`
}

func init() {
	// The commands in this file are only usable by websockets and are
	// notifications.
	flags := UFWebsocketOnly | UFNotification

	MustRegisterCmd(BlockAbeConnectedNtfnMethod, (*BlockAbeConnectedNtfn)(nil), flags)
	MustRegisterCmd(BlockAbeDisconnectedNtfnMethod, (*BlockAbeDisconnectedNtfn)(nil), flags)
}

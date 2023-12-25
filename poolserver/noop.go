package poolserver

import (
	"github.com/abesuite/abe-miningpool-server/pooljson"
)

func handleNoop(wsc AbstractSocketClient, icmd interface{}) (interface{}, error) {
	_, ok := icmd.(*pooljson.NoopCmd)
	if !ok {
		log.Debugf("Client %v sends an invalid mining.noop command.", wsc.RemoteAddr())
		return nil, pooljson.ErrInvalidRequestParams
	}

	resp := pooljson.NoopResult{}

	return &resp, nil
}

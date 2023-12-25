package poolserver

import (
	"strconv"

	"github.com/abesuite/abe-miningpool-server/pooljson"
)

func handleHashRate(wsc AbstractSocketClient, icmd interface{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.HashRateCmd)
	if !ok {
		log.Debugf("Client %v sends an invalid mining.hashrate command.", wsc.RemoteAddr())
		return nil, pooljson.ErrInvalidRequestParams
	}

	hashRate, err := strconv.ParseInt(cmd.HashRate, 16, 64)
	if err != nil {
		log.Debugf("Fail to parse hash rate %v", cmd.HashRate)
		return nil, pooljson.ErrInvalidRequestParams
	}

	minermgr := wsc.GetMinerManager()
	err = minermgr.SetHashRate(wsc.QuitChan(), hashRate)
	if err != nil {
		log.Errorf("Error when set hash rate: %v", err)
		return nil, pooljson.ErrInternal
	}

	resp := pooljson.HashRateResult{}

	return &resp, nil
}

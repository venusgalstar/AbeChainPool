package poolserver

import (
	"github.com/abesuite/abe-miningpool-server/pooljson"
)

// handleSubscribe implements the mining.subscribe command.
func handleSubscribe(wsc AbstractSocketClient, icmd interface{}) (interface{}, error) {
	_, ok := icmd.(*pooljson.SubscribeCmd)
	if !ok {
		log.Debugf("Client %v sends an invalid mining.subscribe command.", wsc.RemoteAddr())
		return nil, pooljson.ErrInvalidRequestParams
	}

	minermgr := wsc.GetMinerManager()
	sessionID, err := minermgr.GenerateNewSession(wsc.QuitChan())
	if err != nil {
		log.Debugf("Client %v subscribes without hello.", wsc.RemoteAddr())
		return nil, err
	}

	log.Debugf("Client %v subscribes, session id is %v.", wsc.RemoteAddr(), sessionID)

	resp := pooljson.SubscribeResult{
		SessionID: sessionID,
	}

	return &resp, nil
}

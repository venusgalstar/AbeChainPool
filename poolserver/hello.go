package poolserver

import (
	"strconv"
	"strings"

	"github.com/abesuite/abe-miningpool-server/pooljson"
	"github.com/abesuite/abe-miningpool-server/utils"
)

func handleHello(wsc AbstractSocketClient, icmd interface{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.HelloCmd)
	if !ok {
		log.Debugf("Client %v sends an invalid mining.hello command.", wsc.RemoteAddr())
		return nil, pooljson.ErrInvalidRequestParams
	}

	log.Debugf("Client (User agent: %v) comes.", cmd.Agent)

	if hasUndesiredUserAgent(wsc.RemoteAddr(), cmd.Agent, wsc.GetAgentBlacklist(), wsc.GetAgentWhitelist()) {
		go wsc.DisconnectGracefully()
		return nil, pooljson.ErrBadUserAgent
	}

	// Currently, we ignore the parameters host and port.
	// Check protocol.
	if cmd.Proto != poolProtocol {
		log.Debugf("Client %v sends an invalid mining.hello command with unsupported protocol %v.", wsc.RemoteAddr(), cmd.Proto)
		go wsc.DisconnectGracefully()
		return nil, pooljson.ErrBadProtocolRequest
	}

	minermgr := wsc.GetMinerManager()
	_, err := minermgr.AddNewMiner(wsc.QuitChan(), wsc.RemoteAddr(), cmd.Agent, cmd.Proto, wsc.Type())
	if err != nil {
		return nil, err
	}

	if wsc.GetAbecBackendNode() == "" {
		wsc.SetAbecBackendNode(utils.GetNodeDesc())
	}
	maxErrors := strconv.FormatInt(int64(wsc.GetMaxErrors()), 16)
	resp := pooljson.HelloResult{
		Proto:     cmd.Proto,
		Encoding:  defaultEncoding,
		Resume:    defaultResume,
		Timeout:   strconv.FormatInt(defaultTimeout, 16),
		MaxErrors: maxErrors,
		Node:      wsc.GetAbecBackendNode(),
	}

	return &resp, nil
}

// HasUndesiredUserAgent determines whether the server should continue to pursue
// a connection with this peer based on its advertised user agent. It performs
// the following steps:
// 1) Reject the peer if it contains a blacklisted agent.
// 2) If no whitelist is provided, accept all user agents.
// 3) Accept the peer if it contains a whitelisted agent.
// 4) Reject all other peers.
func hasUndesiredUserAgent(remoteAddress string, agent string, blacklistedAgents, whitelistedAgents []string) bool {

	// First, if peer's user agent contains any blacklisted substring, we
	// will ignore the connection request.
	for _, blacklistedAgent := range blacklistedAgents {
		if strings.Contains(agent, blacklistedAgent) {
			log.Debugf("Ignoring peer %s, user agent "+
				"contains blacklisted user agent: %s", remoteAddress, agent)
			return true
		}
	}

	// If no whitelist is provided, we will accept all user agents.
	if len(whitelistedAgents) == 0 {
		return false
	}

	// Peer's user agent passed blacklist. Now check to see if it contains
	// one of our whitelisted user agents, if so accept.
	for _, whitelistedAgent := range whitelistedAgents {
		if strings.Contains(agent, whitelistedAgent) {
			return false
		}
	}

	// Otherwise, the peer's user agent was not included in our whitelist.
	// log.Debugf("Ignoring peer %s, user agent: %s not found in "+
	// 	"whitelist", remoteAddress, agent)

	return false
}

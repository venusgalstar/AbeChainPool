package poolserver

import (
	"fmt"
	"strings"

	"github.com/abesuite/abe-miningpool-server/pooljson"
	"github.com/abesuite/abe-miningpool-server/utils"
)

func hasUndesiredUserAgentV1(remoteAddress string, agent string, blacklistedAgents, whitelistedAgents []string) bool {

	fmt.Printf("remoteAddress is %s\n", remoteAddress)
	fmt.Printf("agent is %s\n", agent)

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

		fmt.Printf("whitelist element %s\n", whitelistedAgent)

		if strings.Contains(agent, whitelistedAgent) {
			return false
		}
	}

	fmt.Printf("why am i here?\n")
	// Otherwise, the peer's user agent was not included in our whitelist.
	log.Debugf("Ignoring peer %s, user agent: %s not found in "+
		"whitelist", remoteAddress, agent)

	return false
}

func handleHelloV1(wsc AbstractSocketClient, icmd interface{}) error {
	_, ok := icmd.(*pooljson.SubscribeCmd)
	if !ok {
		log.Debugf("Client %v sends an invalid mining.hello command.", wsc.RemoteAddr())
		return pooljson.ErrInvalidRequestParams
	}

	log.Debugf("Client (User agent: %v) comes.", wsc.RemoteAddr())

	Agent := "e9"

	if hasUndesiredUserAgentV1(wsc.RemoteAddr(), Agent, wsc.GetAgentBlacklist(), wsc.GetAgentWhitelist()) {
		go wsc.DisconnectGracefully()
		return pooljson.ErrBadUserAgent
	}

	minermgr := wsc.GetMinerManager()
	_, err := minermgr.AddNewMiner(wsc.QuitChan(), wsc.RemoteAddr(), Agent, poolProtocol, wsc.Type())
	if err != nil {
		return err
	}

	log.Debugf("GetMinerManager succeed")

	if wsc.GetAbecBackendNode() == "" {
		wsc.SetAbecBackendNode(utils.GetNodeDesc())
	}
	// maxErrors := strconv.FormatInt(int64(wsc.GetMaxErrors()), 16)
	// resp := pooljson.HelloResult{
	// 	Proto:     poolProtocol,
	// 	Encoding:  defaultEncoding,
	// 	Resume:    defaultResume,
	// 	Timeout:   strconv.FormatInt(defaultTimeout, 16),
	// 	MaxErrors: maxErrors,
	// 	Node:      wsc.GetAbecBackendNode(),
	// }

	return nil
}

// handleSubscribe implements the mining.subscribe command.
func handleSubscribe(wsc AbstractSocketClient, icmd interface{}) (interface{}, error) {
	_, ok := icmd.(*pooljson.SubscribeCmd)
	if !ok {
		log.Debugf("Here.")
		log.Debugf("Client %v sends an invalid mining.subscribe command.", wsc.RemoteAddr())
		return nil, pooljson.ErrInvalidRequestParams
	}

	handleHelloV1(wsc, icmd)

	minermgr := wsc.GetMinerManager()

	fmt.Println("GetMinerManager created.", wsc.RemoteAddr())

	sessionID, err := minermgr.GenerateNewSession(wsc.QuitChan())
	if err != nil {
		log.Debugf("Client %v subscribes without hello.", wsc.RemoteAddr())
		return nil, err
	}

	log.Debugf("Client %v subscribes, session id is %v.", wsc.RemoteAddr(), sessionID)

	resp := make([]interface{}, 0, 3)

	body := make([]interface{}, 0, 2)
	notify := make([]interface{}, 0, 2)
	difficulty := make([]interface{}, 0, 2)
	notify = append(notify, "mining.notify")
	notify = append(notify, "000044191")
	difficulty = append(difficulty, "mining.set_difficulty")
	difficulty = append(difficulty, "000044192")
	body = append(body, difficulty)
	body = append(body, notify)

	resp = append(resp, body)
	resp = append(resp, "00004419")
	resp = append(resp, 8)

	return &resp, nil
}

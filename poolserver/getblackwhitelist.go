package poolserver

func handleGetAgentBlacklist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	res := s.GetAgentBlacklist()
	return &res, nil
}

func handleGetAgentWhitelist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	res := s.GetAgentWhitelist()
	return &res, nil
}

func handleGetBlacklist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	blacklist := s.GetBlacklist()
	res := make([]string, 0, len(blacklist))
	for _, ip := range blacklist {
		res = append(res, ip.String())
	}
	return &res, nil
}

func handleGetWhitelist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	whitelist := s.GetWhitelist()
	res := make([]string, 0, len(whitelist))
	for _, ip := range whitelist {
		res = append(res, ip.String())
	}
	return &res, nil
}

func handleGetAdminBlacklist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	blacklist := s.GetAdminBlacklist()
	res := make([]string, 0, len(blacklist))
	for _, ip := range blacklist {
		res = append(res, ip.String())
	}
	return &res, nil
}

func handleGetAdminWhitelist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	whitelist := s.GetAdminWhitelist()
	res := make([]string, 0, len(whitelist))
	for _, ip := range whitelist {
		res = append(res, ip.String())
	}
	return &res, nil
}

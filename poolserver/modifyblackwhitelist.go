package poolserver

import (
	"fmt"
	"net"

	"github.com/abesuite/abe-miningpool-server/pooljson"
	"github.com/abesuite/abe-miningpool-server/utils"
)

func getIPNetFromString(addr string) (*net.IPNet, error) {
	_, ipnet, err := net.ParseCIDR(addr)
	var ip net.IP
	if err != nil {
		ip = net.ParseIP(addr)
		if ip == nil {
			return nil, fmt.Errorf("unable to parse ip: %v", addr)
		}
		var bits int
		if ip.To4() == nil {
			// IPv6
			bits = 128
		} else {
			bits = 32
		}
		ipnet = &net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(bits, bits),
		}
	}
	return ipnet, nil
}

func handleAddAgentBlacklist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.AddAgentBlacklistCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}
	if utils.IsBlank(cmd.Agent) {
		return nil, pooljson.ErrInvalidRequestParams
	}
	s.AddAgentBlacklist(cmd.Agent)
	return &pooljson.ModifyBlackWhitelistResult{
		Success: true,
	}, nil
}

func handleDeleteAgentBlacklist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.DeleteAgentBlacklistCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}
	if utils.IsBlank(cmd.Agent) {
		return nil, pooljson.ErrInvalidRequestParams
	}
	s.DeleteAgentBlacklist(cmd.Agent)
	return &pooljson.ModifyBlackWhitelistResult{
		Success: true,
	}, nil
}

func handleAddAgentWhitelist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.AddAgentWhitelistCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}
	if utils.IsBlank(cmd.Agent) {
		return nil, pooljson.ErrInvalidRequestParams
	}
	s.AddAgentWhitelist(cmd.Agent)
	return &pooljson.ModifyBlackWhitelistResult{
		Success: true,
	}, nil
}

func handleDeleteAgentWhitelist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.DeleteAgentWhitelistCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}
	if utils.IsBlank(cmd.Agent) {
		return nil, pooljson.ErrInvalidRequestParams
	}
	s.DeleteAgentWhitelist(cmd.Agent)
	return &pooljson.ModifyBlackWhitelistResult{
		Success: true,
	}, nil
}

func handleAddBlacklist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.AddBlacklistCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}
	ipnet, err := getIPNetFromString(cmd.IP)
	if err != nil {
		return nil, pooljson.ErrInvalidRequestParams
	}
	s.AddBlacklist(ipnet)
	return &pooljson.ModifyBlackWhitelistResult{
		Success: true,
	}, nil
}

func handleDeleteBlacklist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.DeleteBlacklistCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}
	ipnet, err := getIPNetFromString(cmd.IP)
	if err != nil {
		return nil, pooljson.ErrInvalidRequestParams
	}
	s.DeleteBlacklist(ipnet)
	return &pooljson.ModifyBlackWhitelistResult{
		Success: true,
	}, nil
}

func handleAddWhitelist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.AddWhitelistCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}
	ipnet, err := getIPNetFromString(cmd.IP)
	if err != nil {
		return nil, pooljson.ErrInvalidRequestParams
	}
	s.AddWhitelist(ipnet)
	return &pooljson.ModifyBlackWhitelistResult{
		Success: true,
	}, nil
}

func handleDeleteWhitelist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.DeleteWhitelistCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}
	ipnet, err := getIPNetFromString(cmd.IP)
	if err != nil {
		return nil, pooljson.ErrInvalidRequestParams
	}
	s.DeleteWhitelist(ipnet)
	return &pooljson.ModifyBlackWhitelistResult{
		Success: true,
	}, nil
}

func handleAddAdminBlacklist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.AddAdminBlacklistCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}
	ipnet, err := getIPNetFromString(cmd.IP)
	if err != nil {
		return nil, pooljson.ErrInvalidRequestParams
	}
	s.AddAdminBlacklist(ipnet)
	return &pooljson.ModifyBlackWhitelistResult{
		Success: true,
	}, nil
}

func handleDeleteAdminBlacklist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.DeleteAdminBlacklistCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}
	ipnet, err := getIPNetFromString(cmd.IP)
	if err != nil {
		return nil, pooljson.ErrInvalidRequestParams
	}
	s.DeleteAdminBlacklist(ipnet)
	return &pooljson.ModifyBlackWhitelistResult{
		Success: true,
	}, nil
}

func handleAddAdminWhitelist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.AddAdminWhitelistCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}
	ipnet, err := getIPNetFromString(cmd.IP)
	if err != nil {
		return nil, pooljson.ErrInvalidRequestParams
	}
	s.AddAdminWhitelist(ipnet)
	return &pooljson.ModifyBlackWhitelistResult{
		Success: true,
	}, nil
}

func handleDeleteAdminWhitelist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.DeleteAdminWhitelistCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}
	ipnet, err := getIPNetFromString(cmd.IP)
	if err != nil {
		return nil, pooljson.ErrInvalidRequestParams
	}
	s.DeleteAdminWhitelist(ipnet)
	return &pooljson.ModifyBlackWhitelistResult{
		Success: true,
	}, nil
}

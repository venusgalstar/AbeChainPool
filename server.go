package main

import (
	"errors"

	"github.com/abesuite/abe-miningpool-server/configmgr"
	"github.com/abesuite/abe-miningpool-server/consensus/ethash"
	"github.com/abesuite/abe-miningpool-server/minermgr"
	"github.com/abesuite/abe-miningpool-server/model"
	"github.com/abesuite/abe-miningpool-server/ordermgr"
	"github.com/abesuite/abe-miningpool-server/poolserver"
	"github.com/abesuite/abe-miningpool-server/rewardmgr"
)

type server struct {
	poolRPCServer   *poolserver.PoolServer
	tcpSocketServer *poolserver.TCPSocketServer
	minerManager    *minermgr.MinerManager
	rewardManager   *rewardmgr.RewardManager
	orderManager    *ordermgr.OrderManager
}

func newServer(chainCli *chainClients, walletClient *walletClient) (*server, error) {
	commonCfg := poolserver.CommonConfig{
		AgentBlacklist: cfg.AgentBlacklist,
		AgentWhitelist: cfg.AgentWhitelist,
		Blacklist:      cfg.blacklists,
		Whitelist:      cfg.whitelists,
		AdminBlacklist: cfg.adminBlacklists,
		AdminWhitelist: cfg.adminWhitelists,
	}
	rewardCfg := model.RewardConfig{
		RewardInterval:      cfg.RewardInterval,
		RewardIntervalPre:   cfg.RewardIntervalPre,
		ManageFeePercent:    cfg.ManageFeePercent,
		ManageFeePercentPre: cfg.ManageFeePercentPre,
		RewardMaturity:      cfg.RewardMaturity,
		DelayInterval:       cfg.DelayInterval,
		ConfigChangeHeight:  int64(cfg.ConfigChangeHeight),
	}
	poolSvr, err := poolserver.NewPoolServer(&poolserver.ConfigPoolServer{
		DisableTLS:           cfg.DisableTLS,
		ListenersString:      cfg.Listeners,
		RPCUser:              cfg.PoolUser,
		RPCPass:              cfg.PoolPass,
		RPCLimitUser:         cfg.PoolLimitUser,
		RPCLimitPass:         cfg.PoolLimitPass,
		RPCMaxClients:        cfg.RPCMaxClients,
		RPCMaxWebsockets:     cfg.RPCMaxWebsockets,
		RPCMaxConcurrentReqs: cfg.RPCMaxConcurrentReqs,
		RPCCert:              cfg.PoolCert,
		RPCKey:               cfg.PoolKey,
		ExternalIPs:          cfg.ExternalIPs,
		DisableConnectToAbec: cfg.DisableConnectToAbec,
		UseWallet:            cfg.UseWallet,
		MaxErrors:            cfg.MaxErrors,
		DetailedShareInfo:    cfg.DetailedShareInfo,
	})
	if err != nil {
		return nil, err
	}
	poolSvr.SetCommonConfig(&commonCfg)
	poolSvr.SetRewardConfig(&rewardCfg)

	// Wait until chain client is ready.
	if !cfg.DisableConnectToAbec {
		for chainCli.chainClient == nil {
		}
	}
	if cfg.UseWallet {
		for walletClient.walletRPCClient == nil {
		}
	}

	poolSvr.SetChainClient(chainCli.chainClient)

	// Setup miner manager.
	minerMgr := minermgr.SetupMinerManger(!cfg.DisableAutoDifficultyAdjust, cfg.ExtraNonce2BitLen, cfg.SimNet)
	poolSvr.SetMinerManager(minerMgr)
	minerMgr.Subscribe(poolSvr.HandleMinerManagerNotification)

	// Setup reward manager.
	rewardMgr := rewardmgr.NewRewardManager(&rewardCfg, !cfg.DisableAutoAllocate, chainCli.chainClient)

	// Setup config manager.
	configMgr := configmgr.NewConfigManager(&rewardCfg)

	if !cfg.DisableConnectToAbec {
		chainCli.chainClient.Subscribe(minerMgr.HandleChainClientNotification)
		chainCli.chainClient.Subscribe(poolSvr.HandleChainClientNotification)
		chainCli.chainClient.Subscribe(rewardMgr.HandleChainClientNotification)
		chainCli.chainClient.Subscribe(configMgr.HandleChainClientNotification)
	}

	// Setup order manager
	var orderMgr *ordermgr.OrderManager
	if cfg.UseWallet {
		poolServerLog.Infof("Reward Manager: ManageFeePercent: %v, TxFeeSpecified: %v, ManageFeeAddressNumber: %d, RewardInterval: %d, RewardMaturity: %d", cfg.ManageFeePercent, cfg.TxFeeSpecified, len(cfg.ServerManageFeeAddrs), cfg.RewardInterval, cfg.RewardMaturity)
		orderMgr = ordermgr.NewOrderManager(&rewardCfg, cfg.ServerManageFeeAddrs, cfg.TxFeeSpecified,
			chainCli.chainClient, walletClient.walletRPCClient)
		if orderMgr == nil {
			return nil, errors.New("fail to create order manager")
		}
		poolSvr.SetOrderManager(orderMgr)
		// start two timer handler:
		// one for send transaction request to abewallet,
		// other one for checking the transaction status
		orderMgr.Start()
		walletClient.walletRPCClient.Subscribe(orderMgr.HandleWalletClientNotification)

		if !cfg.DisableConnectToAbec {
			chainCli.chainClient.Subscribe(orderMgr.HandleChainClientNotification)
		}
	}

	abethash := ethash.New(cfg.EthashConfig)
	poolSvr.SetEthash(abethash)
	minerMgr.SetEthash(abethash)
	if chainCli != nil && chainCli.chainClient != nil {
		chainCli.chainClient.SetEthash(abethash)
	}

	// Setup tcp socket server is needed
	var tcpSocketServer *poolserver.TCPSocketServer
	if cfg.EnableTCPSocket {
		tcpSocketServer, err = poolserver.NewTCPSocketServer(&poolserver.ConfigTCPSocketServer{
			DisableTLS:           cfg.DisableTCPSocketTLS,
			ListenersString:      cfg.ListenersTCPSocket,
			RPCMaxClients:        cfg.RPCMaxClients,
			RPCMaxWebsockets:     cfg.RPCMaxWebsockets,
			RPCMaxConcurrentReqs: cfg.RPCMaxConcurrentReqs,
			RPCCert:              cfg.PoolCert,
			RPCKey:               cfg.PoolKey,
			ExternalIPs:          cfg.ExternalIPs,
			DisableConnectToAbec: cfg.DisableConnectToAbec,
			UseWallet:            cfg.UseWallet,
			MaxErrors:            cfg.MaxErrors,
			DetailedShareInfo:    cfg.DetailedShareInfo,
		})
		if err != nil {
			return nil, err
		}

		poolSvr.SetTCPSocketServer(tcpSocketServer)
		tcpSocketServer.SetChainClient(chainCli.chainClient)
		tcpSocketServer.SetMinerManager(minerMgr)
		minerMgr.Subscribe(tcpSocketServer.HandleMinerManagerNotification)
		if !cfg.DisableConnectToAbec {
			chainCli.chainClient.Subscribe(tcpSocketServer.HandleChainClientNotification)
		}
		tcpSocketServer.SetEthash(abethash)
		tcpSocketServer.SetCommonConfig(&commonCfg)
		tcpSocketServer.SetRewardConfig(&rewardCfg)
	}

	ret := &server{
		poolRPCServer:   poolSvr,
		tcpSocketServer: tcpSocketServer,
		minerManager:    minerMgr,
		rewardManager:   rewardMgr,
		orderManager:    orderMgr,
	}
	return ret, nil
}

func (s *server) Start() {
	if s.poolRPCServer != nil {
		s.poolRPCServer.Start()
	}
	if s.tcpSocketServer != nil {
		s.tcpSocketServer.Start()
	}
}

func (s *server) Stop() {
	if s.poolRPCServer != nil {
		s.poolRPCServer.Stop()
	}
	if s.tcpSocketServer != nil {
		s.tcpSocketServer.Stop()
	}

	if s.orderManager != nil {
		s.orderManager.Stop()
	}
}

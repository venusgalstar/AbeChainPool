package main

import (
	"sync"

	"github.com/abesuite/abe-miningpool-server/walletclient"
)

type walletClient struct {
	cfg *config
	//enableAutoSend  bool
	walletRPCClient *walletclient.RPCClient
	handlerMu       sync.Mutex
}

func (wc *walletClient) setWalletClient(rpcClient *walletclient.RPCClient) {
	wc.handlerMu.Lock()
	defer wc.handlerMu.Unlock()

	wc.walletRPCClient = rpcClient
}

func (wc *walletClient) Stop() {
	rpcClient := wc.walletRPCClient
	if rpcClient != nil {
		poolLog.Warn("Stopping Abewallet Chain RPC client...")
		rpcClient.Stop()
		poolLog.Info("Chain RPC client shutdown complete")
	}
}

func createWalletClient(cfg *config) (*walletClient, error) {
	newClient := walletClient{
		cfg: cfg,
	}
	return &newClient, nil
}

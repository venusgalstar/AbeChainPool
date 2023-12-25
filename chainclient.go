package main

import (
	"sync"

	"github.com/abesuite/abe-miningpool-server/chainclient"
	"github.com/abesuite/abe-miningpool-server/consensus/ethash"
)

type chainClients struct {
	cfg         *config
	chainClient *chainclient.RPCClient
	handlerMu   sync.Mutex
}

func (server *chainClients) setChainClient(client *chainclient.RPCClient) {
	server.handlerMu.Lock()
	server.chainClient = client
	server.handlerMu.Unlock()
}

func (server *chainClients) setEthash(ethash *ethash.Ethash) {
	if server.chainClient != nil {
		server.chainClient.SetEthash(ethash)
	}
}

func (server *chainClients) Stop() {
	chainClient := server.chainClient
	if chainClient != nil {
		poolLog.Warn("Stopping Abec Chain RPC client...")
		chainClient.Stop()
		poolLog.Info("Abec Chain RPC client shutdown complete")
	}
}

func createChainClient(cfg *config) (*chainClients, error) {
	newClient := chainClients{
		cfg: cfg,
	}
	return &newClient, nil
}

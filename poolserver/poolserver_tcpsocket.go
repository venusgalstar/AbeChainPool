package poolserver

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"

	"github.com/abesuite/abe-miningpool-server/chainclient"
	"github.com/abesuite/abe-miningpool-server/consensus/ethash"
	"github.com/abesuite/abe-miningpool-server/minermgr"
	"github.com/abesuite/abe-miningpool-server/model"
)

// TCPSocketServer provides a concurrent safe RPC server to a chain server.
type TCPSocketServer struct {
	started                int32
	shutdown               int32
	cfg                    ConfigTCPSocketServer
	commonCfg              *CommonConfig
	rewardCfg              *model.RewardConfig
	ntfnMgr                AbstractNtfnManager
	numClients             int32
	statusLines            map[int]string
	statusLock             sync.RWMutex
	wg                     sync.WaitGroup
	requestProcessShutdown chan struct{}
	quit                   chan int

	minerManager *minermgr.MinerManager
	chainClient  *chainclient.RPCClient

	ethash          *ethash.Ethash
	abecBackendNode string
}

// ConfigTCPSocketServer is a descriptor containing the RPC server configuration.
type ConfigTCPSocketServer struct {
	DisableTLS bool
	// ListenersString an array that contains ip address and port for generating
	// listeners later
	ListenersString []string

	// Listeners defines a slice of listeners for which the RPC server will
	// take ownership of and accept connections.  Since the RPC server takes
	// ownership of these listeners, they will be closed when the RPC server
	// is stopped.
	Listeners []net.Listener

	// StartupTime is the unix timestamp for when the server that is hosting
	// the RPC server started.
	StartupTime int64

	RPCMaxClients        int
	RPCMaxWebsockets     int
	RPCMaxConcurrentReqs int
	RPCKey               string
	RPCCert              string
	ExternalIPs          []string

	DisableConnectToAbec bool
	UseWallet            bool
	MaxErrors            int
	DetailedShareInfo    bool
}

func (svr *TCPSocketServer) SetMinerManager(mgr *minermgr.MinerManager) {
	svr.minerManager = mgr
}

func (svr *TCPSocketServer) SetChainClient(cli *chainclient.RPCClient) {
	svr.chainClient = cli
}

func (svr *TCPSocketServer) SetEthash(ethash *ethash.Ethash) {
	svr.ethash = ethash
}

func (svr *TCPSocketServer) SetCommonConfig(cfg *CommonConfig) {
	svr.commonCfg = cfg
}

func (svr *TCPSocketServer) SetRewardConfig(cfg *model.RewardConfig) {
	svr.rewardCfg = cfg
}

// NewTCPSocketServer returns a new instance of the TCPSocketServer struct.
func NewTCPSocketServer(config *ConfigTCPSocketServer) (*TCPSocketServer, error) {
	rpcListeners, err := setupRPCListeners(config.ListenersString, config.RPCKey, config.RPCCert, config.ExternalIPs,
		config.DisableTLS)
	if err != nil {
		return nil, err
	}
	if len(rpcListeners) == 0 {
		return nil, errors.New("Pool RPCS: No valid listen address")
	}
	config.Listeners = rpcListeners
	rpc := TCPSocketServer{
		cfg:                    *config,
		statusLines:            make(map[int]string),
		requestProcessShutdown: make(chan struct{}),
		quit:                   make(chan int),
		abecBackendNode:        "",
	}

	rpc.ntfnMgr = newTCPSocketNotificationManager(&rpc)

	return &rpc, nil
}

// Start is used by server.go to start the rpc listener.
func (svr *TCPSocketServer) Start() {
	if atomic.AddInt32(&svr.started, 1) != 1 {
		return
	}

	log.Trace("Starting pool TCP socket RPC server...")

	for _, listener := range svr.cfg.Listeners {
		svr.wg.Add(1)
		go func(listener net.Listener) {
			tlsState := "on"
			if svr.cfg.DisableTLS {
				tlsState = "off"
			}
			log.Infof("Pool TCP socket RPC server listening on %s (TLS %s)", listener.Addr(), tlsState)
		out:
			for {
				conn, err := listener.Accept()
				if err != nil {
					select {
					case <-svr.quit:
						break out
					default:
					}
					//log.Errorf("Pool TCP socket RPC listener err: %v", err)
					continue
				}
				if isUndesiredIP(conn.RemoteAddr().String(), svr.commonCfg.Blacklist, svr.commonCfg.Whitelist) {
					conn.Close()
					continue
				}
				go svr.TCPSocketHandler(conn, conn.RemoteAddr().String())
			}
			log.Tracef("Pool TCP socket RPC listener done for %s", listener.Addr())
			svr.wg.Done()
		}(listener)
	}

	svr.ntfnMgr.Start()
}

// Stop is used by server.go to stop the pool rpc listener.
func (svr *TCPSocketServer) Stop() error {
	if atomic.AddInt32(&svr.shutdown, 1) != 1 {
		log.Infof("Pool TCP socket RPC server is already in the process of shutting down")
		return nil
	}
	log.Warnf("Pool TCP socket RPC server shutting down...")
	for _, listener := range svr.cfg.Listeners {
		err := listener.Close()
		if err != nil {
			log.Errorf("Problem shutting down pool TCP socket RPC server: %v", err)
			return err
		}
	}
	svr.ntfnMgr.Shutdown()
	svr.ntfnMgr.WaitForShutdown()
	close(svr.quit)
	svr.wg.Wait()
	log.Infof("Pool TCP socket RPC server shutdown complete")
	return nil
}

// incrementClients adds one to the number of connected RPC clients.  Note
// this only applies to standard clients.  Websocket clients have their own
// limits and are tracked separately.
//
// This function is safe for concurrent access.
func (svr *TCPSocketServer) incrementClients() {
	atomic.AddInt32(&svr.numClients, 1)
}

// decrementClients subtracts one from the number of connected RPC clients.
// Note this only applies to standard clients.  Websocket clients have their own
// limits and are tracked separately.
//
// This function is safe for concurrent access.
func (svr *TCPSocketServer) decrementClients() {
	atomic.AddInt32(&svr.numClients, -1)
}

// TCPSocketHandler handles a new websocket client by creating a new wsClient,
// starting it, and blocking until the connection closes.  Since it blocks, it
// must be run in a separate goroutine.  It should be invoked from the websocket
// server handler which runs each new connection in a new goroutine thereby
// satisfying the requirement.
func (svr *TCPSocketServer) TCPSocketHandler(conn net.Conn, remoteAddr string) {

	// Clear the read deadline that was set before the websocket hijacked
	// the connection.
	//conn.SetReadDeadline(timeZeroVal)

	// Limit max number of websocket clients.
	log.Infof("New tcp socket client %s", remoteAddr)
	if svr.ntfnMgr.NumClients()+1 > svr.cfg.RPCMaxWebsockets {
		log.Infof("Max tcp socket clients exceeded [%d] - "+
			"disconnecting client %s", svr.cfg.RPCMaxWebsockets,
			remoteAddr)
		conn.Close()
		return
	}

	// Create a new websocket client to handle the new websocket connection
	// and wait for it to shutdown.  Once it has shutdown (and hence
	// disconnected), remove it and any notifications it registered for.
	client, err := newTCPSocketClient(svr, conn, remoteAddr)
	if err != nil {
		log.Errorf("Failed to serve tcp socket client %s: %v", remoteAddr, err)
		conn.Close()
		return
	}
	svr.ntfnMgr.AddClient(client)
	client.Start()
	client.WaitForShutdown()
	svr.ntfnMgr.RemoveClient(client)
	log.Infof("Disconnected tcp socket client %s", remoteAddr)
}

// Start begins processing input and output messages.
func (c *TCPSocketClient) Start() {
	log.Tracef("Starting tcp socket client %s", c.addr)

	// Start processing input and output.
	c.wg.Add(3)
	go c.inHandler()
	go c.notificationQueueHandler()
	go c.outHandler()
}

// WaitForShutdown blocks until the websocket client goroutines are stopped
// and the connection is closed.
func (c *TCPSocketClient) WaitForShutdown() {
	c.wg.Wait()
}

// HandleChainClientNotification handles notifications from chain client.  It does
// things such as notify template changed and so on.
func (svr *TCPSocketServer) HandleChainClientNotification(notification *chainclient.Notification) {
	switch notification.Type {

	}
}

// HandleMinerManagerNotification handles notifications from miner manager. It does
// things such as notify share difficulty changed and so on.
func (svr *TCPSocketServer) HandleMinerManagerNotification(notification *minermgr.Notification) {
	switch notification.Type {

	case minermgr.NTNewJobReady:
		newJob, ok := notification.Data.(*model.JobTemplate)
		if !ok {
			log.Warnf("The NTNewJobReady notification is not a job template!")
			break
		}

		svr.ntfnMgr.NotifyNewJob(newJob)

	case minermgr.NTEpochChange:
		newEpoch, ok := notification.Data.(int64)
		if !ok {
			log.Warnf("The NTEpochChange notification is not a int64 value!")
			break
		}

		svr.ntfnMgr.NotifyNewEpoch(newEpoch)
	}

}

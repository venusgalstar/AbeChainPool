package poolserver

import (
	"net"

	"github.com/abesuite/abe-miningpool-server/chainclient"
	"github.com/abesuite/abe-miningpool-server/minermgr"
	"github.com/abesuite/abe-miningpool-server/model"

	"gorm.io/gorm"
)

// abstractCommandHandler describes a callback function used to handle a specific
// command.
type abstractCommandHandler func(AbstractSocketClient, interface{}) (interface{}, error)

// abstractHandlers maps RPC command strings to appropriate websocket handler
// functions.  This is set by init because help references wsHandlers and thus
// causes a dependency loop.
var abstractHandlers map[string]abstractCommandHandler
var abstractHandlersBeforeInit = map[string]abstractCommandHandler{
	// EthereumStratum 2.0
	"mining.hello":     handleHello,
	"mining.subscribe": handleSubscribe,
	"mining.authorize": handleAuthorize,
	"mining.submit":    handleSubmit,
	"mining.get_job":   handleGetJob,
	"mining.noop":      handleNoop,
	"mining.hashrate":  handleHashRate,

	// E9 Miner
	"eth_submitLogin": handleSubmitLogin,
}

type AbstractSocketClient interface {
	Type() string
	RemoteAddr() string
	DisconnectGracefully()
	SetAbecBackendNode(desc string)
	GetAbecBackendNode() string
	GetMaxErrors() int
	GetNtfnManager() AbstractNtfnManager
	GetChainClient() *chainclient.RPCClient
	QuitChan() chan struct{}
	GetMinerManager() *minermgr.MinerManager
	GetDB() *gorm.DB
	GetRewardInterval() int
	GetRewardIntervalPre() int
	GetRewardIntervalChangeHeight() int64
	DetailedShareInfo() bool
	GetAgentBlacklist() []string
	GetAgentWhitelist() []string
	GetBlacklist() []*net.IPNet
	GetWhitelist() []*net.IPNet
	GetAdminBlacklist() []*net.IPNet
	GetAdminWhitelist() []*net.IPNet
}

type AbstractNtfnManager interface {
	Start()
	Shutdown()
	WaitForShutdown()
	NumClients() int
	AddClient(wsc AbstractSocketClient)
	RemoveClient(wsc AbstractSocketClient)
	CloseAndDeleteStallQuit(quit chan struct{})
	NotifyNewJob(newJob *model.JobTemplate)
	NotifyNewEpoch(newEpoch int64)
	AddNotifyNewJobClient(wsc AbstractSocketClient)
	notifySet(notifyClients map[chan struct{}]AbstractSocketClient, epoch int, target string, algo string,
		extraNonce1 string, extraNonceBitsNum int)
	notifyNewJob(clients map[chan struct{}]AbstractSocketClient, job *model.JobTemplate)
	notifyDifficultyV1(notifyClients map[chan struct{}]AbstractSocketClient, target string)
	notifyNewJobV1(clients map[chan struct{}]AbstractSocketClient, job *model.JobTemplate)
	notifyNewJobE9(clients map[chan struct{}]AbstractSocketClient, job *model.JobTemplate)
}

func init() {
	abstractHandlers = abstractHandlersBeforeInit
}

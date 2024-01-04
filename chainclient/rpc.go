package chainclient

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/abesuite/abe-miningpool-server/abejson"
	"github.com/abesuite/abe-miningpool-server/chaincfg"
	"github.com/abesuite/abe-miningpool-server/consensus/ethash"
	"github.com/abesuite/abe-miningpool-server/model"
	"github.com/abesuite/abe-miningpool-server/mylog"
	"github.com/abesuite/abe-miningpool-server/rpcclient"
	"github.com/abesuite/abe-miningpool-server/utils"
)

// NotificationType represents the type of a notification message.
type NotificationType int

// NotificationCallback is used for a caller to provide a callback for
// notifications about various events.
type NotificationCallback func(*Notification)

// Constants for the type of a notification message.
const (
	// NTBlockTemplateChanged indicates a new block template is sent from abec
	NTBlockTemplateChanged NotificationType = iota

	// NTBlockAccepted indicates the candidate block is accepted by abec
	NTBlockAccepted

	// NTBlockRejected indicates the candidate block is rejected by abec
	NTBlockRejected

	// NTBlockConnected indicate the candidate block is connected
	NTBlockConnected

	// NTBlockDisconnected indicate the candidate block is disconnected
	NTBlockDisconnected
)

// notificationTypeStrings is a map of notification types back to their constant
// names for pretty printing.
var notificationTypeStrings = map[NotificationType]string{
	NTBlockTemplateChanged: "NTBlockTemplateChanged",
	NTBlockAccepted:        "NTBlockAccepted",
	NTBlockRejected:        "NTBlockRejected",
	NTBlockConnected:       "NTBlockConnected",
	NTBlockDisconnected:    "NTBlockDisconnected",
}

var (
	getTemplateInterval = time.Second * 20
)

// Notification types.
type (
	// ClientConnected is a notification for when a client connection is
	// opened or reestablished to the chain server.
	ClientConnected struct{}
)

// String returns the NotificationType in human-readable form.
func (n NotificationType) String() string {
	if s, ok := notificationTypeStrings[n]; ok {
		return s
	}
	return fmt.Sprintf("Unknown Notification Type (%d)", int(n))
}

// Notification defines notification that is sent to the caller via the callback
// function provided during the call to New and consists of a notification type
// as well as associated data that depends on the type as follows:
//   - NTBlockTemplateChanged:     *template.BlockTemplate
type Notification struct {
	Type NotificationType
	Data interface{}
}

// Subscribe to notifications. Registers a callback to be executed
// when various events take place.
func (c *RPCClient) Subscribe(callback NotificationCallback) {
	c.notificationsLock.Lock()
	c.notifications = append(c.notifications, callback)
	c.notificationsLock.Unlock()
}

// sendNotification sends a notification with the passed type and data if the
// caller requested notifications by providing a callback function in the call
// to New.
func (c *RPCClient) sendNotification(typ NotificationType, data interface{}) {
	// Generate and send the notification.
	n := Notification{Type: typ, Data: data}
	c.notificationsLock.RLock()
	for _, callback := range c.notifications {
		callback(&n)
	}
	c.notificationsLock.RUnlock()
}

// RPCClient represents a persistent client connection to an abec RPC server
type RPCClient struct {
	*rpcclient.Client
	connConfig        *rpcclient.ConnConfig
	reconnectAttempts int

	enqueueNotification chan interface{}
	dequeueNotification chan interface{}

	currentBlockTemplate *model.BlockTemplate
	useAddrFromAbec      bool
	miningAddrs          []string

	quit                          chan struct{}
	blockConnectedNotification    chan *model.BlockNotification
	blockDisconnectedNotification chan *model.BlockNotification
	wg                            sync.WaitGroup
	started                       bool
	quitMtx                       sync.Mutex
	templateMtx                   sync.Mutex

	// The notifications field stores a slice of callbacks to be executed on
	// certain events.
	notificationsLock sync.RWMutex
	notifications     []NotificationCallback

	submitChan chan *model.BlockSubmitted

	ethash            *ethash.Ethash
	ethashInitialized bool
}

func (c *RPCClient) SetEthash(ethash *ethash.Ethash) {
	c.ethash = ethash
}

// NewRPCClient creates a client connection to the server described by the
// connect string. If disableTLS is false, the remote RPC certificate must be
// provided in the certs slice. The connection is not established immediately,
// but must be done using the Start method.
func NewRPCClient(connect, user, pass string, certs []byte, disableTLS bool,
	reconnectAttempts int, useAddrFromAbec bool, miningAddrs []string) (*RPCClient, error) {

	if reconnectAttempts < 0 {
		return nil, errors.New("reconnectAttempts must be positive")
	}

	client := &RPCClient{
		connConfig: &rpcclient.ConnConfig{
			Host:                 connect,
			Endpoint:             "ws",
			User:                 user,
			Pass:                 pass,
			Certificates:         certs,
			DisableAutoReconnect: false,
			DisableConnectOnNew:  true,
			DisableTLS:           disableTLS,
			Alias:                "Abec",
		},
		reconnectAttempts:             reconnectAttempts,
		enqueueNotification:           make(chan interface{}),
		dequeueNotification:           make(chan interface{}),
		quit:                          make(chan struct{}),
		blockConnectedNotification:    make(chan *model.BlockNotification, 5),
		blockDisconnectedNotification: make(chan *model.BlockNotification, 5),
		submitChan:                    make(chan *model.BlockSubmitted, 5),
		useAddrFromAbec:               useAddrFromAbec,
		miningAddrs:                   miningAddrs,
	}
	ntfnCallbacks := &rpcclient.NotificationHandlers{
		OnClientConnected:      client.onClientConnect,
		OnBlockAbeConnected:    client.onBlockConnect,
		OnBlockAbeDisconnected: client.onBlockDisconnect,
	}
	rpcClient, err := rpcclient.New(client.connConfig, ntfnCallbacks)
	if err != nil {
		return nil, err
	}
	client.Client = rpcClient
	return client, nil
}

// BackEnd returns the name of the driver.
func (c *RPCClient) BackEnd() string {
	return "abec"
}

// Start attempts to establish a client connection with the remote server.
// If successful, handler goroutines are started to process notifications
// sent by the server. After a limited number of connection attempts, this
// function gives up, and therefore will not block forever waiting for the
// connection to be established to a server that may not exist.
func (c *RPCClient) Start() error {
	err := c.Connect(c.reconnectAttempts)
	if err != nil {
		return err
	}

	c.quitMtx.Lock()
	c.started = true
	c.quitMtx.Unlock()

	c.wg.Add(1)
	go c.handler()
	return nil
}

// Stop disconnects the client and signals the shutdown of all goroutines
// started by Start.
func (c *RPCClient) Stop() {
	c.quitMtx.Lock()
	select {
	case <-c.quit:
	default:
		close(c.quit)
		c.Client.Shutdown()

		if !c.started {
			close(c.dequeueNotification)
		}
	}
	c.quitMtx.Unlock()
	log.Trace("Chain client done")
}

// WaitForShutdown blocks until both the client has finished disconnecting
// and all handlers have exited.
func (c *RPCClient) WaitForShutdown() {
	c.Client.WaitForShutdown()
	c.wg.Wait()
}

// Notifications returns a channel of parsed notifications sent by the remote
// abec RPC server. This channel must be continually read or the process
// may abort for running out memory, as unread notifications are queued for
// later reads.
func (c *RPCClient) Notifications() <-chan interface{} {
	return c.dequeueNotification
}

func (c *RPCClient) onClientConnect() {
	select {
	case c.enqueueNotification <- ClientConnected{}:
	case <-c.quit:
	}
	c.wg.Add(3)

	go c.templateHandler()
	go c.notificationHandler()
	go c.submitHandler()
	c.getBackendVersion()
}

func (c *RPCClient) getBackendVersion() {
	versionMap, err := c.Client.Version(10)
	if err != nil {
		log.Infof("Unable to get backend abec version: %v", err)
		return
	}
	backendVersion, ok := versionMap["abec"]
	if !ok {
		log.Error("Unable to get backend abec version: version not found")
		return
	}
	chaincfg.AbecBackendVersion = backendVersion.VersionString
	log.Infof("Backend abec version: %v", chaincfg.AbecBackendVersion)
}

func (c *RPCClient) onBlockConnect(hash *utils.Hash, height int32, t time.Time) {
	if hash == nil {
		log.Errorf("hash in onBlockConnected is nil")
		return
	}
	ntfn := &model.BlockNotification{
		BlockHash: *hash,
		Height:    height,
		Time:      t,
	}
	c.blockConnectedNotification <- ntfn
}

func (c *RPCClient) onBlockDisconnect(hash *utils.Hash, height int32, t time.Time) {
	if hash == nil {
		log.Errorf("hash in onBlockDisconnected is nil")
		return
	}
	ntfn := &model.BlockNotification{
		BlockHash: *hash,
		Height:    height,
		Time:      t,
	}
	c.blockDisconnectedNotification <- ntfn
}

// handler maintains a queue of notifications.
func (c *RPCClient) handler() {

	var notifications []interface{}
	enqueue := c.enqueueNotification
	var dequeue chan interface{}
	var next interface{}
out:
	for {
		select {
		case n, ok := <-enqueue:
			if !ok {
				// If no notifications are queued for handling,
				// the queue is finished.
				if len(notifications) == 0 {
					break out
				}
				// nil channel so no more reads can occur.
				enqueue = nil
				continue
			}
			if len(notifications) == 0 {
				next = n
				dequeue = c.dequeueNotification
			}
			notifications = append(notifications, n)

		case dequeue <- next:

			notifications[0] = nil
			notifications = notifications[1:]
			if len(notifications) != 0 {
				next = notifications[0]
			} else {
				// If no more notifications can be enqueued, the
				// queue is finished.
				if enqueue == nil {
					break out
				}
				dequeue = nil
			}

		case <-c.quit:
			break out
		}
	}

	c.Stop()
	close(c.dequeueNotification)
	c.wg.Done()
}

// POSTClient creates the equivalent HTTP POST rpcclient.Client.
func (c *RPCClient) POSTClient() (*rpcclient.Client, error) {
	configCopy := *c.connConfig
	configCopy.HTTPPostMode = true
	return rpcclient.New(&configCopy, nil)
}

// GetTemplate get the current block template
func (c *RPCClient) GetTemplate() *model.BlockTemplate {
	c.templateMtx.Lock()
	ret := c.currentBlockTemplate
	c.templateMtx.Unlock()
	return ret
}

// SetTemplate set the current block template
func (c *RPCClient) SetTemplate(tmp *model.BlockTemplate) {
	c.templateMtx.Lock()
	c.currentBlockTemplate = tmp
	c.templateMtx.Unlock()
}

// isSameTemplate compare two block template and check if they are the same (except timestamp)
func isSameTemplate(newTemplate *model.BlockTemplate, currentTemplate *model.BlockTemplate) bool {
	if newTemplate == nil && currentTemplate == nil {
		return true
	}
	if newTemplate == nil || currentTemplate == nil {
		return false
	}
	if newTemplate.Version != currentTemplate.Version {
		return false
	}
	if newTemplate.Bits != currentTemplate.Bits {
		return false
	}
	if newTemplate.Height != currentTemplate.Height {
		return false
	}
	if newTemplate.PreviousHash != currentTemplate.PreviousHash {
		return false
	}
	if len(newTemplate.Transactions) != len(currentTemplate.Transactions) {
		return false
	}
	for i := 0; i < len(newTemplate.Transactions); i++ {
		if newTemplate.Transactions[i].TxHash != currentTemplate.Transactions[i].TxHash {
			return false
		}
	}
	if newTemplate.CoinbaseTxn.TxHash != currentTemplate.CoinbaseTxn.TxHash ||
		newTemplate.CoinbaseTxn.ExtraNonceLen != currentTemplate.CoinbaseTxn.ExtraNonceLen ||
		newTemplate.CoinbaseTxn.ExtraNonceOffset != currentTemplate.CoinbaseTxn.ExtraNonceOffset {
		return false
	}
	currentTimestamp := currentTemplate.CurTime
	newTimeStamp := newTemplate.CurTime
	if newTimeStamp-currentTimestamp > 60 {
		log.Infof("Block template timestamp expired, updating...")
		return false
	}
	return true
}

func (c *RPCClient) getBlockTemplateDesc(blockTemplate *model.BlockTemplate) string {
	if blockTemplate == nil {
		return ""
	}
	res := fmt.Sprintf("Height: %v, Previous Hash: %v, Timestamp: %v, Version: %v, Bits: %v, Fee: %v, TX Count: %v",
		blockTemplate.Height, blockTemplate.PreviousHash, blockTemplate.CurTime, blockTemplate.Version, blockTemplate.Bits,
		blockTemplate.CoinbaseTxn.Fee, len(blockTemplate.Transactions)+1)
	if log.Level() <= mylog.LevelDebug {
		res += " ( " + blockTemplate.CoinbaseTxn.TxHash
		for _, tx := range blockTemplate.Transactions {
			res += ", " + tx.TxHash
		}
		res += " )"
	}
	return res
}

// getAndSetTemplate gets the block template from abec
func (c *RPCClient) getAndSetTemplate() bool {
	templateRequest := abejson.TemplateRequest{}
	if !c.useAddrFromAbec {
		templateRequest.Capabilities = []string{"useownaddr"}
		rand.Seed(time.Now().UnixNano())
		idx := rand.Intn(len(c.miningAddrs))
		log.Debugf("Choose address at index %v", idx)
		templateRequest.MiningAddr = c.miningAddrs[idx]
	}
	tmp, err := c.Client.GetBlockTemplate(&templateRequest, 9)
	if err != nil {
		// log.Errorf("Unable to get block template: %v", err)
		return false
	}
	var newBlockTemplate model.BlockTemplate
	if tmp != nil {
		newBlockTemplate = model.BlockTemplate{
			Version:      tmp.Version,
			Bits:         tmp.Bits,
			CurTime:      tmp.CurTime,
			Height:       tmp.Height,
			PreviousHash: tmp.PreviousHash,
			Transactions: tmp.Transactions,
			CoinbaseTxn:  tmp.CoinbaseTxn,
		}
		currentBlockTemplate := c.GetTemplate()
		templateDesc := c.getBlockTemplateDesc(&newBlockTemplate)
		if isSameTemplate(&newBlockTemplate, currentBlockTemplate) {
			log.Infof("Get new block template from abec: %v, same as the previous template.", templateDesc)
			// return false
		} else {
			log.Infof("Get new block template from abec: %v", templateDesc)
		}
		c.SetTemplate(&newBlockTemplate)
		return true
	}
	return false
}

// templateHandler gets the block template from abec periodically
// and notify the miners if template changes
func (c *RPCClient) templateHandler() {
	log.Info("Fetching new block template...")
	c.getAndSetTemplate()
	if c.currentBlockTemplate != nil {
		c.sendNotification(NTBlockTemplateChanged, c.GetTemplate())
	}

	c.InitializeDataset()

	getTemplateTicker := time.NewTicker(getTemplateInterval)
	defer getTemplateTicker.Stop()
out:
	for {
		select {
		case <-getTemplateTicker.C:
			isChanged := c.getAndSetTemplate()
			if isChanged {
				c.sendNotification(NTBlockTemplateChanged, c.GetTemplate())
			}
			if !c.ethashInitialized {
				c.InitializeDataset()
			}

		case blockInfo := <-c.blockConnectedNotification:
			log.Infof("Block %v at height %v connected, fetching new block template...", blockInfo.BlockHash.String(), blockInfo.Height)
			c.sendNotification(NTBlockConnected, blockInfo)
			isChanged := c.getAndSetTemplate()
			if !isChanged {
				log.Warnf("New block notification comes but no available new template, " +
					"this usually occurs when more than one block connected notification comes in a short time")
				continue
			}
			c.sendNotification(NTBlockTemplateChanged, c.GetTemplate())
			getTemplateTicker.Reset(getTemplateInterval)

		case <-c.quit:
			break out

		case <-c.DisconnectedChan():
			break out
		}
	}
	c.wg.Done()
	log.Trace("Template handler done")
}

// notificationHandler handle the notifications except blockConnectedNotification from abec.
func (c *RPCClient) notificationHandler() {
out:
	for {
		select {
		case blockInfo := <-c.blockDisconnectedNotification:
			log.Infof("Block %v at height %v disconnected", blockInfo.BlockHash.String(), blockInfo.Height)
			c.sendNotification(NTBlockDisconnected, blockInfo)

		case <-c.quit:
			break out

		case <-c.DisconnectedChan():
			break out
		}
	}
	c.wg.Done()
	log.Trace("RPCClient (Abec) notification handler done")
}

func (c *RPCClient) InitializeDataset() {
	if c.ethash != nil && c.currentBlockTemplate != nil {
		if c.currentBlockTemplate.Height < int64(chaincfg.ActiveNetParams.BlockHeightEthashPoW) {
			if c.ethash.IsVerifyByFullDAG() {
				log.Infof("Block height does not reach the update height, preparing dataset for epoch 0 in advance...")
				c.ethash.PrepareDataset(0)
			} else {
				log.Infof("Block height does not reach the update height, preparing verification cache for epoch 0 in advance...")
				c.ethash.PrepareCache(0)
			}
		} else {
			currentEpoch := ethash.CalculateEpoch(int32(c.currentBlockTemplate.Height))
			if c.ethash.IsVerifyByFullDAG() {
				log.Infof("Preparing dataset for epoch %v", currentEpoch)
				c.ethash.PrepareDataset(currentEpoch)
			} else {
				log.Infof("Preparing verification cache for epoch %v", currentEpoch)
				c.ethash.PrepareCache(currentEpoch)
			}
		}
	}
	c.ethashInitialized = true
}

func (c *RPCClient) SubmitBlock(candidateBlock *model.BlockSubmitted) {
	c.submitChan <- candidateBlock
}

func (c *RPCClient) submitHandler() {
out:
	for {
		select {
		case submitContent := <-c.submitChan:
			block := submitContent.SimplifiedBlock
			blockHash := submitContent.BlockHash
			if block == nil {
				log.Errorf("Internal error: submit content should not be nil!")
				continue
			}
			log.Infof("Submitting block to abec... ( Height: %v, PrevBlock: %v, TX Count: %v, Version: %v, Timestamp: %v, Merkle: %v, Bits: %v, Nonce: %v )",
				block.Header.Height, block.Header.PrevBlock, len(block.Transactions), block.Header.Version, block.Header.Timestamp,
				block.Header.MerkleRoot, block.Header.Bits, block.Header.Nonce)
			err := c.SubmitSimplifiedBlock(block, nil, 20)
			if err != nil {
				log.Infof("Submit %v fails: %v", blockHash.String(), err.Error())
				blockNotification := model.BlockNotification{
					BlockHash: blockHash,
					Info:      err.Error(),
				}
				c.sendNotification(NTBlockRejected, &blockNotification)
			} else {
				log.Infof("Abec accepts the block %v", blockHash.String())
				blockNotification := model.BlockNotification{
					BlockHash: blockHash,
					Info:      "",
				}
				c.sendNotification(NTBlockAccepted, &blockNotification)
			}
		case <-c.quit:
			break out
		case <-c.DisconnectedChan():
			break out
		}
	}
	c.wg.Done()
	log.Trace("Submit handler done")
}

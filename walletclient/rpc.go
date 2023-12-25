package walletclient

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/abesuite/abe-miningpool-server/abejson"
	"github.com/abesuite/abe-miningpool-server/chaincfg"
	"github.com/abesuite/abe-miningpool-server/dal/do"
	"github.com/abesuite/abe-miningpool-server/rpcclient"
	"github.com/abesuite/abe-miningpool-server/utils"
)

const CheckHeightDiff = 400

// NotificationType represents the type of a notification message.
type NotificationType int

// NotificationCallback is used for a caller to provide a callback for
// notifications about various events.
type NotificationCallback func(*Notification)

const (
	NTTxAccepted NotificationType = iota
	NTTxRollback
	NTTxInvalid
)

// notificationTypeStrings is a map of notification types back to their constant
// names for pretty printing.
var notificationTypeStrings = map[NotificationType]string{
	NTTxAccepted: "NTTxAccepted",
	NTTxRollback: "NTTxRollback",
	NTTxInvalid:  "NTTxInvalid",
}

type TransactionInfo struct {
	TxHash string
	Height int32
}

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
//func (c *RPCClient) Subscribe(callback NotificationCallback) {
//	c.notificationsLock.Lock()
//	c.notifications = append(c.notifications, callback)
//	c.notificationsLock.Unlock()
//}

// sendNotification sends a notification with the passed type and data if the
// caller requested notifications by providing a callback function in the call
// to New.
//func (c *RPCClient) sendNotification(typ NotificationType, data interface{}) {
//	// Generate and send the notification.
//	n := Notification{Type: typ, Data: data}
//	c.notificationsLock.RLock()
//	for _, callback := range c.notifications {
//		callback(&n)
//	}
//	c.notificationsLock.RUnlock()
//}

// RPCClient represents a persistent client connection to an abec RPC server
type RPCClient struct {
	*rpcclient.Client
	connConfig        *rpcclient.ConnConfig
	reconnectAttempts int

	enqueueNotification chan interface{}
	dequeueNotification chan interface{}

	//currentBlockTemplate *model.BlockTemplate
	//useAddrFromAbec      bool
	//miningAddrs          []string

	quit                   chan struct{}
	txAcceptedNotification chan *TransactionInfo
	txRollbackNotification chan *TransactionInfo
	txInvalidNotification  chan *TransactionInfo
	wg                     sync.WaitGroup
	started                bool
	quitMtx                sync.Mutex
	//templateMtx                   sync.Mutex

	notificationsLock sync.RWMutex
	notifications     []NotificationCallback
	walletPrivPass    string

	taskChan  chan *abejson.BatchOrderInfo // handle orders
	checkChan chan *abejson.TxStatusInfo   // handle transaction check
}

func (c *RPCClient) CheckChan() chan *abejson.TxStatusInfo {
	return c.checkChan
}

func (c *RPCClient) SetWalletPrivPass(walletPrivPass string) bool {
	_, err := c.WalletUnlock(&abejson.WalletUnlockCmd{
		Passphrase: walletPrivPass,
		Timeout:    2,
	}, 10)
	if err != nil {
		return false
	}
	log.Debugf("Successful to unlock the wallet! It's time to lock the wallet now...")
	_, err = c.WalletLock(&abejson.WalletLockCmd{}, 10)
	if err != nil {
		log.Infof("Fail to lock the wallet, disable auto sending")
	}
	log.Debugf("Successful to lock the wallet!")
	log.Infof("Enable automatic allocating with wallet")
	c.walletPrivPass = walletPrivPass
	c.wg.Add(1)
	go c.allocateHandler()
	return true
}

// NewRPCClient creates a client connection to the server described by the
// connect string. If disableTLS is false, the remote RPC certificate must be
// provided in the certs slice. The connection is not established immediately,
// but must be done using the Start method.
func NewRPCClient(connect, user, pass string, certs []byte, disableTLS bool,
	reconnectAttempts int) (*RPCClient, error) {

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
			Alias:                "Abewallet",
		},
		reconnectAttempts:   reconnectAttempts,
		enqueueNotification: make(chan interface{}),
		dequeueNotification: make(chan interface{}),
		quit:                make(chan struct{}),

		taskChan:               make(chan *abejson.BatchOrderInfo, 10),
		checkChan:              make(chan *abejson.TxStatusInfo, 50),
		txAcceptedNotification: make(chan *TransactionInfo, 5),
		txRollbackNotification: make(chan *TransactionInfo, 5),
		txInvalidNotification:  make(chan *TransactionInfo, 5),
		//submitChan:                    make(chan *model.BlockSubmitted, 5),
		//useAddrFromAbec:               useAddrFromAbec,
		//miningAddrs:                   miningAddrs,
	}
	ntfnCallbacks := &rpcclient.NotificationHandlers{
		OnClientConnected: client.onClientConnect,
		OnTxAccepted:      client.onTxAccepted,
		OnTxRollback:      client.onTxRollback,
		OnTxInvalid:       client.onTxInvalid,
		//OnBlockAbeConnected:    client.onBlockConnect,
		//OnBlockAbeDisconnected: client.onBlockDisconnect,
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
	return "abewallet"
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
// abewallet RPC server. This channel must be continually read or the process
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
	c.wg.Add(2)

	go c.notificationHandler()
	go c.checkHandler()
	//c.getBackendVersion()
}

//func (c *RPCClient) getBackendVersion() {
//	versionMap, err := c.Client.Version(10)
//	if err != nil {
//		log.Infof("Unable to get backend abewallet version: %v", err)
//		return
//	}
//	backendVersion, ok := versionMap["abewallet"]
//	if !ok {
//		log.Info("Unable to get backend abewallet version: version not found")
//		return
//	}
//	chaincfg.AbewalletBackendVersion = backendVersion.VersionString
//	log.Infof("Backend abewallet version: %v", chaincfg.AbewalletBackendVersion)
//}

//func (c *RPCClient) onBlockConnect(hash *utils.Hash, height int32, t time.Time) {
//	if hash == nil {
//		log.Errorf("hash in onBlockConnected is nil")
//		return
//	}
//	ntfn := &model.BlockNotification{
//		BlockHash: *hash,
//		Height:    height,
//		Time:      t,
//	}
//	c.blockConnectedNotification <- ntfn
//}
//
//func (c *RPCClient) onBlockDisconnect(hash *utils.Hash, height int32, t time.Time) {
//	if hash == nil {
//		log.Errorf("hash in onBlockDisconnected is nil")
//		return
//	}
//	ntfn := &model.BlockNotification{
//		BlockHash: *hash,
//		Height:    height,
//		Time:      t,
//	}
//	c.blockDisconnectedNotification <- ntfn
//}

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
func (c *RPCClient) GetTaskChan() chan<- *abejson.BatchOrderInfo {
	return c.taskChan
}
func (c *RPCClient) GetUTXOs() []*abejson.UTXO {
	utxoList, err := c.GetUTXOList(&abejson.ListUnspentCoinbaseAbeCmd{}, 60)
	if err != nil {
		return nil
	}
	return utxoList
}

func (c *RPCClient) checkHandler() {
	defer c.wg.Done()
	log.Infof("checkHandler working")
out:
	for {
		select {
		case task := <-c.checkChan:
			log.Infof("check channel receive a height, checking it...")
			// Check the transaction status by transaction hash or request hash
			if task.TxHash != nil && !task.TxHash.IsEqual(&utils.ZeroHash) {
				status, err := c.QueryTransactionStatus(task.TxHash, 60)
				if err != nil {
					log.Infof("Fail to query transaction status for %s with QueryTransactionStatus: %v", task.TxHash, err)
					task.ResponseChan <- &abejson.TxStatus{
						Status: -2, // re-query request
					}
					continue
				}
				// the request hash is valid but the transaction hash is empty
				// It means that should manually handle it
				if status == -1 {
					task.ResponseChan <- &abejson.TxStatus{
						Status: do.TransactionManual, // manual
					}
					continue
				}
				log.Infof("Successful query transaction %s status with %d", task.TxHash, status)
				task.ResponseChan <- &abejson.TxStatus{
					TxHash: task.TxHash,
					Status: status,
				}
			} else if task.RequestTxHash != nil && !task.RequestTxHash.IsEqual(&utils.ZeroHash) {
				// resend transaction request
				log.Infof("Query transaction request for request hash %s", task.RequestTxHash)
				txHashStr, status, err := c.QueryTransactionHashFromRequestHash(task.RequestTxHash, 60)
				// if any error, query next time
				if err != nil {
					log.Infof("Fail to query transaction status for %s with QueryTransactionStatus: %v", task.TxHash, err)
					task.ResponseChan <- &abejson.TxStatus{
						Status: -2, // re-query request
					}
					continue
				}
				// the request hash is valid but the transaction hash is empty
				// It means that should manually handle it
				if status == -1 {
					task.ResponseChan <- &abejson.TxStatus{
						Status: do.TransactionManual, // manual
					}
					continue
				}
				txHash, _ := utils.NewHashFromStr(txHashStr)
				task.ResponseChan <- &abejson.TxStatus{
					TxHash: txHash,
					Status: status, // 0 1 2
				}
			} else {
				log.Errorf("unsupported check request that request_hash and transaction_hash are nil or zero hash")
				task.ResponseChan <- &abejson.TxStatus{
					Status: do.TransactionManual, // manual
				}
			}
		case <-c.quit:
			break out
		}
	}
	log.Trace("Check handler done")
}
func (c *RPCClient) allocateHandler() {
	defer c.wg.Done()
	log.Infof("allocateHandler working")
	message, err := c.GetBalancesabe(20)
	if err != nil {
		log.Errorf("Unable to open connection to wallet RPC server: %v", err)
	}
	log.Infof("balance information:%v", message)
	getBalanceInterval := time.Second * 60
	getBalanceTicker := time.NewTicker(getBalanceInterval)
	defer getBalanceTicker.Stop()
out:
	for {
		select {
		case <-getBalanceTicker.C:
			// For Test Connection Of Wallet
			message, err = c.GetBalancesabe(20)
			if err != nil {
				log.Errorf("Unable to open connection to wallet RPC server: %v", err)
			}
			log.Infof("balance information:%v", message)
		case task := <-c.taskChan:
			log.Infof("Task channel receive task, working it...")
			flag := true
			// Check the address and compute total amount
			totalAmount := float64(0)
			for i := 0; i < len(task.Pairs); i++ {
				totalAmount += task.Pairs[i].Amount
				err = utils.CheckAddressValidity(task.Pairs[i].Address, chaincfg.ActiveNetParams)
				if err != nil {
					log.Warnf("Address %s valid error: %v", task.Pairs[i].Address, err)
					flag = false
					break
				}
			}
			if !flag {
				log.Debugf("Fail to allocate share with wallet: some address to allocate valid error: %v", err)
				task.ResponseChan <- &abejson.TxStatus{
					Status: do.TransactionManual,
				}
				break
			}
			log.Infof("Start unlock wallet...")
			_, err = c.WalletUnlock(&abejson.WalletUnlockCmd{
				Passphrase: c.walletPrivPass,
				Timeout:    60,
			}, 60)
			if err != nil {
				log.Infof("Fail to unlock wallet: %v", err)
				task.ResponseChan <- &abejson.TxStatus{
					Status: do.TransactionPending,
				}
				break
			}
			log.Infof("Successful unlock wallet")

			log.Infof("Allocate share with abewallet... (Address Num: %v, Amount: %f, Timestamp: %s)",
				len(task.Pairs), totalAmount, time.Now().Format("2006-01-02 15:04:05"))
			resp := &abejson.TxStatus{}
			txHash, err := c.SendToAddress(task.BatchOrderInfoCmd, 60)
			if err != nil {
				log.Infof("Fail to send cmd to wallet with SendToAddress: %v", err)
				if errors.Is(err, abejson.ErrRPCTimeout) {
					resp.Status = do.TransactionPending
				} else {
					resp.Status = do.TransactionManual
				}
			} else {
				log.Infof("Successful send to wallet with SendToAddress: %s", txHash)
				resp.Status = do.TransactionWaiting
				resp.TxHash = txHash
			}
			task.ResponseChan <- resp

			_, err = c.WalletLock(&abejson.WalletLockCmd{}, 60)
			if err != nil {
				log.Infof("Fail to lock wallet: %v", err.Error())
			} else {
				log.Infof("Successful lock wallet")
			}
		case <-c.quit:
			break out
		}
	}
	log.Trace("Allocate handler done")
}

func (c *RPCClient) onTxAccepted(hash *utils.Hash, height int32) {
	if hash == nil {
		log.Errorf("hash in onTxAccepted is nil")
		return
	}
	c.txAcceptedNotification <- &TransactionInfo{
		TxHash: hash.String(),
		Height: height,
	}
}
func (c *RPCClient) onTxRollback(hash *utils.Hash, height int32) {
	if hash == nil {
		log.Errorf("hash in onTxRollback is nil")
		return
	}
	c.txRollbackNotification <- &TransactionInfo{
		TxHash: hash.String(),
		Height: height,
	}
}
func (c *RPCClient) onTxInvalid(hash *utils.Hash, height int32) {
	if hash == nil {
		log.Errorf("hash in onTxRollback is nil")
		return
	}
	c.txInvalidNotification <- &TransactionInfo{
		TxHash: hash.String(),
		Height: height,
	}
}

// templateHandler gets the block template from abec periodically
// and notify the miners if template changes
func (c *RPCClient) notificationHandler() {
	log.Info("Waiting transaction acception notification...")
out:
	for {
		select {
		case txInfo := <-c.txAcceptedNotification:
			log.Infof("transaction %v accepted...", &TransactionInfo{
				TxHash: txInfo.TxHash,
				Height: txInfo.Height,
			})
			c.sendNotification(NTTxAccepted, txInfo)

		case txInfo := <-c.txRollbackNotification:
			log.Infof("transaction %v rollback...", &TransactionInfo{
				TxHash: txInfo.TxHash,
				Height: txInfo.Height,
			})
			c.sendNotification(NTTxRollback, txInfo)
		case txInfo := <-c.txInvalidNotification:
			log.Infof("transaction %v invalid...", &TransactionInfo{
				TxHash: txInfo.TxHash,
				Height: txInfo.Height,
			})
			c.sendNotification(NTTxInvalid, txInfo)
		case <-c.quit:
			break out
		}
	}
	c.wg.Done()
	log.Trace("Template handler done")
}
func (c *RPCClient) Subscribe(callback NotificationCallback) {
	c.notificationsLock.Lock()
	c.notifications = append(c.notifications, callback)
	c.notificationsLock.Unlock()
}

func (c *RPCClient) sendNotification(typ NotificationType, data interface{}) {
	// Generate and send the notification.
	n := Notification{Type: typ, Data: data}
	c.notificationsLock.RLock()
	for _, callback := range c.notifications {
		callback(&n)
	}
	c.notificationsLock.RUnlock()
}

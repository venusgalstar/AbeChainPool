package poolserver

import (
	"bufio"
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/abesuite/abe-miningpool-server/chainclient"
	"github.com/abesuite/abe-miningpool-server/minermgr"
	"github.com/abesuite/abe-miningpool-server/model"
	"github.com/abesuite/abe-miningpool-server/pooljson"
	"github.com/abesuite/abe-miningpool-server/utils"

	"gorm.io/gorm"
)

// Notification control requests
type notificationRegisterTCPClient TCPSocketClient
type notificationUnregisterTCPClient TCPSocketClient
type notificationNotifyNewJobTCPClient TCPSocketClient

// TCPSocketNotificationManager is a connection and notification manager used for
// websockets.  It allows websocket clients to register for notifications they
// are interested in.  When an event happens elsewhere in the code such as
// transactions being added to the memory pool or block connects/disconnects,
// the notification manager is provided with the relevant details needed to
// figure out which websocket clients need to be notified based on what they
// have registered for and notifies them accordingly.  It is also used to keep
// track of all connected websocket clients.
type TCPSocketNotificationManager struct {
	// server is the RPC server the notification manager is associated with.
	server *TCPSocketServer

	// queueNotification queues a notification for handling.
	queueNotification chan interface{}

	// notificationMsgs feeds notificationHandler with notifications
	// and client (un)registeration requests from a queue as well as
	// registeration and unregisteration requests from clients.
	notificationMsgs chan interface{}

	// Access channel for current number of connected clients.
	numClients chan int

	stallLock     sync.Mutex
	stallHandlers map[chan struct{}]chan struct{}

	// Shutdown handling
	wg   sync.WaitGroup
	quit chan struct{}
}

// newTCPSocketNotificationManager returns a new notification manager ready for use.
// See TCPSocketNotificationManager for more details.
func newTCPSocketNotificationManager(server *TCPSocketServer) *TCPSocketNotificationManager {
	return &TCPSocketNotificationManager{
		server:            server,
		queueNotification: make(chan interface{}),
		notificationMsgs:  make(chan interface{}),
		numClients:        make(chan int),
		stallHandlers:     make(map[chan struct{}]chan struct{}),
		quit:              make(chan struct{}),
	}
}

// Start starts the goroutines required for the manager to queue and process
// websocket client notifications.
func (m *TCPSocketNotificationManager) Start() {
	m.wg.Add(2)
	go m.queueHandler()
	go m.notificationHandler()
}

// NumClients returns the number of clients actively being served.
func (m *TCPSocketNotificationManager) NumClients() (n int) {
	select {
	case n = <-m.numClients:
	case <-m.quit: // Use default n (0) if server has shut down.
	}
	return
}

// newTCPSocketClient returns a new websocket client given the notification
// manager, websocket connection, remote address, and whether or not the client
// has already been authenticated (via HTTP Basic access authentication).  The
// returned client is ready to start.  Once started, the client will process
// incoming and outgoing messages in separate goroutines complete with queuing
// and asynchrous handling for long-running operations.
func newTCPSocketClient(server *TCPSocketServer, conn net.Conn,
	remoteAddr string) (*TCPSocketClient, error) {

	sessionID, err := utils.RandomUint64()
	if err != nil {
		return nil, err
	}

	client := &TCPSocketClient{
		conn:              conn,
		addr:              remoteAddr,
		sessionID:         sessionID,
		server:            server,
		serviceRequestSem: makeSemaphore(server.cfg.RPCMaxConcurrentReqs),
		ntfnChan:          make(chan []byte, 1), // nonblocking sync
		sendChan:          make(chan wsResponse, websocketSendBufferSize),
		quit:              make(chan struct{}),
	}
	return client, nil
}

// TCPSocketClient provides an abstraction for handling a websocket client.  The
// overall data flow is split into 3 main goroutines, a possible 4th goroutine
// for long-running operations (only started if request is made), and a
// websocket manager which is used to allow things such as broadcasting
// requested notifications to all connected websocket clients.   Inbound
// messages are read via the inHandler goroutine and generally dispatched to
// their own handler.  However, certain potentially long-running operations such
// as rescans, are sent to the asyncHander goroutine and are limited to one at a
// time.  There are two outbound message types - one for responding to client
// requests and another for async notifications.  Responses to client requests
// use SendMessage which employs a buffered channel thereby limiting the number
// of outstanding requests that can be made.  Notifications are sent via
// QueueNotification which implements a queue via notificationQueueHandler to
// ensure sending notifications from other subsystems can't block.  Ultimately,
// all messages are sent via the outHandler.
type TCPSocketClient struct {
	sync.Mutex

	// server is the RPC server that is servicing the client.
	server *TCPSocketServer

	// conn is the underlying websocket connection.
	conn net.Conn

	// disconnected indicated whether or not the websocket client is
	// disconnected.
	disconnected bool

	// addr is the remote address of the client.
	addr string

	// sessionID is a random ID generated for each client when connected.
	// These IDs may be queried by a client using the session RPC.  A change
	// to the session ID indicates that the client reconnected.
	sessionID uint64

	// Networking infrastructure.
	serviceRequestSem semaphore
	ntfnChan          chan []byte
	sendChan          chan wsResponse
	quit              chan struct{}
	wg                sync.WaitGroup
}

func (c *TCPSocketClient) Type() string {
	return "tcpsocket"
}

func (c *TCPSocketClient) RemoteAddr() string {
	return c.addr
}

func (c *TCPSocketClient) SetAbecBackendNode(desc string) {
	c.server.abecBackendNode = desc
}

func (c *TCPSocketClient) GetAbecBackendNode() string {
	return c.server.abecBackendNode
}

func (c *TCPSocketClient) GetMaxErrors() int {
	return c.server.cfg.MaxErrors
}

func (c *TCPSocketClient) GetNtfnManager() AbstractNtfnManager {
	return c.server.ntfnMgr
}

func (c *TCPSocketClient) GetChainClient() *chainclient.RPCClient {
	return c.server.chainClient
}

func (c *TCPSocketClient) QuitChan() chan struct{} {
	return c.quit
}

func (c *TCPSocketClient) GetMinerManager() *minermgr.MinerManager {
	return c.server.minerManager
}

func (c *TCPSocketClient) GetDB() *gorm.DB {
	return c.server.minerManager.Db
}

func (c *TCPSocketClient) GetRewardInterval() int {
	return c.server.rewardCfg.RewardInterval
}

func (c *TCPSocketClient) GetRewardIntervalPre() int {
	return c.server.rewardCfg.RewardIntervalPre
}

func (c *TCPSocketClient) GetRewardIntervalChangeHeight() int64 {
	return c.server.rewardCfg.ConfigChangeHeight
}

func (c *TCPSocketClient) DetailedShareInfo() bool {
	return c.server.cfg.DetailedShareInfo
}

func (c *TCPSocketClient) GetAgentBlacklist() []string {
	return c.server.commonCfg.AgentBlacklist
}

func (c *TCPSocketClient) GetAgentWhitelist() []string {
	return c.server.commonCfg.AgentWhitelist
}

func (c *TCPSocketClient) GetBlacklist() []*net.IPNet {
	return c.server.commonCfg.Blacklist
}

func (c *TCPSocketClient) GetWhitelist() []*net.IPNet {
	return c.server.commonCfg.Whitelist
}

func (c *TCPSocketClient) GetAdminBlacklist() []*net.IPNet {
	return nil
}

func (c *TCPSocketClient) GetAdminWhitelist() []*net.IPNet {
	return nil
}

// AddClient adds the passed websocket client to the notification manager.
func (m *TCPSocketNotificationManager) AddClient(wsc AbstractSocketClient) {
	tcpClient, ok := wsc.(*TCPSocketClient)
	if !ok {
		log.Errorf("Error: fail to add client: wsc is not the type *TCPSocketClient")
		return
	}
	m.queueNotification <- (*notificationRegisterTCPClient)(tcpClient)
}

// RemoveClient removes the passed websocket client and all notifications
// registered for it.
func (m *TCPSocketNotificationManager) RemoveClient(wsc AbstractSocketClient) {
	tcpClient, ok := wsc.(*TCPSocketClient)
	if !ok {
		log.Errorf("Error: fail to remove client: wsc is not the type *TCPSocketClient")
		return
	}
	select {
	case m.queueNotification <- (*notificationUnregisterTCPClient)(tcpClient):
	case <-m.quit:
	}
}

// AddNotifyNewJobClient ask the notification handler to add client
// into notify clients.
func (m *TCPSocketNotificationManager) AddNotifyNewJobClient(wsc AbstractSocketClient) {
	tcpClient, ok := wsc.(*TCPSocketClient)
	if !ok {
		log.Errorf("Error: fail to add notify new job client: wsc is not the type *TCPSocketClient")
		return
	}
	m.queueNotification <- (*notificationNotifyNewJobTCPClient)(tcpClient)
}

func (m *TCPSocketNotificationManager) NotifyNewJob(newJob *model.JobTemplate) {
	m.queueNotification <- (*notificationNewJob)(newJob)
}

func (m *TCPSocketNotificationManager) NotifyNewEpoch(epoch int64) {
	m.queueNotification <- (notificationEpochChange)(epoch)
}

func (m *TCPSocketNotificationManager) notifySet(clients map[chan struct{}]AbstractSocketClient, epoch int, target string, algo string,
	extraNonce1 string, extraNonceBitsNum int) {
	var epochSent string
	if epoch < 0 {
		epochSent = ""
	} else {
		epochSent = strconv.FormatInt(int64(epoch), 16)
	}
	var extraNonceBitsNumSent string
	if extraNonceBitsNum < 0 {
		extraNonceBitsNumSent = ""
	} else {
		extraNonceBitsNumSent = strconv.FormatInt(int64(extraNonceBitsNum), 16)
	}
	setNtfn := pooljson.NewSetNtfn(epochSent, target, algo, extraNonce1, extraNonceBitsNumSent)
	marshalledJSON, err := pooljson.MarshalCmdJson(nil, setNtfn)
	if err != nil {
		log.Errorf("Failed to marshal set notification: "+
			"%v", err)
		return
	}
	//fmt.Println(string(marshalledJSON))
	for _, wsc := range clients {
		tcpClient, ok := wsc.(*TCPSocketClient)
		if ok {
			tcpClient.QueueNotification(marshalledJSON)
		} else {
			log.Errorf("Error: fail to notify set: wsc is the not the type *TCPSocketClient")
		}
	}
}

func (m *TCPSocketNotificationManager) notifyNewJob(clients map[chan struct{}]AbstractSocketClient, job *model.JobTemplate) {
	height := strconv.FormatInt(job.Height, 16)
	cleanJob := ""
	if job.CleanJob {
		cleanJob = "1"
	} else {
		cleanJob = "0"
	}
	notifyNtfn := pooljson.NewNotifyNtfn(job.JobId, height, job.ContentHash, cleanJob)
	marshalledJSON, err := pooljson.MarshalCmdJson(nil, notifyNtfn)
	if err != nil {
		log.Errorf("Failed to marshal notify notification notifyNewJob: %v", err)
		return
	}
	//fmt.Println(string(marshalledJSON))
	for c, wsc := range clients {
		tcpClient, ok := wsc.(*TCPSocketClient)
		if ok {
			m.server.minerManager.SwitchJob(c, job)
			tcpClient.QueueNotification(marshalledJSON)
		} else {
			log.Errorf("Error: fail to notify new job: wsc is the not the type *TCPSocketClient")
		}
	}
}

func (m *TCPSocketNotificationManager) notifyNewJobE9(clients map[chan struct{}]AbstractSocketClient, job *model.JobTemplate) {

	notifyNtfn := pooljson.NewNotifyNtfnE9(job.JobId, job.PreviousHash, job.Coinbase, job.CoinbaseWithWitness, job.TxHashes, strconv.FormatUint(uint64(job.Version), 16), job.TargetDifficulty, job.CurrTime, job.CleanJob)
	marshalledJSON, err := json.Marshal(notifyNtfn)
	if err != nil {
		log.Errorf("Failed to marshal notify notification notifyNewJobE9: %v", err)
		return
	}

	fmt.Println("notifyNewJobE9")
	fmt.Println(string(marshalledJSON))

	for c, wsc := range clients {
		tcpClient, ok := wsc.(*TCPSocketClient)
		if ok {
			m.server.minerManager.SwitchJob(c, job)
			tcpClient.QueueNotification(marshalledJSON)
		} else {
			log.Errorf("Error: fail to notify new job: wsc is the not the type *TCPSocketClient")
		}
	}
}

func (m *TCPSocketNotificationManager) notifyNewEpoch(clients map[chan struct{}]AbstractSocketClient, epoch int64) {
	epochStr := strconv.FormatInt(epoch, 16)

	setNtfn := pooljson.NewSetNtfn(epochStr, "", "", "", "")
	marshalledJSON, err := pooljson.MarshalCmdJson(nil, setNtfn)
	if err != nil {
		log.Errorf("Failed to marshal set notification: %v", err)
		return
	}
	//fmt.Println(string(marshalledJSON))
	for _, wsc := range clients {
		tcpClient, ok := wsc.(*TCPSocketClient)
		if ok {
			tcpClient.QueueNotification(marshalledJSON)
		} else {
			log.Errorf("Error: fail to notify new epoch: wsc is the not the type *TCPSocketClient")
		}
	}
}

// inHandler handles all incoming messages for the websocket connection.  It
// must be run as a goroutine.
func (c *TCPSocketClient) inHandler() {
out:
	for {
		// Break out of the loop once the quit channel has been closed.
		// Use a non-blocking select here so we fall through otherwise.
		select {
		case <-c.quit:
			break out
		default:
		}

		msg, err := bufio.NewReader(c.conn).ReadBytes('\n')
		if err != nil {
			// Log the error if it's not due to disconnecting.
			if err != io.EOF {
				log.Errorf("TCP socket receive error from "+
					"%s: %v", c.addr, err)
			}
			break out
		}

		var request pooljson.Request
		err = json.Unmarshal(msg, &request)

		fmt.Printf("inHandler %s, %d\n", msg, len(msg))

		if err != nil {

			if len(msg) == 1 {
				continue
			}

			jsonErr := &pooljson.RPCError{
				Code:    pooljson.ErrRPCParse.Code,
				Message: "Failed to parse request: " + err.Error(),
			}
			reply, err := createMarshalledReply(nil, nil, jsonErr)
			if err != nil {
				log.Errorf("Failed to marshal parse failure "+
					"reply: %v", err)
				continue
			}
			c.SendMessage(reply, nil)
			continue
		}

		if request.ID == nil {
			continue
		}

		cmd := parseCmd(&request)
		if cmd.err != nil {
			log.Errorf("Failed to marshal parse failure "+
				"reply: %v", cmd.err)
			reply, err := createMarshalledReply(cmd.id, nil, cmd.err)
			if err != nil {
				log.Errorf("Failed to marshal parse failure "+
					"reply: %v", err)
				continue
			}
			c.SendMessage(reply, nil)
			continue
		}
		log.Debugf("Received command <%s> from %s", cmd.method, c.addr)

		// Check if the client is using limited RPC credentials and
		// error when not authorized to call this RPC.
		if _, ok := rpcLimited[request.Method]; !ok {
			jsonErr := &pooljson.RPCError{
				Code:    pooljson.ErrRPCInvalidParams.Code,
				Message: "limited user not authorized for this method",
			}
			// Marshal and send response.
			reply, err := createMarshalledReply(request.ID, nil, jsonErr)
			if err != nil {
				log.Errorf("Failed to marshal parse failure "+
					"reply: %v", err)
				continue
			}
			c.SendMessage(reply, nil)
			continue
		}

		// Asynchronously handle the request.  A semaphore is used to
		// limit the number of concurrent requests currently being
		// serviced.  If the semaphore can not be acquired, simply wait
		// until a request finished before reading the next RPC request
		// from the websocket client.
		//
		// This could be a little fancier by timing out and erroring
		// when it takes too long to service the request, but if that is
		// done, the read of the next request should not be blocked by
		// this semaphore, otherwise the next request will be read and
		// will probably sit here for another few seconds before timing
		// out as well.  This will cause the total timeout duration for
		// later requests to be much longer than the check here would
		// imply.
		//
		// If a timeout is added, the semaphore acquiring should be
		// moved inside of the new goroutine with a select statement
		// that also reads a time.After channel.  This will unblock the
		// read of the next request from the websocket client and allow
		// many requests to be waited on concurrently.
		c.serviceRequestSem.acquire()
		go func() {
			c.serviceRequest(cmd)
			c.serviceRequestSem.release()
		}()
	}

	// Ensure the connection is closed.
	c.Disconnect()
	c.wg.Done()
	log.Tracef("TCP socket client input handler done for %s", c.addr)
}

// Disconnected returns whether or not the websocket client is disconnected.
func (c *TCPSocketClient) Disconnected() bool {
	c.Lock()
	isDisconnected := c.disconnected
	c.Unlock()

	return isDisconnected
}

// SendMessage sends the passed json to the websocket client.  It is backed
// by a buffered channel, so it will not block until the send channel is full.
// Note however that QueueNotification must be used for sending async
// notifications instead of the this function.  This approach allows a limit to
// the number of outstanding requests a client can make without preventing or
// blocking on async notifications.
func (c *TCPSocketClient) SendMessage(marshalledJSON []byte, doneChan chan bool) {
	// Don't send the message if disconnected.
	if c.Disconnected() {
		if doneChan != nil {
			doneChan <- false
		}
		return
	}

	c.sendChan <- wsResponse{msg: marshalledJSON, doneChan: doneChan}
}

// serviceRequest services a parsed RPC request by looking up and executing the
// appropriate RPC handler.  The response is marshalled and sent to the
// websocket client.
func (c *TCPSocketClient) serviceRequest(r *parsedRPCCmd) {
	// Recovery
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Panic from %v handler: %v\n", r.method, err)
			var buf [4096]byte
			n := runtime.Stack(buf[:], false)
			log.Errorf("Stack Trace ==>\n %s\n", string(buf[:n]))
			log.Infof("Recovering...")
			// Dump panic file
			_ = utils.DumpPanicInfo(fmt.Sprintf("%v", err) + "\n" + string(buf[:]))

			reply, err := createMarshalledReply(r.id, nil, pooljson.ErrRPCInternal)
			if err != nil {
				log.Errorf("Failed to marshal reply for <%s> "+
					"command: %v", r.method, err)
				return
			}
			c.SendMessage(reply, nil)
		}
	}()

	var (
		result interface{}
		err    error
	)

	// Lookup the for the command.
	handler, ok := abstractHandlers[r.method]
	if ok {
		result, err = handler(c, r.cmd)
	} else {
		result, err = nil, pooljson.ErrRPCMethodNotFound
	}
	reply, err := createMarshalledReply(r.id, result, err)
	if err != nil {
		log.Errorf("Failed to marshal reply for <%s> "+
			"command: %v", r.method, err)
		return
	}
	fmt.Println(string(reply))
	c.SendMessage(reply, nil)
}

// QueueNotification queues the passed notification to be sent to the websocket
// client.  This function, as the name implies, is only intended for
// notifications since it has additional logic to prevent other subsystems, such
// as the memory pool and block manager, from blocking even when the send
// channel is full.
//
// If the client is in the process of shutting down, this function returns
// ErrClientQuit.  This is intended to be checked by long-running notification
// handlers to stop processing if there is no more work needed to be done.
func (c *TCPSocketClient) QueueNotification(marshalledJSON []byte) error {
	// Don't queue the message if disconnected.
	if c.Disconnected() {
		return ErrClientQuit
	}

	c.ntfnChan <- marshalledJSON
	return nil
}

// Disconnect disconnects the websocket client.
func (c *TCPSocketClient) Disconnect() {
	c.Lock()
	defer c.Unlock()

	// Nothing to do if already disconnected.
	if c.disconnected {
		return
	}

	log.Tracef("Disconnecting websocket client %s", c.addr)
	close(c.quit)
	c.conn.Close()
	c.disconnected = true
}

// DisconnectGracefully disconnects the websocket client after several seconds, should be called with goroutine.
// It also sends a mining.bye message.
func (c *TCPSocketClient) DisconnectGracefully() {
	goodByeCmd := pooljson.NewGoodByeNtfn()
	marshalledJSON, err := pooljson.MarshalCmdJson(nil, goodByeCmd)
	if err != nil {
		log.Errorf("Failed to marshal mining.bye notification: %v", err)
		return
	}

	c.QueueNotification(marshalledJSON)

	time.Sleep(time.Second * 5)
	c.Disconnect()
}

// WaitForShutdown blocks until all notification manager goroutines have
// finished.
func (m *TCPSocketNotificationManager) WaitForShutdown() {
	m.wg.Wait()
}

// Shutdown shuts down the manager, stopping the notification queue and
// notification handler goroutines.
func (m *TCPSocketNotificationManager) Shutdown() {
	close(m.quit)
}

func (m *TCPSocketNotificationManager) CloseAndDeleteStallQuit(quit chan struct{}) {
	m.stallLock.Lock()
	defer m.stallLock.Unlock()
	ch, ok := m.stallHandlers[quit]
	if ok {
		delete(m.stallHandlers, quit)
		close(ch)
	}
}

// notificationQueueHandler handles the queuing of outgoing notifications for
// the websocket client.
func (c *TCPSocketClient) notificationQueueHandler() {
	ntfnSentChan := make(chan bool, 1) // nonblocking sync

	// pendingNtfns is used as a queue for notifications that are ready to
	// be sent once there are no outstanding notifications currently being
	// sent.  The waiting flag is used over simply checking for items in the
	// pending list to ensure cleanup knows what has and hasn't been sent
	// to the outHandler.  Currently no special cleanup is needed, however
	// if something like a done channel is added to notifications in the
	// future, not knowing what has and hasn't been sent to the outHandler
	// (and thus who should respond to the done channel) would be
	// problematic without using this approach.
	pendingNtfns := list.New()
	waiting := false
out:
	for {
		select {
		// This channel is notified when a message is being queued to
		// be sent across the network socket.  It will either send the
		// message immediately if a send is not already in progress, or
		// queue the message to be sent once the other pending messages
		// are sent.
		case msg := <-c.ntfnChan:
			if !waiting {
				c.SendMessage(msg, ntfnSentChan)
			} else {
				pendingNtfns.PushBack(msg)
			}
			waiting = true

		// This channel is notified when a notification has been sent
		// across the network socket.
		case <-ntfnSentChan:
			// No longer waiting if there are no more messages in
			// the pending messages queue.
			next := pendingNtfns.Front()
			if next == nil {
				waiting = false
				continue
			}

			// Notify the outHandler about the next item to
			// asynchronously send.
			msg := pendingNtfns.Remove(next).([]byte)
			c.SendMessage(msg, ntfnSentChan)

		case <-c.quit:
			break out
		}
	}

	// Drain any wait channels before exiting so nothing is left waiting
	// around to send.
cleanup:
	for {
		select {
		case <-c.ntfnChan:
		case <-ntfnSentChan:
		default:
			break cleanup
		}
	}
	c.wg.Done()
	log.Tracef("TCP socket client notification queue handler done "+
		"for %s", c.addr)
}

// outHandler handles all outgoing messages for the websocket connection.  It
// must be run as a goroutine.  It uses a buffered channel to serialize output
// messages while allowing the sender to continue running asynchronously.  It
// must be run as a goroutine.
func (c *TCPSocketClient) outHandler() {
out:
	for {
		// Send any messages ready for send until the quit channel is
		// closed.
		select {
		case r := <-c.sendChan:
			msgSent := append(r.msg, '\n')
			_, err := c.conn.Write(msgSent)
			if err != nil {
				c.Disconnect()
				break out
			}
			if r.doneChan != nil {
				r.doneChan <- true
			}

		case <-c.quit:
			break out
		}
	}

	// Drain any wait channels before exiting so nothing is left waiting
	// around to send.
cleanup:
	for {
		select {
		case r := <-c.sendChan:
			if r.doneChan != nil {
				r.doneChan <- false
			}
		default:
			break cleanup
		}
	}
	c.wg.Done()
	log.Tracef("TCP socket client output handler done for %s", c.addr)
}

// queueHandler maintains a queue of notifications and notification handler
// control messages.
func (m *TCPSocketNotificationManager) queueHandler() {
	queueHandler(m.queueNotification, m.notificationMsgs, m.quit)
	m.wg.Done()
}

// notificationHandler reads notifications and control messages from the queue
// handler and processes one at a time.
func (m *TCPSocketNotificationManager) notificationHandler() {
	// clients is a map of all currently connected websocket clients.
	// It should be noticed that this does not mean the client has passed the
	// username and password check.
	// clients include miners and admins
	clients := make(map[chan struct{}]*TCPSocketClient)
	miners := make(map[chan struct{}]*TCPSocketClient)

	newJobNotificationClients := make(map[chan struct{}]AbstractSocketClient)

out:
	for {
		select {
		case n, ok := <-m.notificationMsgs:
			if !ok {
				// queueHandler quit.
				break out
			}
			switch nT := n.(type) {
			case *notificationRegisterTCPClient:
				wsc := (*TCPSocketClient)(nT)
				log.Infof("New miner registered: %v", nT.addr)
				miners[wsc.quit] = wsc

				// Add stall handler for miner
				m.stallLock.Lock()
				stallHandlerQuit := make(chan struct{})
				m.stallHandlers[wsc.quit] = stallHandlerQuit
				m.stallLock.Unlock()
				go stallHandler(wsc, stallHandlerQuit)
				clients[wsc.quit] = wsc

			case *notificationUnregisterTCPClient:
				wsc := (*TCPSocketClient)(nT)
				log.Infof("A miner disconnected: %v", nT.addr)
				delete(miners, wsc.quit)
				delete(clients, wsc.quit)
				delete(newJobNotificationClients, wsc.quit)
				wsc.server.minerManager.DeleteActiveMiner(wsc.quit)
				stopStallHandler(wsc)

			case *notificationNotifyNewJobTCPClient:
				wsc := (*TCPSocketClient)(nT)
				newJobNotificationClients[wsc.quit] = wsc

			case *notificationNewJob:
				newJob := (*model.JobTemplate)(nT)
				// m.notifyNewJob(newJobNotificationClients, newJob)
				m.notifyNewJobE9(newJobNotificationClients, newJob)

			case notificationEpochChange:
				epoch := (int64)(nT)
				m.notifyNewEpoch(newJobNotificationClients, epoch)

			default:
				log.Warnf("Unhandled notification type %v", nT)
			}

		case m.numClients <- len(clients):

		case <-m.quit:
			// Pool RPC server shutting down.
			break out
		}
	}

	for _, c := range clients {
		c.Disconnect()
	}
	m.wg.Done()
}

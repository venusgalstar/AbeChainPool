package rpcclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/abesuite/abe-miningpool-server/abejson"
	"github.com/abesuite/abe-miningpool-server/utils"
)

var (
	// ErrWebsocketsRequired is an error to describe the condition where the
	// caller is trying to use a websocket-only feature, such as requesting
	// notifications or other websocket requests when the client is
	// configured to run in HTTP POST mode.
	ErrWebsocketsRequired = errors.New("a websocket connection is required " +
		"to use this feature")
)

// notificationState is used to track the current state of successfully
// registered notification so the state can be automatically re-established on
// reconnect.
type notificationState struct {
	notifyBlocks      bool
	notifyTransaction bool
}

// Copy returns a deep copy of the receiver.
func (s *notificationState) Copy() *notificationState {
	var stateCopy notificationState
	stateCopy.notifyBlocks = s.notifyBlocks
	stateCopy.notifyTransaction = s.notifyTransaction

	return &stateCopy
}

// newNotificationState returns a new notification state ready to be populated.
func newNotificationState() *notificationState {
	return &notificationState{}
}

// newNilFutureResult returns a new future result channel that already has the
// result waiting on the channel with the reply set to nil.  This is useful
// to ignore things such as notifications when the caller didn't specify any
// notification handlers.
func newNilFutureResult() chan *response {
	responseChan := make(chan *response, 1)
	responseChan <- &response{result: nil, err: nil}
	return responseChan
}

// NotificationHandlers defines callback function pointers to invoke with
// notifications.  Since all of the functions are nil by default, all
// notifications are effectively ignored until their handlers are set to a
// concrete callback.
//
// NOTE: Unless otherwise documented, these handlers must NOT directly call any
// blocking calls on the client instance since the input reader goroutine blocks
// until the callback has completed.  Doing so will result in a deadlock
// situation.
type NotificationHandlers struct {
	// OnClientConnected is invoked when the client connects or reconnects
	// to the RPC server.  This callback is run async with the rest of the
	// notification handlers, and is safe for blocking client requests.
	OnClientConnected func()

	OnClientDisConnected func()

	// OnBlockConnected is invoked when a block is connected to the longest
	// (best) chain.  It will only be invoked if a preceding call to
	// NotifyBlocks has been made to register for the notification and the
	// function is non-nil.
	OnBlockAbeConnected func(hash *utils.Hash, height int32, t time.Time)

	OnBlockAbeDisconnected func(hash *utils.Hash, height int32, t time.Time)

	OnTxAccepted func(hash *utils.Hash, height int32)
	OnTxRollback func(hash *utils.Hash, height int32)
	OnTxInvalid  func(hash *utils.Hash, height int32)

	// OnUnknownNotification is invoked when an unrecognized notification
	// is received.  This typically means the notification handling code
	// for this package needs to be updated for a new notification type or
	// the caller is using a custom notification this package does not know
	// about.
	OnUnknownNotification func(method string, params []json.RawMessage)
}

// handleNotification examines the passed notification type, performs
// conversions to get the raw notification types into higher level types and
// delivers the notification to the appropriate On<X> handler registered with
// the client.
func (c *Client) handleNotification(ntfn *rawNotification) {
	// Ignore the notification if the client is not interested in any
	// notifications.
	if c.ntfnHandlers == nil {
		return
	}

	switch ntfn.Method {
	case abejson.BlockAbeConnectedNtfnMethod:
		// Ignore the notification if the client is not interested in
		// it.
		if c.ntfnHandlers.OnBlockAbeConnected == nil {
			return
		}

		blockHash, blockHeight, blockTime, err := parseChainNtfnParams(ntfn.Params)
		if err != nil {
			log.Warnf("Received invalid block connected "+
				"notification: %v", err)
			return
		}

		c.ntfnHandlers.OnBlockAbeConnected(blockHash, blockHeight, blockTime)

	case abejson.BlockAbeDisconnectedNtfnMethod:
		// Ignore the notification if the client is not interested in
		// it.
		if c.ntfnHandlers.OnBlockAbeDisconnected == nil {
			return
		}

		blockHash, blockHeight, blockTime, err := parseChainNtfnParams(ntfn.Params)
		if err != nil {
			log.Warnf("Received invalid block connected "+
				"notification: %v", err)
			return
		}
		c.ntfnHandlers.OnBlockAbeDisconnected(blockHash, blockHeight, blockTime)

	case abejson.TxAcceptedNtfnMethod:
		// Ignore the notification if the client is not interested in
		// it.
		if c.ntfnHandlers.OnTxAccepted == nil {
			return
		}
		if len(ntfn.Params) != 2 {
			log.Warnf("Received invalid transaction notification: %v", wrongNumParams(2))
			return
		}
		var txHashStr string
		err := json.Unmarshal(ntfn.Params[0], &txHashStr)
		if err != nil {
			log.Warnf("Received invalid tx accepted "+
				"notification(transaction hash): %v", err)
			return
		}

		// Create hash from block hash string.
		txHash, err := utils.NewHashFromStr(txHashStr)
		if err != nil {
			log.Warnf("Received invalid tx accepted "+
				"notification(transaction hash): %v", err)
			return
		}

		var height int32
		err = json.Unmarshal(ntfn.Params[1], &height)
		if err != nil {
			log.Warnf("Received invalid tx accepted "+
				"notification(height): %v", err)
			return
		}
		c.ntfnHandlers.OnTxAccepted(txHash, height)

	case abejson.TxRollbackNtfnMethod:
		// Ignore the notification if the client is not interested in
		// it.
		if c.ntfnHandlers.OnTxRollback == nil {
			return
		}

		if len(ntfn.Params) != 2 {
			log.Warnf("Received invalid transaction notification: %v", wrongNumParams(2))
			return
		}

		var txHashStr string
		err := json.Unmarshal(ntfn.Params[0], &txHashStr)
		if err != nil {
			log.Warnf("Received invalid tx rollback "+
				"notification: %v", err)
			return
		}

		// Create hash from block hash string.
		txHash, err := utils.NewHashFromStr(txHashStr)
		if err != nil {
			log.Warnf("Received invalid tx rollback "+
				"notification: %v", err)
			return
		}

		var height int32
		err = json.Unmarshal(ntfn.Params[1], &height)
		if err != nil {
			log.Warnf("Received invalid tx rollback "+
				"notification: %v", err)
			return
		}

		c.ntfnHandlers.OnTxRollback(txHash, height)

	case abejson.TxInvalidNtfnMethod:
		// Ignore the notification if the client is not interested in
		// it.
		if c.ntfnHandlers.OnTxInvalid == nil {
			return
		}

		if len(ntfn.Params) != 2 {
			log.Warnf("Received invalid transaction notification: %v", wrongNumParams(2))
			return
		}

		var txHashStr string
		err := json.Unmarshal(ntfn.Params[0], &txHashStr)
		if err != nil {
			log.Warnf("Received invalid tx invalid "+
				"notification: %v", err)
			return
		}

		// Create hash from block hash string.
		txHash, err := utils.NewHashFromStr(txHashStr)
		if err != nil {
			log.Warnf("Received invalid tx invalid "+
				"notification: %v", err)
			return
		}

		var height int32
		err = json.Unmarshal(ntfn.Params[1], &height)
		if err != nil {
			log.Warnf("Received invalid tx invalid "+
				"notification: %v", err)
			return
		}

		c.ntfnHandlers.OnTxInvalid(txHash, height)

		// OnUnknownNotification
	default:
		if c.ntfnHandlers.OnUnknownNotification == nil {
			return
		}

		c.ntfnHandlers.OnUnknownNotification(ntfn.Method, ntfn.Params)
	}
}

// wrongNumParams is an error type describing an unparseable JSON-RPC
// notificiation due to an incorrect number of parameters for the
// expected notification type.  The value is the number of parameters
// of the invalid notification.
type wrongNumParams int

// Error satisifies the builtin error interface.
func (e wrongNumParams) Error() string {
	return fmt.Sprintf("wrong number of parameters (%d)", e)
}

// parseChainNtfnParams parses out the block hash and height from the parameters
// of blockconnected and blockdisconnected notifications.
func parseChainNtfnParams(params []json.RawMessage) (*utils.Hash,
	int32, time.Time, error) {

	if len(params) != 3 {
		return nil, 0, time.Time{}, wrongNumParams(len(params))
	}

	// Unmarshal first parameter as a string.
	var blockHashStr string
	err := json.Unmarshal(params[0], &blockHashStr)
	if err != nil {
		return nil, 0, time.Time{}, err
	}

	// Unmarshal second parameter as an integer.
	var blockHeight int32
	err = json.Unmarshal(params[1], &blockHeight)
	if err != nil {
		return nil, 0, time.Time{}, err
	}

	// Unmarshal third parameter as unix time.
	var blockTimeUnix int64
	err = json.Unmarshal(params[2], &blockTimeUnix)
	if err != nil {
		return nil, 0, time.Time{}, err
	}

	// Create hash from block hash string.
	blockHash, err := utils.NewHashFromStr(blockHashStr)
	if err != nil {
		return nil, 0, time.Time{}, err
	}

	// Create time.Time from unix time.
	blockTime := time.Unix(blockTimeUnix, 0)

	return blockHash, blockHeight, blockTime, nil
}

// FutureNotifyBlocksResult is a future promise to deliver the result of a
// NotifyBlocksAsync RPC invocation (or an applicable error).
type FutureNotifyBlocksResult chan *response

// Receive waits for the response promised by the future and returns an error
// if the registration was not successful.
func (r FutureNotifyBlocksResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

// NotifyBlocksAsync returns an instance of a type that can be used to get the
// result of the RPC at some future time by invoking the Receive function on
// the returned instance.
//
// See NotifyBlocks for the blocking version and more details.
func (c *Client) NotifyBlocksAsync() FutureNotifyBlocksResult {
	// Not supported in HTTP POST mode.
	if c.config.HTTPPostMode {
		return newFutureError(ErrWebsocketsRequired)
	}

	// Ignore the notification if the client is not interested in
	// notifications.
	if c.ntfnHandlers == nil {
		return newNilFutureResult()
	}

	cmd := abejson.NewNotifyBlocksCmd()
	return c.sendCmd(cmd)
}

// NotifyBlocks registers the client to receive notifications when blocks are
// connected and disconnected from the main chain.  The notifications are
// delivered to the notification handlers associated with the client.  Calling
// this function has no effect if there are no notification handlers and will
// result in an error if the client is configured to run in HTTP POST mode.
//
// The notifications delivered as a result of this call will be via one of
// OnBlockConnected or OnBlockDisconnected.
func (c *Client) NotifyBlocks() error {
	return c.NotifyBlocksAsync().Receive()
}

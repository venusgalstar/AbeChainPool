package rpcclient

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/abesuite/abe-miningpool-server/abejson"
	"github.com/abesuite/abe-miningpool-server/utils"
)

type FutureWalletUnlockResult chan *response

func (r FutureWalletUnlockResult) Receive(timeout ...int) (interface{}, error) {
	res, err := receiveFuture(r, timeout...)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) WalletUnlockAsync(cmd *abejson.WalletUnlockCmd) FutureWalletUnlockResult {
	return c.sendCmd(cmd)
}

func (c *Client) WalletUnlock(info *abejson.WalletUnlockCmd, timeout ...int) (interface{}, error) {
	return c.WalletUnlockAsync(info).Receive(timeout...)
}

type FutureWalletLockResult chan *response

func (r FutureWalletLockResult) Receive(timeout ...int) (interface{}, error) {
	res, err := receiveFuture(r, timeout...)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) WalletLockAsync(cmd *abejson.WalletLockCmd) FutureWalletLockResult {
	return c.sendCmd(cmd)
}

func (c *Client) WalletLock(info *abejson.WalletLockCmd, timeout ...int) (interface{}, error) {
	return c.WalletLockAsync(info).Receive(timeout...)
}

type FutureSendToAddressResult chan *response

func (r FutureSendToAddressResult) Receive(timeout ...int) (*utils.Hash, error) {
	res, err := receiveFuture(r, timeout...)
	if err != nil {
		return nil, err
	}
	log.Infof("res = %s", res)
	// Unmarshal result as a string.
	txHashStr := string(res[1 : 1+utils.MaxHashStringSize])
	return utils.NewHashFromStr(txHashStr)
}

type FutureQueryTransactionStatus chan *response

func (r FutureQueryTransactionStatus) Receive(timeout ...int) (int, error) {
	res, err := receiveFuture(r, timeout...)
	if err != nil {
		return -1, err
	}
	s := string(res)
	status, err := strconv.Atoi(s)
	if err != nil {
		return -1, err
	}
	return status, nil
}

type FutureTxStatusFromRequestHash chan *response

func (r FutureTxStatusFromRequestHash) Receive(timeout ...int) (string, int, error) {
	// 1. receive the response from abewallet
	// 1.1 (transaction hash + status) and nil
	// 1.2 nil and error
	// 2. something error
	// 2.1 timeout
	// 2.2 other error
	res, err := receiveFuture(r, timeout...)
	if err != nil {
		return "", -1, err
	}

	// parse the response, if any error then return -1 and error
	// otherwise return a string and an int
	var obj map[string]interface{}
	err = json.Unmarshal(res, &obj)
	if err != nil {
		return "", -1, err
	}
	errInvalidResp := errors.New("can not parse response")
	txHash := ""
	status := float64(-1)
	ok := false
	if _, ok = obj["txHash"]; !ok {
		return "", -1, errInvalidResp
	}
	if txHash, ok = obj["txHash"].(string); !ok {
		return "", -1, errInvalidResp
	}
	if _, ok = obj["status"]; !ok {
		return "", -1, errInvalidResp
	}
	if status, ok = obj["status"].(float64); !ok {
		return "", -1, errInvalidResp
	}
	return txHash, int(status), nil
}

type FutureGetUTXOListResult chan *response

func (r FutureGetUTXOListResult) Receive(timeout ...int) ([]*abejson.UTXO, error) {
	// parse the result
	resBytes, err := receiveFuture(r, timeout...)
	if err != nil {
		return nil, err
	}
	res := make([]*abejson.UTXO, 0, 100)
	err = json.Unmarshal(resBytes, &res)
	return res, err
}

func (c *Client) GetUTXOListAsync(cmd *abejson.ListUnspentCoinbaseAbeCmd) FutureGetUTXOListResult {
	return c.sendCmd(cmd)
}

func (c *Client) GetUTXOList(cmd *abejson.ListUnspentCoinbaseAbeCmd, timeout ...int) ([]*abejson.UTXO, error) {
	return c.GetUTXOListAsync(cmd).Receive(timeout...)
}

func (c *Client) GetSendToAddressAsync(cmd *abejson.BatchOrderInfoCmd) FutureSendToAddressResult {
	return c.sendCmd(cmd)
}
func (c *Client) QueryTransactionStatusAsync(cmd *abejson.TxStatusCmd) FutureQueryTransactionStatus {
	return c.sendCmd(cmd)
}
func (c *Client) QueryTransactionHashFromRequestHashAsync(cmd *abejson.TxStatusFromRequestHashCmd) FutureTxStatusFromRequestHash {
	return c.sendCmd(cmd)
}

func (c *Client) SendToAddress(order *abejson.BatchOrderInfoCmd, timeout ...int) (*utils.Hash, error) {
	return c.GetSendToAddressAsync(order).Receive(timeout...)
}
func (c *Client) QueryTransactionStatus(txHash *utils.Hash, timeout ...int) (int, error) {
	return c.QueryTransactionStatusAsync(&abejson.TxStatusCmd{TxHash: txHash.String()}).Receive(timeout...)
}
func (c *Client) QueryTransactionHashFromRequestHash(requestHash *utils.Hash, timeout ...int) (string, int, error) {
	return c.QueryTransactionHashFromRequestHashAsync(&abejson.TxStatusFromRequestHashCmd{RequestHash: requestHash.String()}).Receive(timeout...)
}

// FutureGetBalancesabeResult is a future promise to deliver the result of a version
// RPC invocation (or an applicable error).
type FutureGetBalancesabeResult chan *response

// Receive waits for the response promised by the future and returns the version
// result.
func (r FutureGetBalancesabeResult) Receive(timeout ...int) (string,
	error) {
	res, err := receiveFuture(r, timeout...)
	if err != nil {
		return "", err
	}

	// Unmarshal result as a version result object.
	//var vr *abejson.GetBalancesabeResult
	//err = json.Unmarshal(res, &vr)
	//if err != nil {
	//	return nil, err
	//}

	return string(res), nil
}

// GetBalancesabeAsync returns an instance of a type that can be used to get the result
// of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See GetBalancesabe for the blocking version and more details.
func (c *Client) GetBalancesabeAsync() FutureGetBalancesabeResult {
	cmd := abejson.NewGetBalancesabeCmd()
	return c.sendCmd(cmd)
}

func (c *Client) GetBalancesabe(timeout ...int) (string, error) {
	return c.GetBalancesabeAsync().Receive(timeout...)
}

type FutureNotificationTransactionsResult chan *response

// Receive waits for the response promised by the future and returns an error
// if the registration was not successful.
func (r FutureNotificationTransactionsResult) Receive() error {
	_, err := receiveFuture(r)
	return err
}

func (c *Client) NotifyTransactionsAsync() FutureNotificationTransactionsResult {
	// Not supported in HTTP POST mode.
	if c.config.HTTPPostMode {
		return newFutureError(ErrWebsocketsRequired)
	}

	// Ignore the notification if the client is not interested in
	// notifications.
	if c.ntfnHandlers == nil {
		return newNilFutureResult()
	}

	cmd := abejson.NewNotificationTransactionsCmd()
	return c.sendCmd(cmd)
}

func (c *Client) NotifyTransactions() error {
	return c.NotifyTransactionsAsync().Receive()
}

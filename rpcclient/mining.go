package rpcclient

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/abesuite/abe-miningpool-server/abejson"
	"github.com/abesuite/abe-miningpool-server/utils"
	"github.com/abesuite/abe-miningpool-server/wire"
)

// FutureGetBlockTemplateResponse is a future promise to deliver the result of a
// GetBlockTemplateAsync RPC invocation (or an applicable error).
type FutureGetBlockTemplateResponse chan *response

// Receive waits for the Response promised by the future and returns an error if
// any occurred when retrieving the block template.
func (r FutureGetBlockTemplateResponse) Receive(timeout ...int) (*abejson.GetBlockTemplateResult, error) {
	res, err := receiveFuture(r, timeout...)
	if err != nil {
		return nil, err
	}

	// Unmarshal result as a getwork result object.
	var result abejson.GetBlockTemplateResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetBlockTemplateAsync returns an instance of a type that can be used to get the
// result of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See GetBlockTemplate for the blocking version and more details.
func (c *Client) GetBlockTemplateAsync(req *abejson.TemplateRequest) FutureGetBlockTemplateResponse {
	cmd := abejson.NewGetBlockTemplateCmd(req)
	return c.sendCmd(cmd)
}

// GetBlockTemplate returns a new block template for mining.
func (c *Client) GetBlockTemplate(req *abejson.TemplateRequest, timeout ...int) (*abejson.GetBlockTemplateResult, error) {
	return c.GetBlockTemplateAsync(req).Receive(timeout...)
}

// FutureSubmitSimplifiedBlockResult is a future promise to deliver the result of a
// SubmitSimplifiedBlockAsync RPC invocation (or an applicable error).
type FutureSubmitSimplifiedBlockResult chan *response

// Receive waits for the response promised by the future and returns an error if
// any occurred when submitting the block.
func (r FutureSubmitSimplifiedBlockResult) Receive(timeout ...int) error {
	res, err := receiveFuture(r, timeout...)
	if err != nil {
		return err
	}

	if string(res) != "null" {
		var result string
		err = json.Unmarshal(res, &result)
		if err != nil {
			return err
		}

		return errors.New(result)
	}

	return nil
}

// SubmitSimplifiedBlockAsync returns an instance of a type that can be used to get the
// result of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See SubmitSimplifiedBlock for the blocking version and more details.
func (c *Client) SubmitSimplifiedBlockAsync(block *wire.MsgSimplifiedBlock, options *abejson.SubmitBlockOptions) FutureSubmitSimplifiedBlockResult {
	blockHex := ""
	var buf bytes.Buffer
	if block != nil {
		err := block.Serialize(&buf)
		if err != nil {
			return newFutureError(err)
		}

		blockHex = hex.EncodeToString(buf.Bytes())
	}

	cmd := abejson.NewSubmitSimplifiedBlockCmd(blockHex, options)
	return c.sendCmd(cmd)
}

// SubmitSimplifiedBlock attempts to submit a new block into the abec network.
func (c *Client) SubmitSimplifiedBlock(block *wire.MsgSimplifiedBlock, options *abejson.SubmitBlockOptions, timeout ...int) error {
	return c.SubmitSimplifiedBlockAsync(block, options).Receive(timeout...)
}

// FutureVersionResult is a future promise to deliver the result of a version
// RPC invocation (or an applicable error).
type FutureVersionResult chan *response

// Receive waits for the response promised by the future and returns the version
// result.
func (r FutureVersionResult) Receive(timeout ...int) (map[string]abejson.VersionResult,
	error) {
	res, err := receiveFuture(r, timeout...)
	if err != nil {
		return nil, err
	}

	// Unmarshal result as a version result object.
	var vr map[string]abejson.VersionResult
	err = json.Unmarshal(res, &vr)
	if err != nil {
		return nil, err
	}

	return vr, nil
}

// VersionAsync returns an instance of a type that can be used to get the result
// of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See Version for the blocking version and more details.
func (c *Client) VersionAsync() FutureVersionResult {
	cmd := abejson.NewVersionCmd()
	return c.sendCmd(cmd)
}

// Version returns information about the server's JSON-RPC API versions.
func (c *Client) Version(timeout ...int) (map[string]abejson.VersionResult, error) {
	return c.VersionAsync().Receive(timeout...)
}

// FutureGetBlockHashResult is a future promise to deliver the result of a
// GetBlockHashAsync RPC invocation (or an applicable error).
type FutureGetBlockHashResult chan *response

// Receive waits for the response promised by the future and returns the hash of
// the block in the best block chain at the given height.
func (r FutureGetBlockHashResult) Receive(timeout ...int) (*utils.Hash, error) {
	res, err := receiveFuture(r, timeout...)
	if err != nil {
		return nil, err
	}

	// Unmarshal the result as a string-encoded sha.
	var txHashStr string
	err = json.Unmarshal(res, &txHashStr)
	if err != nil {
		return nil, err
	}
	return utils.NewHashFromStr(txHashStr)
}

// GetBlockHashAsync returns an instance of a type that can be used to get the
// result of the RPC at some future time by invoking the Receive function on the
// returned instance.
//
// See GetBlockHash for the blocking version and more details.
func (c *Client) GetBlockHashAsync(blockHeight int64) FutureGetBlockHashResult {
	cmd := abejson.NewGetBlockHashCmd(blockHeight)
	return c.sendCmd(cmd)
}

// GetBlockHash returns the hash of the block in the best block chain at the
// given height.
func (c *Client) GetBlockHash(blockHeight int64, timeout ...int) (*utils.Hash, error) {
	return c.GetBlockHashAsync(blockHeight).Receive(timeout...)
}

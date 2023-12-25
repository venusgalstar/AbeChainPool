package pooljson

import (
	"time"

	"github.com/abesuite/abe-miningpool-server/utils"
)

const (
	SetDifficultyNtfnMethod = "mining.set_difficutly"
	GoodByeNtfnMethod       = "mining.bye"
	ReconnectNtfnMethod     = "mining.reconnect"
	SetNtfnMethod           = "mining.set"
	NotifyNtfnMethod        = "mining.notify"
	//HashRateNtfnMethod      = "mining.hashratentfn"
)

type SetDifficultyNtfn struct {
	Difficulty int64 `json:"difficulty"`
}

func NewSetDifficultyNtfn(difficulty int64) *SetDifficultyNtfn {
	return &SetDifficultyNtfn{
		Difficulty: difficulty,
	}
}

type GoodByeNtfn struct{}

func NewGoodByeNtfn() *GoodByeNtfn {
	return &GoodByeNtfn{}
}

type ReconnectNtfn struct {
	Host   string `json:"host"`
	Port   string `json:"port"`
	Resume string `json:"resume"`
}

func NewReconnectNtfn(host string, port string, resume string) *ReconnectNtfn {
	return &ReconnectNtfn{
		Host:   host,
		Port:   port,
		Resume: resume,
	}
}

type SetNtfn struct {
	Epoch             string `json:"epoch,omitempty"`
	Target            string `json:"target,omitempty"`
	Algo              string `json:"algo,omitempty"`
	ExtraNonce        string `json:"extranonce,omitempty"`
	ExtraNonceBitsNum string `json:"extra_nonce_bits_num,omitempty"`
}

func NewSetNtfn(epoch string, target string, algo string, extraNonce string, extraNonceBitsNum string) *SetNtfn {
	return &SetNtfn{
		Epoch:             epoch,
		Target:            target,
		Algo:              algo,
		ExtraNonce:        extraNonce,
		ExtraNonceBitsNum: extraNonceBitsNum,
	}
}

type NotifyNtfn struct {
	JobID       string `json:"job_id,omitempty"`
	Height      string `json:"height,omitempty"`
	ContentHash string `json:"content_hash,omitempty"`
	CleanJob    string `json:"clean_job,omitempty"`
}

func NewNotifyNtfn(jobID string, height string, contentHash string, cleanJob string) *NotifyNtfn {
	return &NotifyNtfn{
		JobID:       jobID,
		Height:      height,
		ContentHash: contentHash,
		CleanJob:    cleanJob,
	}
}

type NotifyNtfnE9 struct {
	Id      string   `json:"id"`
	Jsonrpc string   `json:"jsonrpc"`
	Result  []string `json:"result"`
	Algo    string   `json:"algo"`
}

func NewNotifyNtfnE9(jobID string, previousHash string, prefixCoinbase string, suffixCoinbase string, merkleRootHash []utils.Hash, version string, difficulty string, time time.Time, clean bool) *NotifyNtfnE9 {

	// {
	// 	"id": 0,
	// 	"jsonrpc": "2.0",
	// 	"result": [
	// 		"0xb7d931d402bcc5535e0be8e1364e95e822d979cdd525f15d466916f70aa91abb",
	// 		"0xdb2f18e72aba359e05c5a3a96d8a7e6ab1b86705c4b7e54604318afa935f45ff",
	// 		"0x000000007fffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	// 		"0x1211215"
	// 	],
	// 	"algo": "etchash"
	// }

	resultHash := make([]string, 0, 4)

	resultHash = append(resultHash, previousHash)
	resultHash = append(resultHash, "0xdb2f18e72aba359e05c5a3a96d8a7e6ab1b86705c4b7e54604318afa935f45ff")
	resultHash = append(resultHash, "0x000000007fffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	resultHash = append(resultHash, "0x1211215")

	notifyNtfnE9 := &NotifyNtfnE9{
		Id:      "2",
		Jsonrpc: "2.0",
		Result:  resultHash,
		Algo:    "etchash",
	}

	return notifyNtfnE9
}

type HashRateNtfn struct {
	Interval int    `json:"interval"`
	HashRate string `json:"hr"`
	Accepted []int  `json:"accepted"`
	Rejected int    `json:"rejected"`
}

func NewHashRateNtfn(interval int, hashRate string, accepted []int, rejected int) *HashRateNtfn {
	return &HashRateNtfn{
		Interval: interval,
		HashRate: hashRate,
		Accepted: accepted,
		Rejected: rejected,
	}
}

func init() {
	// The commands in this file are only usable by websockets and are
	// notifications.
	flags := UFWebsocketOnly | UFNotification

	MustRegisterCmd(SetDifficultyNtfnMethod, (*SetDifficultyNtfn)(nil), flags)
	MustRegisterCmd(GoodByeNtfnMethod, (*GoodByeNtfn)(nil), flags)
	MustRegisterCmd(ReconnectNtfnMethod, (*ReconnectNtfn)(nil), flags)
	MustRegisterCmd(SetNtfnMethod, (*SetNtfn)(nil), flags)
	MustRegisterCmd(NotifyNtfnMethod, (*NotifyNtfn)(nil), flags)
	//MustRegisterCmd(HashRateNtfnMethod, (*HashRateNtfn)(nil), flags)
}

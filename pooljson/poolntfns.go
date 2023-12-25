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

type NotifyNtfnV1 []interface{}

func NewNotifyNtfnV1(jobID string, previousHash string, prefixCoinbase string, suffixCoinbase string, merkleRootHash []utils.Hash, version string, difficulty string, time time.Time, clean bool) *[]interface{} {

	notifyNtfnV1 := make([]interface{}, 0, 9)
	// args [
	// 	'239334',
	// 	'aa909f8cdfe58ed1b78e5dedcdad0f085946968c000227500000000000000000',
	// 	'01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff5903a5890c1c4d696e656420627920416e74506f6f6c393630ce01250381937f1501fabe6d6dcabebeee9ca0d292c4dbf4b8a97ac14debff6069f66bbd17a7b3bd3e355bca771000000000000000',
	// 	'ffffffff04fd3903320000000017a9144b09d828dfc8baaba5d04ee77397e04b1050cc73870000000000000000266a24aa21a9ed801fc890168a4a2c8d89de23c66a8ab7d4a0485953a9d7bf6e29b9092a072bd900000000000000002f6a2d434f524501a37cf4faa0758b26dca666f3e36d42fa15cc01065997be5a09d05bb9bac27ec60419d0b373f32b2000000000000000002b6a2952534b424c4f434b3a7b15cd27f9013892223af719d5767c3e427fa78ba2ea1797f947491e005a2c4000000000',
	// 	[
	// 	  'f3ff2ae7e44f3b33f56a66b3201c1ddbb04f44433cf633833e3ead9d033efc27',
	// 	  '720726c9878544e01caa2864f901cd469ed9076ec1b82b916a9b756aa4455dfe',
	// 	  '278bcb41d78a21133ed5c19931806f8ffe445d180109130b99db55a295e614df',
	// 	  '6a18524513fd9645cc29e9e2b5227e6d67e03a6c48bb328990333d1ef2a4c887',
	// 	  'fc2af33d1f3b76e6855ddf540e59b5f745c8dea9c693eff44bf6ad96a1581aa7',
	// 	  '3209e587bac4c741a371dc1d12a6f3ea62fbfc6be8f89fcd36b33fa5178dc357',
	// 	  '9845e321cc9e23e6e49d54c529ae4df842e1e70b04082857c82b60845fa4b008',
	// 	  'a9e455e69bda1309ec1912e92451a69da003b26f3a6a70adb26fd113141cce53',
	// 	  '07e445c879990834b228b3a8f504d6ad5592f6dfce66c2a21beca0060ca6eddc',
	// 	  '826b221122890b97bb83c31fe5bf29fbdb2f7e13a1e054a8ee623fb888566f48',
	// 	  '6c83d384ee1beff1e942f1a101ca7eb918183988bb279e862ce2b1713bf09cc8'
	// 	],
	// 	'20000000',
	// 	'17042e95',
	// 	'657f63bf',
	// 	true
	//   ]
	notifyNtfnV1 = append(notifyNtfnV1, jobID)          //JobID
	notifyNtfnV1 = append(notifyNtfnV1, previousHash)   // Previous BlockHash
	notifyNtfnV1 = append(notifyNtfnV1, prefixCoinbase) // Previous BlockHash
	notifyNtfnV1 = append(notifyNtfnV1, suffixCoinbase) // Previous BlockHash
	notifyNtfnV1 = append(notifyNtfnV1, merkleRootHash) // Previous BlockHash
	notifyNtfnV1 = append(notifyNtfnV1, version)        // Previous BlockHash
	notifyNtfnV1 = append(notifyNtfnV1, difficulty)     // Previous BlockHash
	notifyNtfnV1 = append(notifyNtfnV1, time)           // Previous BlockHash

	return &notifyNtfnV1
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

package model

import (
	"math/big"
	"time"

	"github.com/abesuite/abe-miningpool-server/utils"
)

type JobTemplate struct {
	JobId                 string     `json:"job_id"`
	PreviousHash          string     `json:"previous_hash"`
	PreviousHashH         utils.Hash `json:"previous_hash_h"`
	Coinbase              string     `json:"coinbase"`
	CoinbaseWithWitness   string     `json:"coinbase_with_witness"`
	MerkleRoot            string     `json:"merkle_root"`
	MerkleRootH           utils.Hash `json:"merkle_root_h"`
	TxHashes              []utils.Hash
	Version               int32  `json:"version"`
	TargetDifficulty      string `json:"target_difficulty"`
	Bits                  uint32 `json:"bits"`
	CacheTargetDifficulty *big.Int
	CurrTime              time.Time `json:"curr_time"`
	CleanJob              bool      `json:"clean_job"`

	ContentHash  string
	ContentHashH utils.Hash

	Height int64
	Reward uint64
}

type JobTemplateMiner struct {
	CachedTargetShare    *big.Int
	CachedTargetShareStr string
	JobDetails           *JobTemplate
}

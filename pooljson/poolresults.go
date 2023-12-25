package pooljson

import (
	"github.com/abesuite/abe-miningpool-server/model"
)

// E9 response
type EthSubmitLoginResult bool

type HelloResult struct {
	Proto     string `json:"proto"`
	Encoding  string `json:"encoding"`
	Resume    string `json:"resume"`
	Timeout   string `json:"timeout"`
	MaxErrors string `json:"maxerrors"`
	Node      string `json:"node"`
}

type NoopResult struct{}

// SubscribeResult models objects included in the mining.subscribe response.
type SubscribeResult struct {
	SessionID string `json:"session_id"`
}

// AuthorizeResult models objects included in the mining.authorize response.
type AuthorizeResult struct {
	Worker     string `json:"worker"`
	Username   string `json:"username,omitempty"`
	Registered string `json:"registered"`
}

// SubmitResult models objects included in the mining.submit response.
type SubmitResult struct {
	Detail string `json:"detail,omitempty"`
}

// SubmitResult models objects included in the mining.submit response.
// type SubmitResultV1 interface {
// 	Content interface {
// 		Difficulty interface {
// 			Method string
// 			Diff   string
// 		}
// 		Notify interface {
// 			Method string
// 			Job    string
// 		}
// 	}
// 	Nonce1 string
// 	Nonce2 uint
// }

// HashRateResult models objects included in the mining.hashrate response.
type HashRateResult struct{}

// GetJobResult models objects included in the mining.get_job response.
type GetJobResult struct {
	JobId       string `json:"job_id"`
	Height      string `json:"height"`
	Epoch       string `json:"epoch"`
	ContentHash string `json:"content_hash"`
	Target      string `json:"target"`
	ExtraNonce1 string `json:"extra_nonce_1"`
}

// VersionResult models objects included in the version response.  In the actual
// result, these objects are keyed by the program or API name.
type VersionResult struct {
	VersionString string `json:"version,omitempty"`
	Major         uint32 `json:"major,omitempty"`
	Minor         uint32 `json:"minor,omitempty"`
	Patch         uint32 `json:"patch,omitempty"`
	Prerelease    string `json:"prerelease,omitempty"`
	BuildMetadata string `json:"buildmetadata,omitempty"`
}

// GetMinerNumResult models objects included in the getminernum response.
type GetMinerNumResult struct {
	Number int `json:"number"`
}

// GetUserNumResult models objects included in the getusernum response.
type GetUserNumResult struct {
	Number int64 `json:"number"`
}

type GetMetaInfoResult struct {
	Net                  string `json:"net"`
	StartTime            int64  `json:"start_time"`
	Balance              int64  `json:"balance"`
	AllocatedBalance     int64  `json:"allocated_balance"`
	UnallocatedBalance   int64  `json:"unallocated_balance"`
	CurrentHeight        int64  `json:"current_height"`
	LastRewardHeight     int64  `json:"last_reward_height"`
	RewardInterval       int    `json:"reward_interval"`
	ManageFeePercent     int    `json:"manage_fee_percent"`
	RewardMaturity       int    `json:"reward_maturity"`
	ConfigChangeHeight   int64  `json:"config_change_height"`
	NextRewardInterval   int    `json:"next_reward_interval"`
	NextManageFeePercent int    `json:"next_manage_fee_percent"`
}

type GetUserInfoResult struct {
	ID              uint64 `json:"id"`
	Username        string `json:"username"`
	Balance         int64  `json:"balance"`
	RedeemedBalance int64  `json:"redeemed_balance"`
	Password        string `json:"password"`
	Salt            string `json:"salt"`
	RegisteredAt    int64  `json:"registered_at"`
	UpdatedAt       int64  `json:"updated_at"`
	Address         string `json:"address"`
}

type GetUserInfosResult struct {
	Total int64                `json:"total"`
	Users []*GetUserInfoResult `json:"users"`
}

// CheckUsernameExistResult models objects included in the checkusernameexist response.
type CheckUsernameExistResult struct {
	IsExist bool `json:"is_exist"`
}

// RegisterUserResult models objects included in registeruser response.
type RegisterUserResult struct {
	Success bool `json:"success"`
}

// ChangePasswordResult models objects included in changepassword response.
type ChangePasswordResult struct {
	Success bool `json:"success"`
}

// ResetPasswordResult models objects included in resetpassword response.
type ResetPasswordResult struct {
	Success bool `json:"success"`
}

// CheckPasswordResult models objects included in checkpassword response.
type CheckPasswordResult struct {
	Result bool `json:"result"`
}

// WalletPassphraseResult models objects included in walletpassphrase response.
type WalletPassphraseResult struct {
	Success bool   `json:"success"`
	Detail  string `json:"detail"`
}

type Transaction struct {
	TxHash string `json:"txhash"`
	Amount uint64 `json:"amount"`
	Status int    `json:"status"`
}

// GetOrdersByUsernameResult models objects included in getorder response.
type GetOrdersByUsernameResult struct {
	ID        uint64 `json:"id"`
	UserID    uint64 `json:"user_id"`
	Amount    int64  `json:"amount"`
	Status    int    `json:"status"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type GetMinersResult struct {
	Miners []string `json:"miners"`
}

type GetMinerInfoResult struct {
	UserAgent         string   `json:"user_agent"`
	Protocol          string   `json:"protocol"`
	ConnectionType    string   `json:"connection_type"`
	Address           string   `json:"address"`
	SessionID         string   `json:"session_id"`
	Username          []string `json:"username"`
	ExtraNonce1       string   `json:"extra_nonce_1"`
	Difficulty        int64    `json:"difficulty"`
	ShareCount        int64    `json:"share_count"`
	HashRateUpload    int64    `json:"hash_rate_upload"`
	HashRateEstimated int64    `json:"hash_rate_estimated"`
	ArriveAt          int64    `json:"arrive_at"`
	LastActiveAt      int64    `json:"last_active_at"`
	ErrorCounts       int      `json:"error_counts"`
	Accepted          int      `json:"accepted"`
	Rejected          int      `json:"rejected"`
}

type GetOrdersResult struct {
	Total  int64                 `json:"total"`
	Orders []*model.OrderDetails `json:"orders"`
}

type GetTransactionsResult struct {
	Total        int64                            `json:"total"`
	Transactions []*model.OrderTransactionDetails `json:"transactions"`
}

type GetCurrentBlockTemplateResult struct {
	JobId        string   `json:"job_id"`
	PreviousHash string   `json:"previous_hash"`
	TxHashes     []string `json:"tx_hashes"`
	CurrTime     int64    `json:"curr_time"`
	ContentHash  string   `json:"content_hash"`
	Height       int64    `json:"height"`
	Reward       uint64   `json:"reward"`
}

type GetMinedBlocksResult struct {
	Total       int64         `json:"total"`
	MinedBlocks []*MinedBlock `json:"mined_blocks"`
}

type MinedBlock struct {
	ID           uint64 `json:"id"`
	Username     string `json:"username"`
	Height       int64  `json:"height"`
	BlockHash    string `json:"block_hash"`
	SealHash     string `json:"seal_hash"`
	Reward       uint64 `json:"reward"`
	Disconnected int    `json:"disconnected"`
	Connected    int    `json:"connected"`
	Rejected     int    `json:"rejected"`
	Allocated    int    `json:"allocated"`
	Info         string `json:"info"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

type UserShareInfoResult struct {
	ID          uint64 `json:"id"`
	Username    string `json:"username"`
	ShareCount  int64  `json:"share_count"`
	StartHeight int64  `json:"start_height"`
	EndHeight   int64  `json:"end_height"`
	Allocated   int64  `json:"allocated"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

type GetUserShareInfoByUsernameResult struct {
	Total int64                  `json:"total"`
	Share []*UserShareInfoResult `json:"share"`
}

type GetUserShareInfoResult struct {
	Total int64                  `json:"total"`
	Share []*UserShareInfoResult `json:"share"`
}

type GetHashRateResult struct {
	HashRate int64 `json:"hash_rate"`
}

type AllocationInfoResult struct {
	ID          uint64 `json:"id"`
	Username    string `json:"username"`
	ShareCount  int64  `json:"share_count"`
	StartHeight int64  `json:"start_height"`
	EndHeight   int64  `json:"end_height"`
	Balance     int64  `json:"balance"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

type GetAllocationInfoResult struct {
	Total int64                   `json:"total"`
	Share []*AllocationInfoResult `json:"allocation"`
}

type ModifyBlackWhitelistResult struct {
	Success bool `json:"success"`
}

type GetStatusResult struct {
	UserNum            int64 `json:"user_num"`
	MinerNum           int   `json:"miner_num"`
	MinedBlockNum      int64 `json:"mined_block_num"`
	UnfinishedOrderNum int   `json:"unfinished_order_num"`
	EnableAutoFlag     bool  `json:"enable_auto_flag"`
	AutoFlag           bool  `json:"auto_flag"`
}

type CommonResult struct {
	Success bool `json:"success"`
}

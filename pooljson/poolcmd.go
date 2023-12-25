package pooljson

// SubmitCmd defines the mining.submit JSON-RPC command.
type SubmitCmd struct {
	JobID    string `json:"job_id"`
	Nonce    string `json:"nonce"`
	WorkerID string `json:"worker_id"`
}

// NewSubmitCmd returns a new instance which can be used to issue a
// mining.submit JSON_RPC command.
func NewSubmitCmd(jobID string, nonce string, workerID string) *SubmitCmd {
	return &SubmitCmd{
		JobID:    jobID,
		Nonce:    nonce,
		WorkerID: workerID,
	}
}

type GetJobCmd struct{}

func NewGetJobCmd() *GetJobCmd {
	return &GetJobCmd{}
}

// VersionCmd defines the version JSON-RPC command.
type VersionCmd struct{}

// NewVersionCmd returns a new instance which can be used to issue a JSON-RPC
// version command.
func NewVersionCmd() *VersionCmd { return new(VersionCmd) }

// AuthenticateCmd defines the authenticate JSON-RPC command.
// used in RPC authentication
type AuthenticateCmd struct {
	Username   string
	Passphrase string
}

// NewAuthenticateCmd returns a new instance which can be used to issue an
// authenticate JSON-RPC command.
func NewAuthenticateCmd(username, passphrase string) *AuthenticateCmd {
	return &AuthenticateCmd{
		Username:   username,
		Passphrase: passphrase,
	}
}

// AuthorizeCmd defines the authorize JSON-RPC command.
// used in RPC authentication
type AuthorizeCmd struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Address     string `json:"address"`
	Registering string `json:"registering"`
}

// NewAuthorizeCmd returns a new instance which can be used to issue an
// authorize JSON-RPC command.
func NewAuthorizeCmd(username, password, address, registering string) *AuthorizeCmd {
	return &AuthorizeCmd{
		Username:    username,
		Password:    password,
		Address:     address,
		Registering: registering,
	}
}

type HelloCmd struct {
	Agent string `json:"agent"`
	Host  string `json:"host"`
	Port  string `json:"port"`
	Proto string `json:"proto"`
}

func NewHelloCmd(agent string, host string, port string, proto string) *HelloCmd {
	return &HelloCmd{
		Agent: agent,
		Host:  host,
		Port:  port,
		Proto: proto,
	}
}

// EthSumbitLoing command from E9
type EthSubmitLogin struct {
	User string
	Pass string
}

type NoopCmd struct{}

func NewNoopCmd() *NoopCmd {
	return &NoopCmd{}
}

// SubscribeCmd defines the subscribe JSON-RPC command.
type SubscribeCmd struct{}

// NewSubscribeCmd returns a new instance which can be used to issue an
// subscribe JSON-RPC command.
func NewSubscribeCmd() *SubscribeCmd {
	return &SubscribeCmd{}
}

type HashRateCmd struct {
	HashRate string `json:"hash_rate"`
	WorkerID string `json:"worker_id"`
}

func NewHashRateCmd(hashRate string, workerID string) *HashRateCmd {
	return &HashRateCmd{
		HashRate: hashRate,
		WorkerID: workerID,
	}
}

type GetUserInfoCmd struct {
	Username    string `json:"username"`
	WithAddress *bool  `json:"with_address" jsonrpcdefault:"false"`
}

func NewGetUserInfoCmd(username string, withAddress *bool) *GetUserInfoCmd {
	return &GetUserInfoCmd{
		Username:    username,
		WithAddress: withAddress,
	}
}

// GetUserNumCmd defines the getusernum JSON-RPC command.
type GetUserNumCmd struct{}

// NewGetUserNumCmd returns a new instance which can be used to issue a
// getusernum JSON-RPC command.
func NewGetUserNumCmd() *GetUserNumCmd {
	return &GetUserNumCmd{}
}

// GetUserInfosCmd defines the getuserinfos JSON-RPC command.
type GetUserInfosCmd struct {
	Page          int
	Num           int
	WithAddress   *bool `json:"with_address" jsonrpcdefault:"false"`
	PositiveOrder *bool `json:"positive_order" jsonrpcdefault:"false"`
}

// NewGetUserInfosCmd returns a new instance which can be used to issue a
// getuserinfos JSON-RPC command.
func NewGetUserInfosCmd(page int, num int, withAddress *bool, positiveOrder *bool) *GetUserInfosCmd {
	return &GetUserInfosCmd{
		Page:          page,
		Num:           num,
		WithAddress:   withAddress,
		PositiveOrder: positiveOrder,
	}
}

// GetMinerNumCmd defines the getminernum JSON-RPC command.
type GetMinerNumCmd struct{}

// NewGetMinerNumCmd returns a new instance which can be used to issue a
// getminernum JSON-RPC command.
func NewGetMinerNumCmd() *GetMinerNumCmd {
	return &GetMinerNumCmd{}
}

// GetMetaInfoCmd defines the getmetainfo JSON-RPC command.
type GetMetaInfoCmd struct{}

// NewGetMetaInfoCmd returns a new instance which can be used to issue a
// getmetainfo JSON-RPC command.
func NewGetMetaInfoCmd() *GetMetaInfoCmd {
	return &GetMetaInfoCmd{}
}

// GetMinersCmd defines the getminers JSON-RPC command.
type GetMinersCmd struct{}

// NewGetMinersCmd returns a new instance which can be used to issue a
// getminers JSON-RPC command.
func NewGetMinersCmd() *GetMinersCmd {
	return &GetMinersCmd{}
}

// GetMinerInfoCmd defines the getminerinfo JSON-RPC command.
type GetMinerInfoCmd struct {
	RemoteAddress string `json:"remote_address"`
}

// NewGetMinerInfoCmd returns a new instance which can be used to issue a
// getminerinfo JSON-RPC command.
func NewGetMinerInfoCmd(remoteAddress string) *GetMinerInfoCmd {
	return &GetMinerInfoCmd{
		RemoteAddress: remoteAddress,
	}
}

// GetMinerInfosCmd defines the getminerinfos JSON-RPC command.
type GetMinerInfosCmd struct{}

// NewGetMinerInfosCmd returns a new instance which can be used to issue a
// getminerinfos JSON-RPC command.
func NewGetMinerInfosCmd() *GetMinerInfosCmd {
	return &GetMinerInfosCmd{}
}

// CheckUsernameExistCmd defines the checkusernameexist JSON-RPC command.
type CheckUsernameExistCmd struct {
	Username string `json:"username"`
}

// NewCheckUsernameExistCmd returns a new instance which can be used to issue a
// checkusernameexist JSON-RPC command.
func NewCheckUsernameExistCmd(username string) *CheckUsernameExistCmd {
	return &CheckUsernameExistCmd{
		Username: username,
	}
}

// RegisterUserCmd defines the registeruser JSON-RPC command.
type RegisterUserCmd struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Address  string `json:"address"`
}

// NewRegisterUserCmd returns a new instance which can be used to issue a
// registeruser JSON-RPC command.
func NewRegisterUserCmd(username string, password string, address string) *RegisterUserCmd {
	return &RegisterUserCmd{
		Username: username,
		Password: password,
		Address:  address,
	}
}

// ChangePasswordCmd defines the changepassword JSON-RPC command.
type ChangePasswordCmd struct {
	Username    string `json:"username"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// NewChangePasswordCmd returns a new instance which can be used to issue a
// changepassword JSON-RPC command.
func NewChangePasswordCmd(username string, oldPassword string, newPassword string) *ChangePasswordCmd {
	return &ChangePasswordCmd{
		Username:    username,
		OldPassword: oldPassword,
		NewPassword: newPassword,
	}
}

// ResetPasswordCmd defines the resetpassword JSON-RPC command.
type ResetPasswordCmd struct {
	Username    string `json:"username"`
	NewPassword string `json:"new_password"`
}

// NewResetPasswordCmd returns a new instance which can be used to issue a
// resetpassword JSON-RPC command.
func NewResetPasswordCmd(username string, newPassword string) *ResetPasswordCmd {
	return &ResetPasswordCmd{
		Username:    username,
		NewPassword: newPassword,
	}
}

// CheckPasswordCmd defines the checkpassword JSON-RPC command.
type CheckPasswordCmd struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// NewCheckPasswordCmd returns a new instance which can be used to issue a
// checkpassword JSON-RPC command.
func NewCheckPasswordCmd(username string, password string) *CheckPasswordCmd {
	return &CheckPasswordCmd{
		Username: username,
		Password: password,
	}
}

// WalletPassphraseCmd defines the walletpassphrase JSON-RPC command.
type WalletPassphraseCmd struct {
	Password string `json:"password"`
}

// NewWalletPassphraseCmd returns a new instance which can be used to issue a
// walletpassphrase JSON-RPC command.
func NewWalletPassphraseCmd(password string) *WalletPassphraseCmd {
	return &WalletPassphraseCmd{
		Password: password,
	}
}

// GetOrdersCmd defines the getorders JSON-RPC command.
type GetOrdersCmd struct {
	Page          int
	Num           int
	WithAddress   *bool `json:"with_address" jsonrpcdefault:"false"`
	PositiveOrder *bool `json:"positive_order" jsonrpcdefault:"false"`
}

// NewGetOrdersCmd returns a new instance which can be used to issue a
// getorders JSON-RPC command.
func NewGetOrdersCmd(page int, num int, withAddress *bool, positiveOrder *bool) *GetOrdersCmd {
	return &GetOrdersCmd{
		Page:          page,
		Num:           num,
		WithAddress:   withAddress,
		PositiveOrder: positiveOrder,
	}
}

// GetOrdersByUsernameCmd defines the getordersbyusername JSON-RPC command.
type GetOrdersByUsernameCmd struct {
	Username      string `json:"username"`
	Page          int    `json:"page"`
	Num           int    `json:"num"`
	WithAddress   *bool  `json:"with_address" jsonrpcdefault:"false"`
	PositiveOrder *bool  `json:"positive_order" jsonrpcdefault:"false"`
}

// NewGetOrdersByUsernameCmd returns a new instance which can be used to issue a
// getordersbyusername JSON-RPC command.
func NewGetOrdersByUsernameCmd(username string, page int, num int, withAddress *bool, positiveOrder *bool) *GetOrdersByUsernameCmd {
	return &GetOrdersByUsernameCmd{
		Username:      username,
		Page:          page,
		Num:           num,
		WithAddress:   withAddress,
		PositiveOrder: positiveOrder,
	}
}

// GetTransactionsCmd defines the getorders JSON-RPC command.
type GetTransactionsCmd struct {
	Page          int
	Num           int
	WithAddress   *bool `json:"with_address" jsonrpcdefault:"false"`
	PositiveOrder *bool `json:"positive_order" jsonrpcdefault:"false"`
}

// NewGetTransactionsCmd returns a new instance which can be used to issue a
// gettransactions JSON-RPC command.
func NewGetTransactionsCmd(page int, num int, withAddress *bool, positiveOrder *bool) *GetTransactionsCmd {
	return &GetTransactionsCmd{
		Page:          page,
		Num:           num,
		WithAddress:   withAddress,
		PositiveOrder: positiveOrder,
	}
}

type GetCurrentBlockTemplateCmd struct{}

func NewGetCurrentBlockTemplateCmd() *GetCurrentBlockTemplateCmd {
	return &GetCurrentBlockTemplateCmd{}
}

type GetMinedBlocksCmd struct {
	Page          int   `json:"page"`
	Num           int   `json:"num"`
	PositiveOrder *bool `json:"positive_order" jsonrpcdefault:"false"`
}

func NewGetMindBlocksCmd(page int, num int, positiveOrder *bool) *GetMinedBlocksCmd {
	return &GetMinedBlocksCmd{
		Page:          page,
		Num:           num,
		PositiveOrder: positiveOrder,
	}
}

type GetUserShareInfoByUsernameCmd struct {
	Username      string `json:"username"`
	Page          int    `json:"page"`
	Num           int    `json:"num"`
	PositiveOrder *bool  `json:"positive_order" jsonrpcdefault:"false"`
}

func NewGetUserShareInfoByUsername(username string, page int, num int, positiveOrder *bool) *GetUserShareInfoByUsernameCmd {
	return &GetUserShareInfoByUsernameCmd{
		Username:      username,
		Page:          page,
		Num:           num,
		PositiveOrder: positiveOrder,
	}
}

type GetUserShareInfoCmd struct {
	Page          int   `json:"page"`
	Num           int   `json:"num"`
	PositiveOrder *bool `json:"positive_order" jsonrpcdefault:"false"`
}

func NewGetUserShareInfo(page int, num int, positiveOrder *bool) *GetUserShareInfoCmd {
	return &GetUserShareInfoCmd{
		Page:          page,
		Num:           num,
		PositiveOrder: positiveOrder,
	}
}

type GetAllocationInfoCmd struct {
	Page          int   `json:"page"`
	Num           int   `json:"num"`
	PositiveOrder *bool `json:"positive_order" jsonrpcdefault:"false"`
}

func NewGetAllocationInfoCmd(page int, num int, positiveOrder *bool) *GetAllocationInfoCmd {
	return &GetAllocationInfoCmd{
		Page:          page,
		Num:           num,
		PositiveOrder: positiveOrder,
	}
}

type GetAllocationInfoByUsernameCmd struct {
	Username      string `json:"username"`
	Page          int    `json:"page"`
	Num           int    `json:"num"`
	PositiveOrder *bool  `json:"positive_order" jsonrpcdefault:"false"`
}

func NewGetAllocationInfoByUsernameCmd(username string, page int, num int, positiveOrder *bool) *GetAllocationInfoByUsernameCmd {
	return &GetAllocationInfoByUsernameCmd{
		Username:      username,
		Page:          page,
		Num:           num,
		PositiveOrder: positiveOrder,
	}
}

type GetHashRateCmd struct{}

func NewGetHashRateCmd() *GetHashRateCmd {
	return &GetHashRateCmd{}
}

type GetBlacklistCmd struct{}

func NewGetBlacklistCmd() *GetBlacklistCmd {
	return &GetBlacklistCmd{}
}

type GetWhitelistCmd struct{}

func NewGetWhitelistCmd() *GetWhitelistCmd {
	return &GetWhitelistCmd{}
}

type GetAgentBlacklistCmd struct{}

func NewGetAgentBlacklistCmd() *GetAgentBlacklistCmd {
	return &GetAgentBlacklistCmd{}
}

type GetAgentWhitelistCmd struct{}

func NewGetAgentWhitelistCmd() *GetAgentWhitelistCmd {
	return &GetAgentWhitelistCmd{}
}

type GetAdminBlacklistCmd struct{}

func NewGetAdminBlacklistCmd() *GetAdminBlacklistCmd {
	return &GetAdminBlacklistCmd{}
}

type GetAdminWhitelistCmd struct{}

func NewGetAdminWhitelistCmd() *GetAdminWhitelistCmd {
	return &GetAdminWhitelistCmd{}
}

type GetStatusCmd struct{}

func NewGetStatusCmd() *GetStatusCmd {
	return &GetStatusCmd{}
}

type AddAgentBlacklistCmd struct {
	Agent string `json:"agent"`
}

type DeleteAgentBlacklistCmd struct {
	Agent string `json:"agent"`
}

type AddAgentWhitelistCmd struct {
	Agent string `json:"agent"`
}

type DeleteAgentWhitelistCmd struct {
	Agent string `json:"agent"`
}

type AddBlacklistCmd struct {
	IP string `json:"ip"`
}

type DeleteBlacklistCmd struct {
	IP string `json:"ip"`
}

type AddWhitelistCmd struct {
	IP string `json:"ip"`
}

type DeleteWhitelistCmd struct {
	IP string `json:"ip"`
}

type AddAdminBlacklistCmd struct {
	IP string `json:"ip"`
}

type DeleteAdminBlacklistCmd struct {
	IP string `json:"ip"`
}

type AddAdminWhitelistCmd struct {
	IP string `json:"ip"`
}

type DeleteAdminWhitelistCmd struct {
	IP string `json:"ip"`
}

type SetRewardIntervalCmd struct {
	RewardInterval int `json:"reward_interval"`
}

type SetRewardMaturityCmd struct {
	RewardMaturity int `json:"reward_maturity"`
}

type SetManageFeePercentCmd struct {
	ManageFeePercent int `json:"manage_fee_percent"`
}

func init() {
	// No special flags for commands in this file.
	flags := UsageFlag(0)

	// Common command
	MustRegisterCmd("version", (*VersionCmd)(nil), flags)
	MustRegisterCmd("authenticate", (*AuthenticateCmd)(nil), flags)

	// AbelianStratum command
	MustRegisterCmd("mining.hello", (*HelloCmd)(nil), flags)
	MustRegisterCmd("mining.noop", (*NoopCmd)(nil), flags)
	MustRegisterCmd("mining.subscribe", (*SubscribeCmd)(nil), flags)
	// MustRegisterCmd("mining.authorize", (*AuthorizeCmd)(nil), flags), for v2

	MustRegisterCmd("mining.submit", (*SubmitCmd)(nil), flags)
	MustRegisterCmd("mining.hashrate", (*HashRateCmd)(nil), flags)

	// E9 miner
	MustRegisterCmd("eth_submitLogin", (*EthSubmitLogin)(nil), flags)

	// Extended command
	MustRegisterCmd("mining.get_job", (*GetJobCmd)(nil), flags)

	MustRegisterCmd("getmetainfo", (*GetMetaInfoCmd)(nil), flags)
	MustRegisterCmd("gethashrate", (*GetHashRateCmd)(nil), flags)

	MustRegisterCmd("getusernum", (*GetUserNumCmd)(nil), flags)
	MustRegisterCmd("getuserinfo", (*GetUserInfoCmd)(nil), flags)
	MustRegisterCmd("getuserinfobyusername", (*GetUserInfoCmd)(nil), flags)
	MustRegisterCmd("getuserinfos", (*GetUserInfosCmd)(nil), flags)

	MustRegisterCmd("getminernum", (*GetMinerNumCmd)(nil), flags)
	MustRegisterCmd("getminers", (*GetMinersCmd)(nil), flags)
	MustRegisterCmd("getminerinfo", (*GetMinerInfoCmd)(nil), flags)
	MustRegisterCmd("getminerinfobyremoteaddress", (*GetMinerInfoCmd)(nil), flags)
	MustRegisterCmd("getminerinfos", (*GetMinerInfosCmd)(nil), flags)

	MustRegisterCmd("getorders", (*GetOrdersCmd)(nil), flags)
	MustRegisterCmd("getordersbyusername", (*GetOrdersByUsernameCmd)(nil), flags)

	MustRegisterCmd("gettransactions", (*GetTransactionsCmd)(nil), flags)

	MustRegisterCmd("registeruser", (*RegisterUserCmd)(nil), flags)
	MustRegisterCmd("checkusernameexist", (*CheckUsernameExistCmd)(nil), flags)
	MustRegisterCmd("checkpassword", (*CheckPasswordCmd)(nil), flags)
	MustRegisterCmd("changepassword", (*ChangePasswordCmd)(nil), flags)
	MustRegisterCmd("resetpassword", (*ResetPasswordCmd)(nil), flags)
	MustRegisterCmd("walletpassphrase", (*WalletPassphraseCmd)(nil), flags)

	MustRegisterCmd("getcurrentblocktemplate", (*GetCurrentBlockTemplateCmd)(nil), flags)

	MustRegisterCmd("getminedblocks", (*GetMinedBlocksCmd)(nil), flags)

	MustRegisterCmd("getusershareinfo", (*GetUserShareInfoCmd)(nil), flags)
	MustRegisterCmd("getusershareinfobyusername", (*GetUserShareInfoByUsernameCmd)(nil), flags)

	MustRegisterCmd("getallocationinfo", (*GetAllocationInfoCmd)(nil), flags)
	MustRegisterCmd("getallocationinfobyusername", (*GetAllocationInfoByUsernameCmd)(nil), flags)

	MustRegisterCmd("getagentblacklist", (*GetAgentBlacklistCmd)(nil), flags)
	MustRegisterCmd("getagentwhitelist", (*GetAgentWhitelistCmd)(nil), flags)
	MustRegisterCmd("getblacklist", (*GetBlacklistCmd)(nil), flags)
	MustRegisterCmd("getwhitelist", (*GetWhitelistCmd)(nil), flags)
	MustRegisterCmd("getadminblacklist", (*GetAdminBlacklistCmd)(nil), flags)
	MustRegisterCmd("getadminwhitelist", (*GetAdminWhitelistCmd)(nil), flags)
	MustRegisterCmd("addagentblacklist", (*AddAgentBlacklistCmd)(nil), flags)
	MustRegisterCmd("deleteagentblacklist", (*DeleteAgentBlacklistCmd)(nil), flags)
	MustRegisterCmd("addagentwhitelist", (*AddAgentWhitelistCmd)(nil), flags)
	MustRegisterCmd("deleteagentwhitelist", (*DeleteAgentWhitelistCmd)(nil), flags)
	MustRegisterCmd("addblacklist", (*AddBlacklistCmd)(nil), flags)
	MustRegisterCmd("deleteblacklist", (*DeleteBlacklistCmd)(nil), flags)
	MustRegisterCmd("addwhitelist", (*AddWhitelistCmd)(nil), flags)
	MustRegisterCmd("deletewhitelist", (*DeleteWhitelistCmd)(nil), flags)
	MustRegisterCmd("addadminblacklist", (*AddAdminBlacklistCmd)(nil), flags)
	MustRegisterCmd("deleteadminblacklist", (*DeleteAdminBlacklistCmd)(nil), flags)
	MustRegisterCmd("addadminwhitelist", (*AddAdminWhitelistCmd)(nil), flags)
	MustRegisterCmd("deleteadminwhitelist", (*DeleteAdminWhitelistCmd)(nil), flags)

	MustRegisterCmd("getstatus", (*GetStatusCmd)(nil), flags)

	MustRegisterCmd("setrewardinterval", (*SetRewardIntervalCmd)(nil), flags)
	MustRegisterCmd("setrewardmaturity", (*SetRewardMaturityCmd)(nil), flags)
	MustRegisterCmd("setmanagefeepercent", (*SetManageFeePercentCmd)(nil), flags)
}

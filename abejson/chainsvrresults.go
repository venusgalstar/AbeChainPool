package abejson

// GetBlockTemplateResultTxAbe models the transactions field of the
// getblocktemplate command.
type GetBlockTemplateResultTxAbe struct {
	Data   string `json:"data"`
	TxHash string `json:"txhash"`

	WitnessHash string `json:"witnesshash"`
	Fee         uint64 `json:"fee"`
	Size        int64  `json:"size"`
}

// GetBlockTemplateResultCoinbase models the coinbase transaction field of the
// getblocktemplate command.
type GetBlockTemplateResultCoinbase struct {
	Data             string `json:"data"`
	TxHash           string `json:"txhash"`
	WitnessHash      string `json:"witnesshash"`
	Fee              uint64 `json:"fee"`
	Size             int64  `json:"size"`
	ExtraNonceOffset int64  `json:"extranonceoffset"`
	ExtraNonceLen    int64  `json:"extranoncelen"`
	WitnessOffset    int64  `json:"witness_offset"`
}

// GetBlockTemplateResult models the data returned from the getblocktemplate
// command.
type GetBlockTemplateResult struct {
	// CoinbaseAux is optional.  One of
	// CoinbaseTxn or CoinbaseValue must be specified, but not both.
	Bits         string                          `json:"bits"`
	CurTime      int64                           `json:"curtime"`
	Height       int64                           `json:"height"`
	PreviousHash string                          `json:"previousblockhash"`
	SizeLimit    int64                           `json:"sizelimit,omitempty"`
	Transactions []GetBlockTemplateResultTxAbe   `json:"transactions"`
	Version      int32                           `json:"version"`
	CoinbaseTxn  *GetBlockTemplateResultCoinbase `json:"coinbasetxn,omitempty"`
	WorkID       string                          `json:"workid,omitempty"`

	// Witness commitment defined in BIP 0141.
	DefaultWitnessCommitment string `json:"default_witness_commitment,omitempty"`

	// Optional long polling from BIP 0022.
	LongPollID  string `json:"longpollid,omitempty"`
	LongPollURI string `json:"longpolluri,omitempty"`
	SubmitOld   *bool  `json:"submitold,omitempty"`

	// Basic pool extension from BIP 0023.
	Target  string `json:"target,omitempty"`
	Expires int64  `json:"expires,omitempty"`

	// Mutations from BIP 0023.
	MaxTime    int64    `json:"maxtime,omitempty"`
	MinTime    int64    `json:"mintime,omitempty"`
	Mutable    []string `json:"mutable,omitempty"`
	NonceRange string   `json:"noncerange,omitempty"`

	// Block proposal from BIP 0023.
	Capabilities  []string `json:"capabilities,omitempty"`
	RejectReasion string   `json:"reject-reason,omitempty"`
}

// VersionResult models objects included in the version response.  In the actual
// result, these objects are keyed by the program or API name.
type VersionResult struct {
	VersionString string `json:"versionstring"`
	Major         uint32 `json:"major"`
	Minor         uint32 `json:"minor"`
	Patch         uint32 `json:"patch"`
	Prerelease    string `json:"prerelease"`
	BuildMetadata string `json:"buildmetadata"`
}

package chaincfg

// Params is used to group parameters for various networks such as the main
// network and test networks.
type Params struct {
	Name                     string
	DefaultPort              string
	CoinbaseMaturity         uint16
	RPCClientPort            string
	WalletRPCClientPort      string
	BlockHeightEthashPoW     int32
	EthashEpochLength        int32
	PQRingCTID               byte
	SubsidyReductionInterval int32
}

// MainNetParams contains parameters on the main network
var MainNetParams = Params{
	Name:                     "mainnet",
	DefaultPort:              "8666",
	CoinbaseMaturity:         200,
	RPCClientPort:            "8667",
	WalletRPCClientPort:      "8665",
	BlockHeightEthashPoW:     56000,
	EthashEpochLength:        4000,
	PQRingCTID:               0x00,
	SubsidyReductionInterval: 400_000,
}

// TestNet3Params contains parameters on the test network
var TestNet3Params = Params{
	Name:                     "testnet3",
	DefaultPort:              "18666",
	CoinbaseMaturity:         6,
	RPCClientPort:            "18667",
	WalletRPCClientPort:      "18665",
	BlockHeightEthashPoW:     300,
	EthashEpochLength:        200,
	PQRingCTID:               0x02,
	SubsidyReductionInterval: 400_000,
}

// SimNetParams contains parameters specific to the simulation test network
var SimNetParams = Params{
	Name:                     "simnet",
	DefaultPort:              "18888",
	CoinbaseMaturity:         200,
	RPCClientPort:            "18889",
	WalletRPCClientPort:      "18887",
	BlockHeightEthashPoW:     300,
	EthashEpochLength:        200,
	PQRingCTID:               0x03,
	SubsidyReductionInterval: 400_000,
}

// RegNetParams contains parameters specific to the regression test network
var RegNetParams = Params{
	Name:             "regtest",
	DefaultPort:      "18777",
	CoinbaseMaturity: 200,
	RPCClientPort:    "18667",
	//WalletRPCClientPort:  "", // wallet not yet compatible with regtest
	BlockHeightEthashPoW:     300,
	EthashEpochLength:        200,
	PQRingCTID:               0x01,
	SubsidyReductionInterval: 400_000,
}

var ActiveNetParams = &MainNetParams

var AbecBackendVersion = "unknown"
var AbewalletBackendVersion = "unknown"

var PoolBackendVersion = "unknown"

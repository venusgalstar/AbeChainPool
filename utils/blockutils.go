package utils

import (
	"github.com/abesuite/abe-miningpool-server/chaincfg"
)

const baseSubsidy = 256 * uint64(10_000_000)

func CalculateEpoch(blockHeight int64) int64 {
	if blockHeight-int64(chaincfg.ActiveNetParams.BlockHeightEthashPoW) < 0 {
		return -1
	}
	return (blockHeight - int64(chaincfg.ActiveNetParams.BlockHeightEthashPoW)) / int64(chaincfg.ActiveNetParams.EthashEpochLength)
}

func CalculateEpochStartHeight(blockHeight int64) int64 {
	if blockHeight-int64(chaincfg.ActiveNetParams.BlockHeightEthashPoW) < 0 {
		return -1
	}

	return blockHeight - (blockHeight-int64(chaincfg.ActiveNetParams.BlockHeightEthashPoW))%int64(chaincfg.ActiveNetParams.EthashEpochLength)
}

func CalculateEpochEndHeight(blockHeight int64) int64 {
	if blockHeight-int64(chaincfg.ActiveNetParams.BlockHeightEthashPoW) < 0 {
		return -1
	}

	return blockHeight - (blockHeight-int64(chaincfg.ActiveNetParams.BlockHeightEthashPoW))%int64(chaincfg.ActiveNetParams.EthashEpochLength) +
		int64(chaincfg.ActiveNetParams.EthashEpochLength)
}

func IsEpochStart(blockHeight int64) bool {
	if blockHeight-int64(chaincfg.ActiveNetParams.BlockHeightEthashPoW) < 0 {
		return false
	}

	return (blockHeight-int64(chaincfg.ActiveNetParams.BlockHeightEthashPoW))%int64(chaincfg.ActiveNetParams.EthashEpochLength) == 0
}

func CalculateCoinbaseRewardByBlockHeight(blockHeight int64) uint64 {
	if chaincfg.ActiveNetParams.SubsidyReductionInterval == 0 {
		return baseSubsidy
	}

	return baseSubsidy >> uint(blockHeight/int64(chaincfg.ActiveNetParams.SubsidyReductionInterval))
}
func CalculateCoinbaseRewardByEpoch(epoch int64) uint64 {
	if epoch < 0 {
		panic("the epoch number is invalid")
	}
	// [56000,60000)
	epochStartHeight := int64(chaincfg.ActiveNetParams.BlockHeightEthashPoW) + epoch*int64(chaincfg.ActiveNetParams.EthashEpochLength)

	return baseSubsidy >> uint(epochStartHeight/int64(chaincfg.ActiveNetParams.SubsidyReductionInterval))
}

package model

import "github.com/abesuite/abe-miningpool-server/dal/do"

func ConvertShareInfoToDO(shareInfo *ShareInfo) *do.DetailedShareInfo {
	if shareInfo == nil {
		return nil
	}
	return &do.DetailedShareInfo{
		Username:    shareInfo.Username,
		ShareCount:  shareInfo.ShareCount,
		SealHash:    shareInfo.SealHash,
		Height:      shareInfo.Height,
		ShareHash:   shareInfo.ShareHash,
		ContentHash: shareInfo.ContentHash,
		Nonce:       shareInfo.Nonce,
	}
}

func ConvertShareInfoToMinedBlockDO(shareInfo *ShareInfo) *do.MinedBlockInfo {
	if shareInfo == nil {
		return nil
	}
	return &do.MinedBlockInfo{
		Username:    shareInfo.Username,
		Height:      shareInfo.Height,
		BlockHash:   shareInfo.ShareHash,
		SealHash:    shareInfo.SealHash,
		ContentHash: shareInfo.ContentHash,
		Reward:      shareInfo.Reward,
	}
}

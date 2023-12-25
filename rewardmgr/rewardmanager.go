package rewardmgr

import (
	"context"

	"github.com/abesuite/abe-miningpool-server/chaincfg"
	"github.com/abesuite/abe-miningpool-server/chainclient"
	"github.com/abesuite/abe-miningpool-server/dal"
	"github.com/abesuite/abe-miningpool-server/dal/do"
	"github.com/abesuite/abe-miningpool-server/model"
	"github.com/abesuite/abe-miningpool-server/service"

	"gorm.io/gorm"
)

type RewardManager struct {
	autoAllocate  bool
	rewardConfig  *model.RewardConfig
	currentHeight int64
	chain         *chainclient.RPCClient
	db            *gorm.DB
}

func NewRewardManager(config *model.RewardConfig, autoAllocate bool, chainClient *chainclient.RPCClient) *RewardManager {
	return &RewardManager{
		autoAllocate:  autoAllocate,
		rewardConfig:  config,
		currentHeight: -1,
		db:            dal.GlobalDBClient,
		chain:         chainClient,
	}
}

// HandleChainClientNotification handles notifications from chain client.
func (m *RewardManager) HandleChainClientNotification(notification *chainclient.Notification) {
	switch notification.Type {

	case chainclient.NTBlockTemplateChanged:
		log.Trace("Reward manager receives a new block template from chain client, updating...")
		blockTemplate, ok := notification.Data.(*model.BlockTemplate)
		if !ok {
			log.Errorf("Reward manager accepted notification is not a block template.")
			break
		}
		if blockTemplate == nil {
			log.Errorf("Received block template is nil")
			break
		}

		if m.db == nil {
			log.Errorf("DB is nil in reward manager.")
			return
		}
		userService := service.GetUserService()
		// Scan unallocated share when start.
		if m.currentHeight == -1 {
			endHeight := int(blockTemplate.Height) - m.rewardConfig.RewardMaturity - 1
			if endHeight > 0 {
				// Find and allocate reward that is missing before.
				unallocatedRanges, err := userService.GetUnallocatedRanges(context.Background(), m.db, endHeight)
				if err != nil {
					log.Errorf("Internal error: unable to get unallocated ranges, %v", err)
					return
				}
				for heightPair := range unallocatedRanges {
					log.Infof("Allocating reward in height [%v, %v)", heightPair.StartHeight, heightPair.EndHeight)
					err := m.allocateBetweenHeight(int(heightPair.StartHeight), int(heightPair.EndHeight))
					if err != nil {
						return
					}
				}
			}
		}

		heightChanged := false
		if m.currentHeight != blockTemplate.Height {
			heightChanged = true
			m.currentHeight = blockTemplate.Height
		}

		// Allocate reward if matches the height.
		if !heightChanged || !m.autoAllocate {
			return
		}

		if (int(m.currentHeight)-m.rewardConfig.RewardMaturity-int(chaincfg.ActiveNetParams.BlockHeightEthashPoW))%m.rewardConfig.RewardInterval == 0 {
			endHeight := int(m.currentHeight) - m.rewardConfig.RewardMaturity
			if endHeight <= 0 {
				return
			}

			// Find and allocate reward that has not been allocated before.
			unallocatedRanges, err := userService.GetUnallocatedRanges(context.Background(), m.db, endHeight)
			if err != nil {
				log.Errorf("Internal error: unable to get unallocated ranges, %v", err)
				return
			}
			for heightPair := range unallocatedRanges {
				log.Infof("Allocating reward in height [%v, %v)", heightPair.StartHeight, heightPair.EndHeight)
				err := m.allocateBetweenHeight(int(heightPair.StartHeight), int(heightPair.EndHeight))
				if err != nil {
					return
				}
			}
		}
	}
}

func (m *RewardManager) checkConnectedBlocks(connectedBlocks []*do.MinedBlockInfo) ([]*do.MinedBlockInfo, []*do.MinedBlockInfo, error) {
	connected := make([]*do.MinedBlockInfo, 0)
	disconnected := make([]*do.MinedBlockInfo, 0)
	blockNum := len(connectedBlocks)
	for i := 0; i < blockNum; i++ {
		block := connectedBlocks[i]
		realBlockHash, err := m.chain.GetBlockHash(block.Height, 10)
		if err != nil {
			log.Errorf("Unable to get block hash from chain: %v", err)
			return nil, nil, err
		}
		if realBlockHash.String() == block.BlockHash {
			connected = append(connected, block)
		} else {
			disconnected = append(disconnected, block)
		}
	}
	return connected, disconnected, nil
}

func (m *RewardManager) checkAcceptedBlocks(acceptedBlocks []*do.MinedBlockInfo) ([]*do.MinedBlockInfo, []*do.MinedBlockInfo, error) {
	connected := make([]*do.MinedBlockInfo, 0)
	disconnected := make([]*do.MinedBlockInfo, 0)
	blockNum := len(acceptedBlocks)
	for i := 0; i < blockNum; i++ {
		block := acceptedBlocks[i]
		realBlockHash, err := m.chain.GetBlockHash(block.Height, 10)
		if err != nil {
			log.Errorf("Unable to get block hash from chain: %v", err)
			return nil, nil, err
		}
		if realBlockHash.String() == block.BlockHash {
			connected = append(connected, block)
		} else {
			disconnected = append(disconnected, block)
		}
	}
	return connected, disconnected, nil
}

func (m *RewardManager) allocateBetweenHeight(startHeight int, endHeight int) error {
	userService := service.GetUserService()
	configService := service.GetConfigInfoService()

	// Find blocks that are mined and accepted between startHeight and endHeight.
	acceptedBlocks, err := userService.GetAcceptedBlocksByHeight(context.Background(), m.db, startHeight, endHeight)
	if err != nil {
		log.Errorf("Internal error: unable to get accepted blocks by height, %v", err)
		return err
	}

	// Check the connected state of these blocks.
	connectedBlocks, disconnectedBlocks, err := m.checkAcceptedBlocks(acceptedBlocks)
	if err != nil {
		log.Errorf("Internal error: unable to check accepted blocks, %v", err)
		return err
	}

	connectedMap := make(map[string]struct{})
	disconnectedMap := make(map[string]struct{})
	for _, block := range connectedBlocks {
		connectedMap[block.BlockHash] = struct{}{}
	}
	for _, block := range disconnectedBlocks {
		disconnectedMap[block.BlockHash] = struct{}{}
	}

	// Refresh the state of accepted blocks.
	for _, block := range acceptedBlocks {
		if block.Connected == 1 {
			if _, ok := disconnectedMap[block.BlockHash]; ok {
				log.Debugf("Block %v state: connected -> disconnected", block.BlockHash)
				err := userService.DisconnectBlock(context.Background(), m.db, block.BlockHash)
				if err != nil {
					log.Errorf("Internal error: unable to disconnect block %v: %v", block.BlockHash, err)
					return err
				}
			}
		} else if block.Disconnected == 1 {
			if _, ok := connectedMap[block.BlockHash]; ok {
				log.Debugf("Block %v state: disconnected -> connected", block.BlockHash)
				err := userService.ConnectBlock(context.Background(), m.db, block.BlockHash)
				if err != nil {
					log.Errorf("Internal error: unable to connect block %v: %v", block.BlockHash, err)
					return err
				}
			}
		} else {
			if _, ok := connectedMap[block.BlockHash]; ok {
				log.Debugf("Block %v state: unknown -> connected", block.BlockHash)
				err := userService.ConnectBlock(context.Background(), m.db, block.BlockHash)
				if err != nil {
					log.Errorf("Internal error: unable to connect block %v: %v", block.BlockHash, err)
					return err
				}
			} else {
				log.Debugf("Block %v state: unknown -> disconnected", block.BlockHash)
				err := userService.DisconnectBlock(context.Background(), m.db, block.BlockHash)
				if err != nil {
					log.Errorf("Internal error: unable to disconnect block %v: %v", block.BlockHash, err)
					return err
				}
			}
		}
	}

	// Fetch manage fee ratio.
	realManageFeePercent := m.rewardConfig.ManageFeePercent
	currentConfig, err := configService.GetByHeight(context.Background(), m.db, int64(startHeight))
	if err != nil || currentConfig == nil {
		log.Warnf("Unable to find corresponding config info with height %v, use default %v", startHeight, m.rewardConfig.ManageFeePercent)
	} else {
		realManageFeePercent = currentConfig.ManageFeePercent
	}

	err = userService.AllocateRewardsWithBlocks(context.Background(), m.db, startHeight, endHeight, realManageFeePercent, connectedBlocks)
	if err != nil {
		log.Errorf("Internal error: unable to allocate rewards, %v", err)
		return err
	}
	return nil
}

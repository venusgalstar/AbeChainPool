package configmgr

import (
	"context"
	"sync"

	"github.com/abesuite/abe-miningpool-server/chainclient"
	"github.com/abesuite/abe-miningpool-server/dal"
	"github.com/abesuite/abe-miningpool-server/dal/do"
	"github.com/abesuite/abe-miningpool-server/model"
	"github.com/abesuite/abe-miningpool-server/service"
	"github.com/abesuite/abe-miningpool-server/utils"

	"gorm.io/gorm"
)

type ConfigManager struct {
	rewardConfig  *model.RewardConfig
	currentHeight int64
	configLock    sync.Mutex
	db            *gorm.DB
}

func NewConfigManager(config *model.RewardConfig) *ConfigManager {
	return &ConfigManager{
		rewardConfig:  config,
		currentHeight: -1,
		db:            dal.GlobalDBClient,
	}
}

// HandleChainClientNotification handles notifications from chain client.
func (m *ConfigManager) HandleChainClientNotification(notification *chainclient.Notification) {
	switch notification.Type {

	case chainclient.NTBlockTemplateChanged:
		log.Trace("RewardConfig manager receives a new block template from chain client, updating...")
		blockTemplate, ok := notification.Data.(*model.BlockTemplate)
		if !ok {
			log.Errorf("RewardConfig manager accepted notification is not a block template.")
			break
		}
		if blockTemplate == nil {
			log.Errorf("Received block template is nil")
			break
		}

		height := blockTemplate.Height
		if height < 0 {
			log.Errorf("Invalid height %v received in config manager.", height)
			return
		}
		if m.db == nil {
			log.Errorf("DB is nil in config manager.")
			return
		}

		needCreate := false
		if m.currentHeight == -1 || utils.IsEpochStart(height) || height%5 == 0 {
			needCreate = true
		}
		m.currentHeight = height

		// Check config record and create if not exist.
		if needCreate {
			m.configLock.Lock()
			defer m.configLock.Unlock()

			configService := service.GetConfigInfoService()
			epoch := utils.CalculateEpoch(height)
			if epoch < 0 {
				log.Errorf("Epoch with height %d not supported", height)
				return
			}
			info, err := configService.GetByEpoch(context.Background(), m.db, epoch)
			if err != nil {
				log.Errorf("Error when getting config info by epoch: %v", err)
				return
			}
			if info == nil {
				log.Infof("RewardConfig record at height %v, epoch %v not found, creating...", height, epoch)
				startHeight := utils.CalculateEpochStartHeight(height)
				endHeight := utils.CalculateEpochEndHeight(height)
				realRewardInterval := m.rewardConfig.RewardInterval
				if m.currentHeight < m.rewardConfig.ConfigChangeHeight {
					realRewardInterval = m.rewardConfig.RewardIntervalPre
				}
				newConfigInfo := do.ConfigInfo{
					Epoch:            epoch,
					StartHeight:      startHeight,
					EndHeight:        endHeight,
					RewardInterval:   int64(realRewardInterval),
					RewardMaturity:   int64(m.rewardConfig.RewardMaturity),
					ManageFeePercent: m.rewardConfig.ManageFeePercent,
				}
				_, err := configService.Create(context.Background(), m.db, &newConfigInfo)
				if err != nil {
					log.Errorf("Error when creating config record: %v", err)
					return
				}
			}
		}
	}
}

func (m *ConfigManager) GetConfigByEpoch(epoch int64) (*do.ConfigInfo, error) {
	m.configLock.Lock()
	defer m.configLock.Unlock()

	configService := service.GetConfigInfoService()
	info, err := configService.GetByEpoch(context.Background(), m.db, epoch)
	if err != nil {
		log.Errorf("Error when getting config info by epoch: %v", err)
		return nil, err
	}
	return info, nil
}

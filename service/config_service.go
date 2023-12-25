package service

import (
	"context"
	"errors"

	"github.com/abesuite/abe-miningpool-server/dal/dao"
	"github.com/abesuite/abe-miningpool-server/dal/do"
	"github.com/abesuite/abe-miningpool-server/utils"

	"gorm.io/gorm"
)

type ConfigInfoService interface {
	Create(ctx context.Context, tx *gorm.DB, info *do.ConfigInfo) (int64, error)
	GetByEpoch(ctx context.Context, tx *gorm.DB, epoch int64) (*do.ConfigInfo, error)
	GetByHeight(ctx context.Context, tx *gorm.DB, height int64) (*do.ConfigInfo, error)
	GetLatestConfig(ctx context.Context, tx *gorm.DB) (*do.ConfigInfo, error)
}

type ConfigInfoServiceImpl struct {
	configInfoDao dao.ConfigInfoDAO
}

var configInfoService ConfigInfoService = &ConfigInfoServiceImpl{
	configInfoDao: dao.GetConfigInfoDAOImpl(),
}

func GetConfigInfoService() ConfigInfoService {
	return configInfoService
}

func (c *ConfigInfoServiceImpl) Create(ctx context.Context, tx *gorm.DB, info *do.ConfigInfo) (int64, error) {
	return c.configInfoDao.Create(ctx, tx, info)
}

func (c *ConfigInfoServiceImpl) GetByEpoch(ctx context.Context, tx *gorm.DB, epoch int64) (*do.ConfigInfo, error) {
	return c.configInfoDao.GetByEpoch(ctx, tx, epoch)
}

func (c *ConfigInfoServiceImpl) GetByHeight(ctx context.Context, tx *gorm.DB, height int64) (*do.ConfigInfo, error) {
	epoch := utils.CalculateEpoch(height)
	if epoch < 0 {
		return nil, errors.New("unsupported epoch")
	}
	return c.configInfoDao.GetByEpoch(ctx, tx, epoch)
}

func (c *ConfigInfoServiceImpl) GetLatestConfig(ctx context.Context, tx *gorm.DB) (*do.ConfigInfo, error) {
	return c.configInfoDao.GetLatestConfig(ctx, tx)
}

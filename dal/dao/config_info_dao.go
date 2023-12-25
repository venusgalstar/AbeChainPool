package dao

import (
	"context"
	"errors"
	"gorm.io/gorm"

	"github.com/abesuite/abe-miningpool-server/dal/do"
	"github.com/abesuite/abe-miningpool-server/errcode"
)

type ConfigInfoDAO interface {
	Create(ctx context.Context, tx *gorm.DB, info *do.ConfigInfo) (int64, error)
	GetByEpoch(ctx context.Context, tx *gorm.DB, epoch int64) (*do.ConfigInfo, error)
	GetLatestEpoch(ctx context.Context, tx *gorm.DB) (int64, error)
	GetLatestConfig(ctx context.Context, tx *gorm.DB) (*do.ConfigInfo, error)
}

type ConfigInfoDAOImpl struct{}

var configInfoDAO ConfigInfoDAO = &ConfigInfoDAOImpl{}

func GetConfigInfoDAOImpl() ConfigInfoDAO {
	return configInfoDAO
}

func (c *ConfigInfoDAOImpl) Create(ctx context.Context, tx *gorm.DB, info *do.ConfigInfo) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	if info == nil {
		return 0, errors.New("fail to create config info: nil config info")
	}

	query := tx.Create(info)
	return query.RowsAffected, query.Error
}

func (c *ConfigInfoDAOImpl) GetByEpoch(ctx context.Context, tx *gorm.DB, epoch int64) (*do.ConfigInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	info := do.ConfigInfo{}
	query := tx.Model(&do.ConfigInfo{}).Where("epoch = ?", epoch).Take(&info)
	if query.Error != nil {
		if errors.Is(query.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, query.Error
	}
	return &info, nil
}

func (c *ConfigInfoDAOImpl) GetLatestEpoch(ctx context.Context, tx *gorm.DB) (int64, error) {
	type MaxStruct struct {
		MaxValue int64 `json:"max_value"`
	}
	var result MaxStruct
	err := tx.Raw("SELECT MAX( epoch ) AS max_value FROM config_infos").Scan(&result).Error
	if err != nil {
		return 0, err
	}
	return result.MaxValue, nil
}

func (c *ConfigInfoDAOImpl) GetLatestConfig(ctx context.Context, tx *gorm.DB) (*do.ConfigInfo, error) {
	epoch, err := c.GetLatestEpoch(ctx, tx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	res, err := c.GetByEpoch(ctx, tx, epoch)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

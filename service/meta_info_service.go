package service

import (
	"context"

	"github.com/abesuite/abe-miningpool-server/dal/dao"
	"github.com/abesuite/abe-miningpool-server/dal/do"

	"gorm.io/gorm"
)

type MetaInfoService interface {
	Get(ctx context.Context, tx *gorm.DB) (*do.MetaInfo, error)
	AddBalance(ctx context.Context, tx *gorm.DB, balance int64) (int64, error)
	UpdateLastRewardHeight(ctx context.Context, tx *gorm.DB, height int64) (int64, error)
	UpdateLastTxCheckHeight(ctx context.Context, tx *gorm.DB, height int64) (int64, error)
	UpdateLastRedeemedHeight(ctx context.Context, tx *gorm.DB, height int64) (int64, error)
}

type MetaInfoServiceImpl struct {
	metaInfoDao dao.MetaInfoDAO
}

var metaInfoService MetaInfoService = &MetaInfoServiceImpl{
	metaInfoDao: dao.GetMetaInfoDAOImpl(),
}

func GetMetaInfoService() MetaInfoService {
	return metaInfoService
}

func (m *MetaInfoServiceImpl) Get(ctx context.Context, tx *gorm.DB) (*do.MetaInfo, error) {
	return m.metaInfoDao.Get(ctx, tx)
}

func (m *MetaInfoServiceImpl) AddBalance(ctx context.Context, tx *gorm.DB, balance int64) (int64, error) {
	return m.metaInfoDao.UpdateBalance(ctx, tx, balance)
}

func (m *MetaInfoServiceImpl) UpdateLastRewardHeight(ctx context.Context, tx *gorm.DB, height int64) (int64, error) {
	return m.metaInfoDao.UpdateLastRewardHeight(ctx, tx, height)
}

func (m *MetaInfoServiceImpl) UpdateLastTxCheckHeight(ctx context.Context, tx *gorm.DB, height int64) (int64, error) {
	return m.metaInfoDao.UpdateLastTxCheckHeight(ctx, tx, height)
}

func (m *MetaInfoServiceImpl) UpdateLastRedeemedHeight(ctx context.Context, tx *gorm.DB, height int64) (int64, error) {
	return m.metaInfoDao.UpdateLastRedeemedHeight(ctx, tx, height)
}

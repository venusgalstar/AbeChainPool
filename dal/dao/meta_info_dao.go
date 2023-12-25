package dao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/abesuite/abe-miningpool-server/dal/do"
	"github.com/abesuite/abe-miningpool-server/errcode"

	"gorm.io/gorm"
)

type MetaInfoDAO interface {
	Create(ctx context.Context, tx *gorm.DB, info *do.MetaInfo) (int64, error)
	Get(ctx context.Context, tx *gorm.DB) (*do.MetaInfo, error)
	Update(ctx context.Context, tx *gorm.DB, info *do.MetaInfo) (int64, error)
	UpdateBalance(ctx context.Context, tx *gorm.DB, balance int64) (int64, error)
	AddBalance(ctx context.Context, tx *gorm.DB, balance int64) (int64, error)
	AddAllocatedBalance(ctx context.Context, tx *gorm.DB, balance int64) (int64, error)
	UpdateLastRewardHeight(ctx context.Context, tx *gorm.DB, height int64) (int64, error)
	UpdateLastTxCheckHeight(ctx context.Context, tx *gorm.DB, height int64) (int64, error)
	UpdateLastRedeemedHeight(ctx context.Context, tx *gorm.DB, height int64) (int64, error)
}

type MetaInfoDAOImpl struct{}

var metaInfoDAO MetaInfoDAO = &MetaInfoDAOImpl{}

func GetMetaInfoDAOImpl() MetaInfoDAO {
	return metaInfoDAO
}

func (m *MetaInfoDAOImpl) Create(ctx context.Context, tx *gorm.DB, info *do.MetaInfo) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	if info == nil {
		return 0, errors.New("fail to create meta info: nil meta info")
	}

	info.ID = 1

	query := tx.Create(info)
	if query.Error != nil {
		return 0, query.Error
	}
	return query.RowsAffected, nil
}

func (m *MetaInfoDAOImpl) Get(ctx context.Context, tx *gorm.DB) (*do.MetaInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	metaInfo := do.MetaInfo{}
	query := tx.Model(&do.MetaInfo{}).Where("id = ?", 1).First(&metaInfo)
	if errors.Is(query.Error, gorm.ErrRecordNotFound) {
		metaInfo.ID = 1
		metaInfo.LastRewardTime = time.Now()
		_, err := m.Create(ctx, tx, &metaInfo)
		if err != nil {
			return nil, err
		}
	} else if query.Error != nil {
		return nil, query.Error
	}

	return &metaInfo, nil
}

func (m *MetaInfoDAOImpl) Update(ctx context.Context, tx *gorm.DB, info *do.MetaInfo) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	if info == nil {
		return 0, errors.New("fail to update meta info: nil meta info")
	}
	info.ID = 1
	query := tx.Save(info)
	if query.Error != nil {
		return 0, errors.New(fmt.Sprintf("fail to update meta info: %v", query.Error))
	}
	return 0, nil
}

func (m *MetaInfoDAOImpl) UpdateBalance(ctx context.Context, tx *gorm.DB, balance int64) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	query := tx.Model(&do.MetaInfo{}).Where("id = ?", 1).Update("balance", balance)
	return query.RowsAffected, query.Error
}

func (m *MetaInfoDAOImpl) AddBalance(ctx context.Context, tx *gorm.DB, balance int64) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	query := tx.Model(&do.MetaInfo{}).Where("id = ?", 1).Update("balance", gorm.Expr("balance + ?", balance))
	return query.RowsAffected, query.Error
}

func (m *MetaInfoDAOImpl) UpdateLastRewardHeight(ctx context.Context, tx *gorm.DB, height int64) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	query := tx.Model(&do.MetaInfo{}).Where("id = ?", 1).Update("last_reward_height", height)
	return query.RowsAffected, query.Error
}

func (m *MetaInfoDAOImpl) UpdateLastTxCheckHeight(ctx context.Context, tx *gorm.DB, height int64) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	query := tx.Model(&do.MetaInfo{}).Where("id = ?", 1).Update("last_tx_check_height", height)
	return query.RowsAffected, query.Error
}

func (m *MetaInfoDAOImpl) UpdateLastRedeemedHeight(ctx context.Context, tx *gorm.DB, height int64) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	query := tx.Model(&do.MetaInfo{}).Where("id = ?", 1).Update("last_redeemed_height", height)
	return query.RowsAffected, query.Error
}

func (m *MetaInfoDAOImpl) AddAllocatedBalance(ctx context.Context, tx *gorm.DB, balance int64) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	query := tx.Model(&do.MetaInfo{}).Where("id = ?", 1).Update("allocated_balance", gorm.Expr("allocated_balance + ?", balance))
	return query.RowsAffected, query.Error
}

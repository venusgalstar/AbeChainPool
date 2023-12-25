package dao

import (
	"context"
	"errors"
	"gorm.io/gorm"

	"github.com/abesuite/abe-miningpool-server/dal/do"
	"github.com/abesuite/abe-miningpool-server/errcode"
)

type AllocationInfoDAO interface {
	Create(ctx context.Context, tx *gorm.DB, info *do.AllocationInfo) (int64, error)
	MCreate(ctx context.Context, tx *gorm.DB, infos []*do.AllocationInfo) (int64, error)
	GetAll(ctx context.Context, tx *gorm.DB) ([]*do.AllocationInfo, error)
	Get(ctx context.Context, tx *gorm.DB, page int, num int, positiveOrder bool) ([]*do.AllocationInfo, error)
	GetNum(ctx context.Context, tx *gorm.DB) (int64, error)
	GetByUsername(ctx context.Context, tx *gorm.DB, username string, page int, num int, positiveOrder bool) ([]*do.AllocationInfo, error)
	GetNumByUsername(ctx context.Context, tx *gorm.DB, username string) (int64, error)
}

type AllocationInfoDAOImpl struct{}

var allocationInfoDAO AllocationInfoDAO = &AllocationInfoDAOImpl{}

func GetAllocationInfoDAOImpl() AllocationInfoDAO {
	return allocationInfoDAO
}

func (a *AllocationInfoDAOImpl) Create(ctx context.Context, tx *gorm.DB, info *do.AllocationInfo) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	if info == nil {
		return 0, errors.New("fail to create allocation info: nil allocation info")
	}

	query := tx.Create(info)
	return query.RowsAffected, query.Error
}

func (a *AllocationInfoDAOImpl) MCreate(ctx context.Context, tx *gorm.DB, infos []*do.AllocationInfo) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	if infos == nil {
		return 0, errors.New("fail to multi create allocation info: nil allocation infos")
	}

	if len(infos) == 0 {
		return 0, nil
	}

	query := tx.CreateInBatches(infos, len(infos))
	return query.RowsAffected, query.Error
}

func (a *AllocationInfoDAOImpl) GetAll(ctx context.Context, tx *gorm.DB) ([]*do.AllocationInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	var infos []*do.AllocationInfo
	query := tx.Model(&do.AllocationInfo{}).Find(&infos)
	if query.Error != nil {
		return nil, query.Error
	}
	return infos, nil
}

func (a *AllocationInfoDAOImpl) GetNum(ctx context.Context, tx *gorm.DB) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	var res int64
	query := tx.Model(&do.AllocationInfo{}).Count(&res)
	return res, query.Error
}

func (a *AllocationInfoDAOImpl) GetNumByUsername(ctx context.Context, tx *gorm.DB, username string) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	var res int64
	query := tx.Model(&do.AllocationInfo{}).Where("username = ?", username).Count(&res)
	return res, query.Error
}

func (a *AllocationInfoDAOImpl) Get(ctx context.Context, tx *gorm.DB, page int, num int, positiveOrder bool) ([]*do.AllocationInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.AllocationInfo, 0)
	if page < 1 || num < 1 {
		return res, nil
	}
	var query *gorm.DB
	if positiveOrder {
		query = tx.Model(&do.AllocationInfo{}).Offset((page - 1) * num).Limit(num).Find(&res)
	} else {
		query = tx.Model(&do.AllocationInfo{}).Order("id desc").Offset((page - 1) * num).Limit(num).Find(&res)
	}

	if query.Error != nil {
		return nil, query.Error
	}
	return res, nil
}

func (a *AllocationInfoDAOImpl) GetByUsername(ctx context.Context, tx *gorm.DB, username string, page int, num int, positiveOrder bool) ([]*do.AllocationInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.AllocationInfo, 0)
	if page < 1 || num < 1 {
		return res, nil
	}
	var query *gorm.DB
	if positiveOrder {
		query = tx.Model(&do.AllocationInfo{}).Where("username = ?", username).Offset((page - 1) * num).Limit(num).Find(&res)
	} else {
		query = tx.Model(&do.AllocationInfo{}).Where("username = ?", username).Order("id desc").Offset((page - 1) * num).Limit(num).Find(&res)
	}

	if query.Error != nil {
		return nil, query.Error
	}
	return res, nil
}

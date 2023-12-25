package dao

import (
	"context"
	"errors"

	"github.com/abesuite/abe-miningpool-server/dal/do"
	"github.com/abesuite/abe-miningpool-server/errcode"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserShareInfoDAO interface {
	Create(ctx context.Context, tx *gorm.DB, info *do.UserShareInfo) error
	Update(ctx context.Context, tx *gorm.DB, info *do.UserShareInfo) error
	Get(ctx context.Context, tx *gorm.DB, page int, num int, positiveOrder bool) ([]*do.UserShareInfo, error)
	GetNum(ctx context.Context, tx *gorm.DB) (int64, error)
	AddShareByID(ctx context.Context, tx *gorm.DB, id uint64, shareCount int64, exclusive ...bool) error
	GetByUsername(ctx context.Context, tx *gorm.DB, username string, page int, num int, positiveOrder bool) ([]*do.UserShareInfo, error)
	GetNumByUsername(ctx context.Context, tx *gorm.DB, username string) (int64, error)
	GetByUsernameAndHeight(ctx context.Context, tx *gorm.DB, username string, startHeight int64, endHeight int64, exclusive ...bool) (*do.UserShareInfo, error)
	GetUnallocatedBeforeHeight(ctx context.Context, tx *gorm.DB, height int64) ([]*do.UserShareInfo, error)
	GetUnallocatedByHeight(ctx context.Context, tx *gorm.DB, startHeight int64, endHeight int64) ([]*do.UserShareInfo, error)
	SetAllocated(ctx context.Context, tx *gorm.DB, ids []uint64) error
	GetLatest(ctx context.Context, tx *gorm.DB) (*do.UserShareInfo, error)
}

type UserShareInfoDAOImpl struct{}

var userShareInfoDAO UserShareInfoDAO = &UserShareInfoDAOImpl{}

func GetUserShareInfoDAOImpl() UserShareInfoDAO {
	return userShareInfoDAO
}

func (u *UserShareInfoDAOImpl) Create(ctx context.Context, tx *gorm.DB, info *do.UserShareInfo) error {
	if tx == nil {
		return errcode.ErrNilGormDB
	}

	if info == nil {
		return errors.New("nil share info when creating")
	}

	query := tx.Create(info)
	return query.Error
}

func (u *UserShareInfoDAOImpl) Update(ctx context.Context, tx *gorm.DB, info *do.UserShareInfo) error {
	if tx == nil {
		return errcode.ErrNilGormDB
	}

	if info == nil {
		return errors.New("nil share info when creating")
	}

	query := tx.Save(info)
	return query.Error
}

func (u *UserShareInfoDAOImpl) Get(ctx context.Context, tx *gorm.DB, page int, num int, positiveOrder bool) ([]*do.UserShareInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.UserShareInfo, 0)
	if page < 1 || num < 1 {
		return res, nil
	}
	var query *gorm.DB
	if positiveOrder {
		query = tx.Model(&do.UserShareInfo{}).Offset((page - 1) * num).Limit(num).Find(&res)
	} else {
		query = tx.Model(&do.UserShareInfo{}).Order("id desc").Offset((page - 1) * num).Limit(num).Find(&res)
	}

	if query.Error != nil {
		return nil, query.Error
	}
	return res, nil
}

func (u *UserShareInfoDAOImpl) GetNum(ctx context.Context, tx *gorm.DB) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	var res int64
	query := tx.Model(&do.UserShareInfo{}).Count(&res)
	return res, query.Error
}

func (u *UserShareInfoDAOImpl) AddShareByID(ctx context.Context, tx *gorm.DB, id uint64, shareCount int64, exclusive ...bool) error {
	if tx == nil {
		return errcode.ErrNilGormDB
	}

	if len(exclusive) > 0 && exclusive[0] {
		tx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	query := tx.Model(&do.UserShareInfo{}).Where("id = ?", id).Update("share_count", gorm.Expr("share_count + ?", shareCount))
	return query.Error
}

func (u *UserShareInfoDAOImpl) GetByUsername(ctx context.Context, tx *gorm.DB, username string, page int, num int, positiveOrder bool) ([]*do.UserShareInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.UserShareInfo, 0)
	if page < 1 || num < 1 {
		return res, nil
	}
	var query *gorm.DB
	if positiveOrder {
		query = tx.Model(&do.UserShareInfo{}).Where("username = ?", username).Offset((page - 1) * num).Limit(num).Find(&res)
	} else {
		query = tx.Model(&do.UserShareInfo{}).Where("username = ?", username).Order("id desc").Offset((page - 1) * num).Limit(num).Find(&res)
	}

	if query.Error != nil {
		return nil, query.Error
	}
	return res, nil
}

func (u *UserShareInfoDAOImpl) GetNumByUsername(ctx context.Context, tx *gorm.DB, username string) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	var res int64
	query := tx.Model(&do.UserShareInfo{}).Where("username = ?", username).Count(&res)
	return res, query.Error
}

func (u *UserShareInfoDAOImpl) GetByUsernameAndHeight(ctx context.Context, tx *gorm.DB, username string, startHeight int64, endHeight int64, exclusive ...bool) (*do.UserShareInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	if len(exclusive) > 0 && exclusive[0] {
		tx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	res := do.UserShareInfo{}
	query := tx.Model(&do.UserShareInfo{}).Where("username = ?", username).
		Where("start_height = ?", startHeight).
		Where("end_height = ?", endHeight).
		Take(&res)
	if query.Error != nil {
		return nil, query.Error
	}
	return &res, nil
}

func (u *UserShareInfoDAOImpl) GetUnallocatedBeforeHeight(ctx context.Context, tx *gorm.DB, height int64) ([]*do.UserShareInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.UserShareInfo, 0)
	query := tx.Model(&do.UserShareInfo{}).Where("allocated = ?", 0).Where("end_height <= ?", height).Find(&res)
	return res, query.Error
}

func (u *UserShareInfoDAOImpl) GetUnallocatedByHeight(ctx context.Context, tx *gorm.DB, startHeight int64, endHeight int64) ([]*do.UserShareInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.UserShareInfo, 0)
	query := tx.Model(&do.UserShareInfo{}).Where("start_height = ?", startHeight).
		Where("end_height = ?", endHeight).Where("allocated = ?", 0).Find(&res)
	return res, query.Error
}

func (u *UserShareInfoDAOImpl) SetAllocated(ctx context.Context, tx *gorm.DB, ids []uint64) error {
	if tx == nil {
		return errcode.ErrNilGormDB
	}

	query := tx.Model(&do.UserShareInfo{}).Where("id in ?", ids).Update("allocated", 1)
	return query.Error
}

func (u *UserShareInfoDAOImpl) GetLatest(ctx context.Context, tx *gorm.DB) (*do.UserShareInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := do.UserShareInfo{}
	err := tx.Last(&res).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &res, nil
}

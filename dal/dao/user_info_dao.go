package dao

import (
	"context"
	"errors"

	"github.com/abesuite/abe-miningpool-server/dal/do"
	"github.com/abesuite/abe-miningpool-server/errcode"

	"gorm.io/gorm"
)

type UserInfoDAO interface {
	Create(ctx context.Context, tx *gorm.DB, info *do.UserInfo) (*do.UserInfo, error)
	UpdateByUsername(ctx context.Context, tx *gorm.DB, info *do.UserInfo) (int64, error)
	GetByUsername(ctx context.Context, tx *gorm.DB, username string) (*do.UserInfo, error)
	GetUserNum(ctx context.Context, tx *gorm.DB) (int64, error)
	GetByID(ctx context.Context, tx *gorm.DB, id uint64) (*do.UserInfo, error)
	GetUsers(ctx context.Context, tx *gorm.DB, page int, num int, withAddress bool, positiveOrder bool) ([]*do.UserInfo, error)
	GetAll(ctx context.Context, tx *gorm.DB) ([]*do.UserInfo, error)
	GetNeedAllocate(ctx context.Context, tx *gorm.DB) ([]*do.UserInfo, error)
	GetIDByUsername(ctx context.Context, tx *gorm.DB, username string) (uint64, error)
	ExistUsername(ctx context.Context, tx *gorm.DB, username string) (bool, error)
	UpdateBalanceByUsername(ctx context.Context, tx *gorm.DB, username string, balance int64) (int64, error)
	AddBalanceByUsername(ctx context.Context, tx *gorm.DB, username string, balance int64) (int64, error)
	UpdatePasswordByUsername(ctx context.Context, tx *gorm.DB, username string, password string) (int64, error)
	UpdateEmailByUsername(ctx context.Context, tx *gorm.DB, username string, email string) (int64, error)
	UpdateRedeemedBalanceByUsername(ctx context.Context, tx *gorm.DB, username string, balance int64) (int64, error)
	AddRedeemedBalanceByUsername(ctx context.Context, tx *gorm.DB, username string, balance int64) (int64, error)
	AddRedeemedBalanceByID(ctx context.Context, tx *gorm.DB, id uint64, balance int64) (int64, error)
}

type UserInfoDAOImpl struct{}

var userInfoDAO UserInfoDAO = &UserInfoDAOImpl{}

func GetUserInfoDAOImpl() UserInfoDAO {
	return userInfoDAO
}

func (u *UserInfoDAOImpl) Create(ctx context.Context, tx *gorm.DB, info *do.UserInfo) (*do.UserInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	if info == nil {
		return nil, errors.New("nil user info when creating")
	}

	query := tx.Create(info)
	return info, query.Error
}

func (u *UserInfoDAOImpl) GetByUsername(ctx context.Context, tx *gorm.DB, username string) (*do.UserInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := do.UserInfo{}
	query := tx.Model(&do.UserInfo{}).Where("username = ?", username).Take(&res)
	return &res, query.Error
}

func (u *UserInfoDAOImpl) GetByID(ctx context.Context, tx *gorm.DB, id uint64) (*do.UserInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := do.UserInfo{}
	query := tx.Model(&do.UserInfo{}).Where("id = ?", id).Take(&res)
	return &res, query.Error
}

func (u *UserInfoDAOImpl) GetUsers(ctx context.Context, tx *gorm.DB, page int, num int, withAddress bool, positiveOrder bool) ([]*do.UserInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.UserInfo, 0)
	if page <= 0 || num <= 0 {
		return res, nil
	}
	var query *gorm.DB
	if withAddress {
		query = tx.Model(&do.UserInfo{}).Offset((page - 1) * num).Limit(num)
	} else {
		query = tx.Model(&do.UserInfo{}).Select("id", "username", "password", "salt", "balance", "redeemed_balance", "created_at", "updated_at").
			Offset((page - 1) * num).Limit(num)
	}

	if !positiveOrder {
		query = query.Order("id desc")
	}
	query = query.Find(&res)

	return res, query.Error
}

func (u *UserInfoDAOImpl) GetAll(ctx context.Context, tx *gorm.DB) ([]*do.UserInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	userInfos := make([]*do.UserInfo, 0)
	query := tx.Find(&userInfos)
	return userInfos, query.Error
}
func (u *UserInfoDAOImpl) GetNeedAllocate(ctx context.Context, tx *gorm.DB) ([]*do.UserInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	userInfos := make([]*do.UserInfo, 0)
	query := tx.Model(&do.UserInfo{}).Where("balance > redeemed_balance").Find(&userInfos)
	return userInfos, query.Error
}

func (u *UserInfoDAOImpl) ExistUsername(ctx context.Context, tx *gorm.DB, username string) (bool, error) {
	if tx == nil {
		return false, errcode.ErrNilGormDB
	}

	var count int64
	query := tx.Model(&do.UserInfo{}).Where("username = ?", username).Count(&count)
	if query.Error != nil {
		return false, query.Error
	}
	if count > 0 {
		return true, nil
	}
	return false, nil
}

func (u *UserInfoDAOImpl) UpdateBalanceByUsername(ctx context.Context, tx *gorm.DB, username string, balance int64) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	query := tx.Model(&do.UserInfo{}).Where("username = ?", username).Update("balance", balance)
	return query.RowsAffected, query.Error
}

func (u *UserInfoDAOImpl) AddBalanceByUsername(ctx context.Context, tx *gorm.DB, username string, balance int64) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	query := tx.Model(&do.UserInfo{}).Where("username = ?", username).Update("balance", gorm.Expr("balance + ?", balance))
	return query.RowsAffected, query.Error
}

func (u *UserInfoDAOImpl) UpdatePasswordByUsername(ctx context.Context, tx *gorm.DB, username string, password string) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	query := tx.Model(&do.UserInfo{}).Where("username = ?", username).Update("password", password)
	return query.RowsAffected, query.Error
}

func (u *UserInfoDAOImpl) UpdateEmailByUsername(ctx context.Context, tx *gorm.DB, username string, email string) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	query := tx.Model(&do.UserInfo{}).Where("username = ?", username).Update("email", email)
	return query.RowsAffected, query.Error
}

func (u *UserInfoDAOImpl) UpdateRedeemedBalanceByUsername(ctx context.Context, tx *gorm.DB, username string, balance int64) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	query := tx.Model(&do.UserInfo{}).Where("username = ?", username).Update("redeemed_balance", balance)
	return query.RowsAffected, query.Error
}

func (u *UserInfoDAOImpl) AddRedeemedBalanceByUsername(ctx context.Context, tx *gorm.DB, username string, balance int64) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	query := tx.Model(&do.UserInfo{}).Where("username = ?", username).Update("redeemed_balance", gorm.Expr("redeemed_balance + ?", balance))
	return query.RowsAffected, query.Error
}

func (u *UserInfoDAOImpl) UpdateByUsername(ctx context.Context, tx *gorm.DB, info *do.UserInfo) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	query := tx.Model(&do.UserInfo{}).Where("username = ?", info.Username).Updates(info)
	return query.RowsAffected, query.Error
}

func (u *UserInfoDAOImpl) GetUserNum(ctx context.Context, tx *gorm.DB) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	var res int64
	query := tx.Model(&do.UserInfo{}).Count(&res)
	return res, query.Error
}

func (u *UserInfoDAOImpl) AddRedeemedBalanceByID(ctx context.Context, tx *gorm.DB, id uint64, balance int64) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	query := tx.Model(&do.UserInfo{}).Where("id = ?", id).Update("redeemed_balance", gorm.Expr("redeemed_balance + ?", balance))
	return query.RowsAffected, query.Error
}

func (u *UserInfoDAOImpl) GetIDByUsername(ctx context.Context, tx *gorm.DB, username string) (uint64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	res := do.UserInfo{}
	query := tx.Model(&do.UserInfo{}).Select("id").Where("username = ?", username).Take(&res)
	return res.ID, query.Error
}

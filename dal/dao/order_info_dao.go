package dao

import (
	"context"
	"errors"

	"github.com/abesuite/abe-miningpool-server/dal/do"
	"github.com/abesuite/abe-miningpool-server/errcode"

	"gorm.io/gorm"
)

type OrderInfoDAO interface {
	Create(ctx context.Context, tx *gorm.DB, info *do.OrderInfo) (*do.OrderInfo, error)
	Get(ctx context.Context, tx *gorm.DB, page int, num int, positiveOrder bool) ([]*do.OrderInfo, error)
	GetByID(ctx context.Context, tx *gorm.DB, id uint64) (*do.OrderInfo, error)
	GetByIDs(ctx context.Context, tx *gorm.DB, ids []uint64) ([]*do.OrderInfo, error)
	GetByUserID(ctx context.Context, tx *gorm.DB, userID uint64, page int, num int, positiveOrder bool) ([]*do.OrderInfo, error)
	GetAllUnfinished(ctx context.Context, tx *gorm.DB) ([]*do.OrderInfo, error)
	GetUnfinished(ctx context.Context, tx *gorm.DB) ([]*do.OrderInfo, error)
	GetUnallocatedOrdersByAmount(ctx context.Context, tx *gorm.DB, amount uint64) ([]*do.OrderInfo, error)
	GetUnfinishedOrdersByAmount(ctx context.Context, tx *gorm.DB, amount int64) ([]*do.OrderInfo, error)
	GetOrderNum(ctx context.Context, tx *gorm.DB) (int64, error)
	GetOrderNumByUserID(ctx context.Context, tx *gorm.DB, userID uint64) (int64, error)
	UpdateStatusByID(ctx context.Context, tx *gorm.DB, id uint64, status int) error
	UpdateUTXOHashByID(ctx context.Context, tx *gorm.DB, id uint64, utxoHash string) error
}

type OrderInfoDAOImpl struct{}

var orderInfoDAO OrderInfoDAO = &OrderInfoDAOImpl{}

func GetOrderInfoDAOImpl() OrderInfoDAO {
	return orderInfoDAO
}

func (o *OrderInfoDAOImpl) Create(ctx context.Context, tx *gorm.DB, info *do.OrderInfo) (*do.OrderInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	if info == nil {
		return nil, errors.New("nil order info when creating")
	}

	query := tx.Create(info)
	return info, query.Error
}

func (o *OrderInfoDAOImpl) Get(ctx context.Context, tx *gorm.DB, page int, num int, positiveOrder bool) ([]*do.OrderInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.OrderInfo, 0)
	if page <= 0 || num <= 0 {
		return res, nil
	}
	query := tx.Model(&do.OrderInfo{}).Offset((page - 1) * num).Limit(num)
	if !positiveOrder {
		query = query.Order("id desc")
	}
	query = query.Find(&res)
	return res, query.Error
}

func (o *OrderInfoDAOImpl) GetByID(ctx context.Context, tx *gorm.DB, id uint64) (*do.OrderInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := do.OrderInfo{}
	query := tx.Model(&do.OrderInfo{}).Where("id = ?", id).Take(&res)
	return &res, query.Error
}

func (o *OrderInfoDAOImpl) GetByIDs(ctx context.Context, tx *gorm.DB, ids []uint64) ([]*do.OrderInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	var res []*do.OrderInfo
	query := tx.Model(&do.OrderInfo{}).Where("id IN ?", ids).Find(&res)
	return res, query.Error
}

func (o *OrderInfoDAOImpl) GetByUserID(ctx context.Context, tx *gorm.DB, userID uint64, page int, num int, positiveOrder bool) ([]*do.OrderInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.OrderInfo, 0)
	query := tx.Model(&do.OrderInfo{}).Where("user_id = ?", userID).Offset((page - 1) * num).Limit(num)
	if !positiveOrder {
		query = query.Order("id desc")
	}
	query = query.Find(&res)
	return res, nil
}

func (o *OrderInfoDAOImpl) GetAllUnfinished(ctx context.Context, tx *gorm.DB) ([]*do.OrderInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.OrderInfo, 0)
	query := tx.Model(&do.OrderInfo{}).Where(" status = ? ", 0).Order("amount DESC, created_at DESC").Scan(&res)
	return res, query.Error
}

func (o *OrderInfoDAOImpl) GetUnfinished(ctx context.Context, tx *gorm.DB) ([]*do.OrderInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.OrderInfo, 0)
	query := tx.Model(&do.OrderInfo{}).Where(" status = ? AND utxo_hash !='' ", 0).Order("created_at DESC").Scan(&res)
	return res, query.Error
}

func (o *OrderInfoDAOImpl) GetUnallocatedOrdersByAmount(ctx context.Context, tx *gorm.DB, amount uint64) ([]*do.OrderInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.OrderInfo, 0)
	query := tx.Model(&do.OrderInfo{}).Where(" status=? AND amount = ? AND utxo_hash = '' ", 0, amount).Order("created_at DESC").Scan(&res)
	return res, query.Error
}

func (o *OrderInfoDAOImpl) GetUnfinishedOrdersByAmount(ctx context.Context, tx *gorm.DB, amount int64) ([]*do.OrderInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.OrderInfo, 0)
	query := tx.Model(&do.OrderInfo{}).Where(" amount = ? AND status = ? AND utxo_hash != '' ", amount, 0).Order("id ASC,created_at ASC").Scan(&res)
	return res, query.Error
}

func (o *OrderInfoDAOImpl) UpdateStatusByID(ctx context.Context, tx *gorm.DB, id uint64, status int) error {
	if tx == nil {
		return errcode.ErrNilGormDB
	}

	query := tx.Model(&do.OrderInfo{}).Where("id = ?", id).Update("status", status)
	return query.Error
}

func (o *OrderInfoDAOImpl) UpdateUTXOHashByID(ctx context.Context, tx *gorm.DB, id uint64, utxoHash string) error {
	if tx == nil {
		return errcode.ErrNilGormDB
	}

	query := tx.Model(&do.OrderInfo{}).Where("id = ?", id).Update("utxo_hash", utxoHash)
	return query.Error
}

func (o *OrderInfoDAOImpl) GetOrderNum(ctx context.Context, tx *gorm.DB) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	var res int64
	query := tx.Model(&do.OrderInfo{}).Count(&res)
	return res, query.Error
}

func (o *OrderInfoDAOImpl) GetOrderNumByUserID(ctx context.Context, tx *gorm.DB, userID uint64) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	var res int64
	query := tx.Model(&do.OrderInfo{}).Where("user_id = ?", userID).Count(&res)
	return res, query.Error
}

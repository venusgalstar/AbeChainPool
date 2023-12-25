package dao

import (
	"context"
	"errors"

	"github.com/abesuite/abe-miningpool-server/dal/do"
	"github.com/abesuite/abe-miningpool-server/errcode"

	"gorm.io/gorm"
)

type OrderTransactionInfoDAO interface {
	Create(ctx context.Context, tx *gorm.DB, info *do.OrderTransactionInfo) (*do.OrderTransactionInfo, error)
	GetTransactionNum(ctx context.Context, tx *gorm.DB) (int64, error)
	Get(ctx context.Context, tx *gorm.DB, page int, num int, withAddress bool, positiveOrder bool) ([]*do.OrderTransactionInfo, error)
	GetByID(ctx context.Context, tx *gorm.DB, id uint64) (*do.OrderTransactionInfo, error)
	GetByOrderID(ctx context.Context, tx *gorm.DB, orderID uint64, withAddress bool) ([]*do.OrderTransactionInfo, error)
	UpdateStatusByTxHash(ctx context.Context, tx *gorm.DB, hash string, height int, status int) error
	UpdateByIDs(ctx context.Context, tx *gorm.DB, id []uint64, params map[string]interface{}) error
	GetTransactionRange(ctx context.Context, tx *gorm.DB, lowHeight int64, highHeight int64) ([]*do.OrderTransactionInfo, error)
	GetPendingTransactions(ctx context.Context, tx *gorm.DB) ([]*do.OrderTransactionInfo, error)
	GetTransactionRequestHashPending(ctx context.Context, tx *gorm.DB) ([]string, error)
	GetOrderTransactionPendingWithRequestHash(ctx context.Context, tx *gorm.DB, requestHash string) ([]*do.OrderTransactionInfo, error)
	GetTransactionPending(ctx context.Context, tx *gorm.DB) ([]*do.OrderTransactionInfo, error)
	GetUnfinishedOrderTransactions(ctx context.Context, tx *gorm.DB) ([]*do.OrderTransactionInfo, error)
}

type OrderTransactionInfoDAOImpl struct{}

var orderTransactionInfoDAO OrderTransactionInfoDAO = &OrderTransactionInfoDAOImpl{}

func GetOrderTransactionInfoDAOImpl() OrderTransactionInfoDAO {
	return orderTransactionInfoDAO
}

func (o *OrderTransactionInfoDAOImpl) Create(ctx context.Context, tx *gorm.DB, info *do.OrderTransactionInfo) (*do.OrderTransactionInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	if info == nil {
		return nil, errors.New("nil order transaction info when creating")
	}

	query := tx.Create(info)
	return info, query.Error
}

func (o *OrderTransactionInfoDAOImpl) Get(ctx context.Context, tx *gorm.DB, page int, num int, withAddress bool, positiveOrder bool) ([]*do.OrderTransactionInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.OrderTransactionInfo, 0)
	if page <= 0 || num <= 0 {
		return res, nil
	}
	var query *gorm.DB
	if withAddress {
		query = tx.Model(&do.OrderTransactionInfo{}).Offset((page - 1) * num).Limit(num)
	} else {
		query = tx.Model(&do.OrderTransactionInfo{}).Select("id", "order_id", "transaction_hash", "amount", "status", "height", "created_at", "updated_at").
			Offset((page - 1) * num).Limit(num)
	}
	if !positiveOrder {
		query = query.Order("id desc")
	}
	query = query.Find(&res)
	return res, query.Error
}

func (o *OrderTransactionInfoDAOImpl) GetByID(ctx context.Context, tx *gorm.DB, id uint64) (*do.OrderTransactionInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := do.OrderTransactionInfo{}
	query := tx.Model(&do.OrderTransactionInfo{}).Where("id = ?", id).Take(&res)
	return &res, query.Error
}

func (o *OrderTransactionInfoDAOImpl) GetByOrderID(ctx context.Context, tx *gorm.DB, orderID uint64, withAddress bool) ([]*do.OrderTransactionInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.OrderTransactionInfo, 0)
	var query *gorm.DB
	if withAddress {
		query = tx.Model(&do.OrderTransactionInfo{}).Where("order_id = ?", orderID).Find(&res)
	} else {
		query = tx.Model(&do.OrderTransactionInfo{}).Select("id", "order_id", "transaction_hash", "amount", "status", "height", "created_at", "updated_at").
			Where("order_id = ?", orderID).Find(&res)
	}
	return res, query.Error
}

func (o *OrderTransactionInfoDAOImpl) GetTransactionRange(ctx context.Context, tx *gorm.DB, lowHeight int64, highHeight int64) ([]*do.OrderTransactionInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.OrderTransactionInfo, 0)

	query := tx.Debug().Model(&do.OrderTransactionInfo{}).Where(" status IN ? AND (transaction_request_hash != '' OR transaction_hash != '') AND last_checked_height BETWEEN ? AND ?",
		[]int{do.TransactionPending, do.TransactionWaiting, do.TransactionInChain, do.TransactionInvalid},
		lowHeight,
		highHeight).Find(&res)

	return res, query.Error
}

func (o *OrderTransactionInfoDAOImpl) GetPendingTransactions(ctx context.Context, tx *gorm.DB) ([]*do.OrderTransactionInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.OrderTransactionInfo, 0)

	query := tx.Debug().Model(&do.OrderTransactionInfo{}).Where(" status = ? AND transaction_request_hash != '' AND transaction_hash = ''",
		do.TransactionPending).Find(&res)

	return res, query.Error
}

func (o *OrderTransactionInfoDAOImpl) GetTransactionRequestHashPending(ctx context.Context, tx *gorm.DB) ([]string, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]string, 0)

	query := tx.Debug().Model(&do.OrderTransactionInfo{}).Distinct("transaction_request_hash").Where(" status = ? AND transaction_request_hash !='' ", do.TransactionPending).Pluck("transaction_request_hash", &res)

	return res, query.Error
}

func (o *OrderTransactionInfoDAOImpl) GetOrderTransactionPendingWithRequestHash(ctx context.Context, tx *gorm.DB, requestHash string) ([]*do.OrderTransactionInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.OrderTransactionInfo, 0)

	query := tx.Debug().Model(&do.OrderTransactionInfo{}).Preload("OrderInfo").Where(" status = ? AND transaction_request_hash = ? ", do.TransactionPending, requestHash).Order("id ASC").Find(&res)

	return res, query.Error
}

func (o *OrderTransactionInfoDAOImpl) GetTransactionPending(ctx context.Context, tx *gorm.DB) ([]*do.OrderTransactionInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.OrderTransactionInfo, 0)

	query := tx.Debug().Model(&do.OrderTransactionInfo{}).Preload("OrderInfo").Where(" status = ? AND transaction_request_hash !='' ", do.TransactionPending).Order("id ASC").Find(&res)

	return res, query.Error
}
func (o *OrderTransactionInfoDAOImpl) GetUnfinishedOrderTransactions(ctx context.Context, tx *gorm.DB) ([]*do.OrderTransactionInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.OrderTransactionInfo, 0)

	query := tx.Debug().Model(&do.OrderTransactionInfo{}).Preload("OrderInfo").Where(" status IN ? ", []int{do.TransactionPending, do.TransactionManual}).Find(&res)

	return res, query.Error
}

func (o *OrderTransactionInfoDAOImpl) UpdateStatusByTxHash(ctx context.Context, tx *gorm.DB, hash string, height int, status int) error {
	if tx == nil {
		return errcode.ErrNilGormDB
	}

	query := tx.Model(&do.OrderTransactionInfo{}).Where("transaction_hash = ?", hash).Updates(map[string]interface{}{
		"status":              status,
		"height":              height,
		"last_checked_height": height,
	})
	return query.Error
}

func (o *OrderTransactionInfoDAOImpl) UpdateByIDs(ctx context.Context, tx *gorm.DB, ids []uint64, params map[string]interface{}) error {
	if tx == nil {
		return errcode.ErrNilGormDB
	}

	query := tx.Model(&do.OrderTransactionInfo{}).Where("id IN ?", ids).Updates(params)
	return query.Error
}

func (o *OrderTransactionInfoDAOImpl) GetTransactionNum(ctx context.Context, tx *gorm.DB) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	var res int64
	query := tx.Model(&do.OrderTransactionInfo{}).Count(&res)
	return res, query.Error
}

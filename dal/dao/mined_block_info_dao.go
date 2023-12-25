package dao

import (
	"context"
	"errors"

	"github.com/abesuite/abe-miningpool-server/dal/do"
	"github.com/abesuite/abe-miningpool-server/errcode"
	"github.com/abesuite/abe-miningpool-server/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MinedBlockInfoDAO interface {
	Create(ctx context.Context, tx *gorm.DB, info *do.MinedBlockInfo) (int64, error)
	Get(ctx context.Context, tx *gorm.DB, page int, num int, positiveOrder bool) ([]*do.MinedBlockInfo, error)
	GetBlockNum(ctx context.Context, tx *gorm.DB) (int64, error)
	GetAll(ctx context.Context, tx *gorm.DB) ([]*do.MinedBlockInfo, error)
	ConnectByBlockHash(ctx context.Context, tx *gorm.DB, blockHash string) (int64, error)
	DisconnectByBlockHash(ctx context.Context, tx *gorm.DB, blockHash string) (int64, error)
	RejectedByBlockHash(ctx context.Context, tx *gorm.DB, blockHash string, info string) (int64, error)
	GetConnectedBlocksHigherThanHeight(ctx context.Context, tx *gorm.DB, height int64) ([]*do.MinedBlockInfo, error)
	GetUnallocatedBetweenHeight(ctx context.Context, tx *gorm.DB, startHeight int64, endHeight int64, exclusive ...bool) ([]*do.MinedBlockInfo, error)
	GetAcceptedBlocksBetweenHeight(ctx context.Context, tx *gorm.DB, startHeight int64, endHeight int64, exclusive ...bool) ([]*do.MinedBlockInfo, error)
	SetAllocated(ctx context.Context, tx *gorm.DB, ids []uint64) error
}

type MinedBlockInfoDAOImpl struct{}

var minedBlockInfoDAO MinedBlockInfoDAO = &MinedBlockInfoDAOImpl{}

func GetMinedBlockInfoDAOImpl() MinedBlockInfoDAO {
	return minedBlockInfoDAO
}

func (m *MinedBlockInfoDAOImpl) Create(ctx context.Context, tx *gorm.DB, info *do.MinedBlockInfo) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	if info == nil {
		return 0, errors.New("nil mined block info when creating")
	}

	query := tx.Create(info)
	return query.RowsAffected, query.Error
}

func (m *MinedBlockInfoDAOImpl) Get(ctx context.Context, tx *gorm.DB, page int, num int, positiveOrder bool) ([]*do.MinedBlockInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	res := make([]*do.MinedBlockInfo, 0)
	if page <= 0 || num <= 0 {
		return res, nil
	}
	var query *gorm.DB
	if positiveOrder {
		query = tx.Model(&do.MinedBlockInfo{}).Offset((page - 1) * num).Limit(num).Find(&res)
	} else {
		query = tx.Model(&do.MinedBlockInfo{}).Order("id desc").Offset((page - 1) * num).Limit(num).Find(&res)
	}
	return res, query.Error
}

func (m *MinedBlockInfoDAOImpl) GetBlockNum(ctx context.Context, tx *gorm.DB) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	var res int64
	query := tx.Model(&do.MinedBlockInfo{}).Count(&res)
	return res, query.Error
}

func (m *MinedBlockInfoDAOImpl) ConnectByBlockHash(ctx context.Context, tx *gorm.DB, blockHash string) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	if utils.IsBlank(blockHash) {
		return 0, errors.New("internal error: block blockHash")
	}

	query := tx.Model(&do.MinedBlockInfo{}).Where("block_hash = ?", blockHash).Update("connected", 1)
	if query.Error != nil {
		return 0, query.Error
	}
	query = tx.Model(&do.MinedBlockInfo{}).Where("block_hash = ?", blockHash).Update("disconnected", 0)
	return query.RowsAffected, query.Error
}

func (m *MinedBlockInfoDAOImpl) DisconnectByBlockHash(ctx context.Context, tx *gorm.DB, blockHash string) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	if utils.IsBlank(blockHash) {
		return 0, errors.New("internal error: block blockHash")
	}

	query := tx.Model(&do.MinedBlockInfo{}).Where("block_hash = ?", blockHash).Update("disconnected", 1)
	if query.Error != nil {
		return 0, query.Error
	}
	query = tx.Model(&do.MinedBlockInfo{}).Where("block_hash = ?", blockHash).Update("connected", 0)
	return query.RowsAffected, query.Error
}

func (m *MinedBlockInfoDAOImpl) RejectedByBlockHash(ctx context.Context, tx *gorm.DB, blockHash string, info string) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	if utils.IsBlank(blockHash) {
		return 0, errors.New("internal error: block blockHash")
	}

	query := tx.Model(&do.MinedBlockInfo{}).Where("block_hash = ?", blockHash).Updates(do.MinedBlockInfo{Rejected: 1, Info: info})
	return query.RowsAffected, query.Error
}

func (m *MinedBlockInfoDAOImpl) GetConnectedBlocksHigherThanHeight(ctx context.Context, tx *gorm.DB, height int64) ([]*do.MinedBlockInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	var res []*do.MinedBlockInfo
	query := tx.Model(&do.MinedBlockInfo{}).Where("height > ?", height).Where("connected = ?", 1).Where("disconnected = ?", 0).Find(&res)
	return res, query.Error
}

func (m *MinedBlockInfoDAOImpl) GetAll(ctx context.Context, tx *gorm.DB) ([]*do.MinedBlockInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	var infos []*do.MinedBlockInfo
	query := tx.Model(&do.MinedBlockInfo{}).Find(&infos)
	if query.Error != nil {
		return nil, query.Error
	}
	return infos, nil
}

// GetUnallocatedBetweenHeight returns the connected and unallocated blocks whose height is in [startHeight, endHeight).
func (m *MinedBlockInfoDAOImpl) GetUnallocatedBetweenHeight(ctx context.Context, tx *gorm.DB, startHeight int64, endHeight int64, exclusive ...bool) ([]*do.MinedBlockInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	if len(exclusive) > 0 && exclusive[0] {
		tx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	var res []*do.MinedBlockInfo
	query := tx.Model(&do.MinedBlockInfo{}).Where("height >= ?", startHeight).
		Where("height < ?", endHeight).
		Where("connected = ?", 1).
		Where("allocated = ?", 0).
		Find(&res)
	return res, query.Error
}

func (m *MinedBlockInfoDAOImpl) GetAcceptedBlocksBetweenHeight(ctx context.Context, tx *gorm.DB, startHeight int64, endHeight int64, exclusive ...bool) ([]*do.MinedBlockInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	if len(exclusive) > 0 && exclusive[0] {
		tx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	var res []*do.MinedBlockInfo
	query := tx.Model(&do.MinedBlockInfo{}).Where("height >= ?", startHeight).
		Where("height < ?", endHeight).
		Where("rejected = ?", 0).
		Where("allocated = ?", 0).
		Find(&res)
	return res, query.Error
}

func (m *MinedBlockInfoDAOImpl) SetAllocated(ctx context.Context, tx *gorm.DB, ids []uint64) error {
	if tx == nil {
		return errcode.ErrNilGormDB
	}

	query := tx.Model(&do.MinedBlockInfo{}).Where("id in ?", ids).Update("allocated", 1)
	return query.Error
}

package dao

import (
	"context"
	"errors"
	"gorm.io/gorm"

	"github.com/abesuite/abe-miningpool-server/dal/do"
	"github.com/abesuite/abe-miningpool-server/errcode"
)

type DetailedShareInfoDAO interface {
	Create(ctx context.Context, tx *gorm.DB, info *do.DetailedShareInfo) (int64, error)
	GetAll(ctx context.Context, tx *gorm.DB) ([]*do.DetailedShareInfo, error)
}

type DetailedShareInfoDAOImpl struct{}

var detailedShareInfoDAO DetailedShareInfoDAO = &DetailedShareInfoDAOImpl{}

func GetDetailedShareInfoDAOImpl() DetailedShareInfoDAO {
	return detailedShareInfoDAO
}

func (s *DetailedShareInfoDAOImpl) Create(ctx context.Context, tx *gorm.DB, info *do.DetailedShareInfo) (int64, error) {
	if tx == nil {
		return 0, errcode.ErrNilGormDB
	}

	if info == nil {
		return 0, errors.New("nil share info when creating")
	}

	query := tx.Create(info)
	return query.RowsAffected, query.Error
}

func (s *DetailedShareInfoDAOImpl) GetAll(ctx context.Context, tx *gorm.DB) ([]*do.DetailedShareInfo, error) {
	if tx == nil {
		return nil, errcode.ErrNilGormDB
	}

	var infos []*do.DetailedShareInfo
	query := tx.Model(&do.DetailedShareInfo{}).Find(&infos)
	if query.Error != nil {
		return nil, query.Error
	}
	return infos, nil
}

package dao

import (
	"context"
	"fmt"
	"testing"

	"github.com/abesuite/abe-miningpool-server/dal/do"
	"gorm.io/driver/mysql"

	"gorm.io/gorm"
)

func TestMetaInfoDAOImpl_Get(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", "alice", "123456",
		"127.0.0.1:3306", "abe_mining_pool")
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	t.Run("test_1", func(t *testing.T) {
		m := &MetaInfoDAOImpl{}
		res, err := m.Get(context.Background(), db)
		if err != nil {
			t.Error(err.Error())
		}
		fmt.Println(res)
	})
}

func TestMetaInfoDAOImpl_Update(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", "alice", "123456",
		"127.0.0.1:3306", "abe_mining_pool")
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	t.Run("test_1", func(t *testing.T) {
		m := &MetaInfoDAOImpl{}
		metaInfo := do.MetaInfo{}
		db.First(&metaInfo)
		metaInfo.Balance = 200000
		_, err := m.Update(context.Background(), db, &metaInfo)
		if err != nil {
			t.Error(err.Error())
		}
	})
}

func TestMetaInfoDAOImpl_UpdateBalance(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", "alice", "123456",
		"127.0.0.1:3306", "abe_mining_pool")
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	t.Run("test_1", func(t *testing.T) {
		m := &MetaInfoDAOImpl{}
		_, err := m.UpdateBalance(context.Background(), db, 1000000)
		if err != nil {
			t.Error(err.Error())
		}
	})
}

func TestMetaInfoDAOImpl_AddBalance(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", "alice", "123456",
		"127.0.0.1:3306", "abe_mining_pool")
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	t.Run("test_1", func(t *testing.T) {
		m := &MetaInfoDAOImpl{}
		_, err := m.AddBalance(context.Background(), db, 500000)
		if err != nil {
			t.Error(err.Error())
		}
	})
}

func TestMetaInfoDAOImpl_UpdateLastRewardHeight(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", "alice", "123456",
		"127.0.0.1:3306", "abe_mining_pool")
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	t.Run("test_1", func(t *testing.T) {
		m := &MetaInfoDAOImpl{}
		_, err := m.UpdateLastRewardHeight(context.Background(), db, 100)
		if err != nil {
			t.Error(err.Error())
		}
	})
}

package dao

import (
	"context"
	"fmt"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMinedBlockInfoDAOImpl_GetBlocksHigherThanHeight(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", "alice", "123456",
		"127.0.0.1:3306", "abe_mining_pool")
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	t.Run("test_1", func(t *testing.T) {
		m := MinedBlockInfoDAOImpl{}
		infos, err := m.GetConnectedBlocksHigherThanHeight(context.Background(), db, 10)
		if err != nil {
			t.Error(err.Error())
		}
		for _, info := range infos {
			fmt.Println(*info)
		}
	})
}

func TestMinedBlockInfoDAOImpl_GetAll(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", "alice", "123456",
		"127.0.0.1:3306", "abe_mining_pool")
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	t.Run("test_1", func(t *testing.T) {
		m := &MinedBlockInfoDAOImpl{}
		infos, err := m.GetAll(context.Background(), db)
		if err != nil {
			t.Error(err.Error())
		}
		for _, info := range infos {
			fmt.Println(*info)
		}
	})
}

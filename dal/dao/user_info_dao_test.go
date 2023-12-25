package dao

import (
	"context"
	"fmt"
	"testing"

	"github.com/abesuite/abe-miningpool-server/dal/do"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestUserInfoDAOImpl_Create(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", "alice", "123456",
		"127.0.0.1:3306", "abe_mining_pool")
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	t.Run("test_1", func(t *testing.T) {
		email := "23223@qq.com"
		info := &do.UserInfo{
			Username: "admin",
			Password: "1223456",
			Salt:     "232g34f34",
			Email:    &email,
		}
		m := UserInfoDAOImpl{}
		_, err := m.Create(context.Background(), db, info)
		if err != nil {
			t.Error(err.Error())
		}
	})
}

func TestUserInfoDAOImpl_UpdateBalanceByUsername(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", "alice", "123456",
		"127.0.0.1:3306", "abe_mining_pool")
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	t.Run("test_1", func(t *testing.T) {
		m := UserInfoDAOImpl{}
		_, err := m.UpdateBalanceByUsername(context.Background(), db, "admin", 800)
		if err != nil {
			t.Error(err.Error())
		}
	})
}

func TestUserInfoDAOImpl_GetByUsername(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", "alice", "123456",
		"127.0.0.1:3306", "abe_mining_pool")
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	t.Run("test_1", func(t *testing.T) {
		m := UserInfoDAOImpl{}
		info, err := m.GetByUsername(context.Background(), db, "admin")
		if err != nil {
			t.Error(err.Error())
		}
		fmt.Println(info)
	})
}

func TestUserInfoDAOImpl_GetAll(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", "alice", "123456",
		"127.0.0.1:3306", "abe_mining_pool")
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	t.Run("test_1", func(t *testing.T) {
		m := UserInfoDAOImpl{}
		infos, err := m.GetAll(context.Background(), db)
		if err != nil {
			t.Error(err.Error())
		}
		for _, info := range infos {
			fmt.Printf("%+v\n", info)
		}
	})
}

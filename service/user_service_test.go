package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/abesuite/abe-miningpool-server/dal/dao"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestUserServiceImpl_Login(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", "alice", "123456",
		"127.0.0.1:3306", "abe_mining_pool")
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	t.Run("test_1", func(t *testing.T) {
		d := &dao.UserInfoDAOImpl{}
		m := UserServiceImpl{
			userInfoDao: d,
		}
		success, info, err := m.Login(context.Background(), db, "admin", "123456")
		if err != nil {
			t.Error(err.Error())
		}
		fmt.Println(success)
		fmt.Printf("%+v", info)
	})
}

func TestUserServiceImpl_RegisterUser(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", "alice", "123456",
		"127.0.0.1:3306", "abe_mining_pool")
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	t.Run("test_1", func(t *testing.T) {
		d := &dao.UserInfoDAOImpl{}
		m := UserServiceImpl{
			userInfoDao: d,
		}
		info, err := m.RegisterUser(context.Background(), db, "admin", "123456", "1234567890", nil)
		if err != nil {
			t.Error(err.Error())
		}
		fmt.Printf("%+v", info)
	})
}

func TestUserServiceImpl_DisconnectBlock(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", "alice", "123456",
		"127.0.0.1:3306", "abe_mining_pool")
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	t.Run("test_1", func(t *testing.T) {
		d := &dao.UserInfoDAOImpl{}
		s := &dao.DetailedShareInfoDAOImpl{}
		q := &dao.MinedBlockInfoDAOImpl{}
		m := UserServiceImpl{
			userInfoDao:          d,
			detailedShareInfoDao: s,
			minedBlockInfoDao:    q,
		}
		err := m.DisconnectBlock(context.Background(), db, "000000d7b61be9d1f2a2c2a74a5551ea571726be57bf5a1e45c22a06c016f80a")
		if err != nil {
			t.Error(err.Error())
		}
	})
}

func TestUserServiceImpl_AllocateRewards(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", "alice", "123456",
		"127.0.0.1:3306", "abe_mining_pool")
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	t.Run("test_1", func(t *testing.T) {
		d := &dao.UserInfoDAOImpl{}
		s := &dao.DetailedShareInfoDAOImpl{}
		q := &dao.MinedBlockInfoDAOImpl{}
		a := &dao.AllocationInfoDAOImpl{}
		u := &dao.UserShareInfoDAOImpl{}
		m := UserServiceImpl{
			userInfoDao:          d,
			detailedShareInfoDao: s,
			minedBlockInfoDao:    q,
			allocationInfoDAO:    a,
			userShareInfoDao:     u,
		}
		err := m.AllocateRewards(context.Background(), db, 1000, 1100, 20)
		if err != nil {
			t.Error(err.Error())
		}
	})
}

package dal

import (
	"context"
	"fmt"

	"github.com/abesuite/abe-miningpool-server/dal/do"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var GlobalDBClient *gorm.DB

func GetDB(ctx context.Context) *gorm.DB {
	return GlobalDBClient.WithContext(ctx)
}

type DBConfig struct {
	Username string
	Password string
	// Address including the ip address and port of database (e.g. 127.0.0.1:3306)
	Address      string
	DatabaseName string
}

func InitDB(cfg *DBConfig, autoCreate bool) error {
	if autoCreate {
		err := CreateDatabase(cfg)
		if err != nil {
			return err
		}
		err = CreateTables(cfg)
		if err != nil {
			return err
		}
	}

	log.Infof("Connecting to database %v at %v...", cfg.DatabaseName, cfg.Address)

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", cfg.Username, cfg.Password,
		cfg.Address, cfg.DatabaseName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	GlobalDBClient = db

	log.Infof("Successfully connect to database")

	return nil
}

func CreateDatabase(cfg *DBConfig) error {
	log.Infof("Creating database %s...", cfg.DatabaseName)

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/?charset=utf8mb4&parseTime=True&loc=Local", cfg.Username, cfg.Password,
		cfg.Address)
	db, err := gorm.Open(mysql.Open(dsn), nil)
	if err != nil {
		return err
	}

	createSQL := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4;",
		cfg.DatabaseName,
	)

	err = db.Exec(createSQL).Error
	if err != nil {
		log.Infof("Unable to create database %s...", cfg.DatabaseName)
		return err
	}
	return nil
}

func CreateTables(cfg *DBConfig) error {

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", cfg.Username, cfg.Password,
		cfg.Address, cfg.DatabaseName)
	db, err := gorm.Open(mysql.Open(dsn), nil)
	if err != nil {
		return err
	}

	log.Infof("Creating table meta_infos...")
	err = db.AutoMigrate(&do.MetaInfo{})
	if err != nil {
		log.Infof("Fail to create table meta_infos")
		return err
	}

	log.Infof("Creating table user_infos...")
	err = db.AutoMigrate(&do.UserInfo{})
	if err != nil {
		log.Infof("Fail to create table user_infos")
		return err
	}

	log.Infof("Creating table detailed_share_infos...")
	err = db.AutoMigrate(&do.DetailedShareInfo{})
	if err != nil {
		log.Infof("Fail to create table share_infos")
		return err
	}

	log.Infof("Creating table mined_block_infos...")
	err = db.AutoMigrate(&do.MinedBlockInfo{})
	if err != nil {
		log.Infof("Fail to create table mined_block_infos")
		return err
	}

	log.Infof("Creating table allocation_infos...")
	err = db.AutoMigrate(&do.AllocationInfo{})
	if err != nil {
		log.Infof("Fail to create table allocation_infos")
		return err
	}

	log.Infof("Creating table user_share_infos...")
	err = db.AutoMigrate(&do.UserShareInfo{})
	if err != nil {
		log.Infof("Fail to create table user_share_infos")
		return err
	}

	log.Infof("Creating table order_infos...")
	err = db.AutoMigrate(&do.OrderInfo{})
	if err != nil {
		log.Infof("Fail to create table order_infos")
		return err
	}

	log.Infof("Creating table order_transaction_infos...")
	err = db.AutoMigrate(&do.OrderTransactionInfo{})
	if err != nil {
		log.Infof("Fail to create table order_transaction_infos")
		return err
	}

	log.Infof("Creating table config_infos...")
	err = db.AutoMigrate(&do.ConfigInfo{})
	if err != nil {
		log.Infof("Fail to create table config_infos")
		return err
	}
	return nil
}

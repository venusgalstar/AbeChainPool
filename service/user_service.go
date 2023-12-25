package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/abesuite/abe-miningpool-server/abejson"
	"github.com/abesuite/abe-miningpool-server/constdef"
	"github.com/abesuite/abe-miningpool-server/dal/dao"
	"github.com/abesuite/abe-miningpool-server/dal/do"
	"github.com/abesuite/abe-miningpool-server/model"
	"github.com/abesuite/abe-miningpool-server/pooljson"
	"github.com/abesuite/abe-miningpool-server/utils"

	"gorm.io/gorm"
)

type UserService interface {
	GetUserByUsername(ctx context.Context, tx *gorm.DB, username string) (*do.UserInfo, error)
	GetUserByID(ctx context.Context, tx *gorm.DB, id uint64) (*do.UserInfo, error)
	GetUsers(ctx context.Context, tx *gorm.DB, page int, num int, withAddress bool, positiveOrder bool) ([]*do.UserInfo, error)
	UsernameExist(ctx context.Context, tx *gorm.DB, username string) (bool, error)
	GetUserNum(ctx context.Context, tx *gorm.DB) (int64, error)
	RegisterAdmin(ctx context.Context, tx *gorm.DB, password string) error
	RegisterUser(ctx context.Context, tx *gorm.DB, username string, password string, address string, email *string) (*do.UserInfo, error)
	LoginAdmin(ctx context.Context, tx *gorm.DB, password string) (bool, error)
	Login(ctx context.Context, tx *gorm.DB, username string, password string) (bool, *do.UserInfo, error)
	GetShare(ctx context.Context, db *gorm.DB, page int, num int, positiveOrder bool) ([]*do.UserShareInfo, error)
	GetShareCount(ctx context.Context, db *gorm.DB) (int64, error)
	GetShareByUsername(ctx context.Context, db *gorm.DB, username string, page int, num int, positiveOrder bool) ([]*do.UserShareInfo, error)
	GetShareCountByUsername(ctx context.Context, db *gorm.DB, username string) (int64, error)
	AddShareByUsernameOld(ctx context.Context, tx *gorm.DB, shareInfo *model.ShareInfo, rewardInterval int, recordDetail bool) error
	AddShareByUsername(ctx context.Context, tx *gorm.DB, shareInfo *model.ShareInfo, rewardInterval int, recordDetail bool) error
	AddMinedBlock(ctx context.Context, tx *gorm.DB, minedBlockInfo *model.ShareInfo) error
	GetMinedBlocks(ctx context.Context, tx *gorm.DB, page int, num int, positiveOrder bool) ([]*do.MinedBlockInfo, error)
	GetMinedBlockNum(ctx context.Context, tx *gorm.DB) (int64, error)
	ConnectBlock(ctx context.Context, tx *gorm.DB, blockHash string) error
	DisconnectBlock(ctx context.Context, tx *gorm.DB, blockHash string) error
	RejectBlock(ctx context.Context, tx *gorm.DB, blockNotification *model.BlockNotification) error
	GetUnallocatedReward(ctx context.Context, tx *gorm.DB) (int64, error)
	ChangePassword(ctx context.Context, tx *gorm.DB, username string, oldPassword string, newPassword string) error
	ResetPassword(ctx context.Context, tx *gorm.DB, username string, password string) error
	AllocateRewards(ctx context.Context, tx *gorm.DB, startHeight int, endHeight int, manageFeePercent int) error
	AllocateRewardsBeforeHeight(ctx context.Context, tx *gorm.DB, height int, manageFeePercent int) error
	AllocateRewardsWithBlocks(ctx context.Context, tx *gorm.DB, startHeight int, endHeight int, manageFeePercent int, connectedBlocks []*do.MinedBlockInfo) error
	GetUnallocatedByHeight(ctx context.Context, tx *gorm.DB, startHeight int, endHeight int) ([]*do.MinedBlockInfo, error)
	GetAcceptedBlocksByHeight(ctx context.Context, tx *gorm.DB, startHeight int, endHeight int) ([]*do.MinedBlockInfo, error)
	GetUnallocatedRanges(ctx context.Context, tx *gorm.DB, height int) (map[model.HeightPair]struct{}, error)
	CreateOrders(ctx context.Context, tx *gorm.DB, unitCnt int, unit uint64, utxoList []*abejson.UTXO) error
	GetOrders(ctx context.Context, tx *gorm.DB, page int, num int, withAddress bool, positiveOrder bool) ([]*model.OrderDetails, error)
	GetOrdersByIDs(ctx context.Context, tx *gorm.DB, ids []uint64) ([]*do.OrderInfo, error)
	GetOrdersByUsername(ctx context.Context, tx *gorm.DB, username string, page int, num int, positiveOrder bool) ([]*do.OrderInfo, error)
	GetAllUnfinishedOrders(ctx context.Context, tx *gorm.DB) ([]*do.OrderInfo, error)
	GetUnfinishedOrders(ctx context.Context, tx *gorm.DB) ([]*do.OrderInfo, error)
	GetUnfinishedOrderTransactions(ctx context.Context, tx *gorm.DB) ([]*do.OrderTransactionInfo, error)
	GetUnfinishedOrdersByAmountUnit(ctx context.Context, tx *gorm.DB, unit int64) (map[string][]*do.OrderInfo, error)
	GetOrderTransactionInfoRange(ctx context.Context, tx *gorm.DB, lowHeight int64, highHeight int64) ([]*model.OrderTransactionStatus, error)
	GetTransactionRequestHash(ctx context.Context, tx *gorm.DB) ([]string, error)
	GetOrderTransactionByTransactionRequestHash(ctx context.Context, tx *gorm.DB, requestHash string) ([]*model.OrderTransactionInfo, error)
	GetOrderTransactionPending(ctx context.Context, tx *gorm.DB) ([]*model.OrderTransactionInfo, error)
	GetPendingOrderTransaction(ctx context.Context, tx *gorm.DB) ([]*model.OrderTransactionStatus, error)
	CreateOrder(ctx context.Context, tx *gorm.DB, userID uint64, amount int64) (*do.OrderInfo, error)
	CreateOrderByUsername(ctx context.Context, tx *gorm.DB, username string, amount int64) (*do.OrderInfo, error)
	CreateOrderForAdmin(ctx context.Context, tx *gorm.DB, adminUserID uint64, amount int64, utxoHash string) (*do.OrderInfo, error)
	GetOrderCount(ctx context.Context, tx *gorm.DB) (int64, error)
	GetOrderCountByUsername(ctx context.Context, tx *gorm.DB, username string) (int64, error)
	SetOrderStatus(ctx context.Context, tx *gorm.DB, orderID uint64, newStatus int) error
	SetOrderUTXOHash(ctx context.Context, tx *gorm.DB, orderID uint64, utxoHash string) error
	CreateOrderTransaction(ctx context.Context, tx *gorm.DB, transaction *model.OrderTransaction) (*do.OrderTransactionInfo, error)
	GetTransactions(ctx context.Context, tx *gorm.DB, page int, num int, withAddress bool, positiveOrder bool) ([]*model.OrderTransactionDetails, error)
	GetTransactionNum(ctx context.Context, tx *gorm.DB) (int64, error)
	SetOrderTransactionRequestHashByIDs(ctx context.Context, tx *gorm.DB, IDs []uint64, requestHash string) error
	SetOrderTransactionInfoByIDs(ctx context.Context, tx *gorm.DB, IDs []uint64, txHash string, previousStatus int, newStatus int, height int) error
	SetOrderTransactionInfoByTxHash(ctx context.Context, tx *gorm.DB, txHash string, newStatus int, height int) error
	GetLatestRewardInterval(ctx context.Context, tx *gorm.DB) (int, error)
	GetLatestUserShareInfo(ctx context.Context, tx *gorm.DB) (*do.UserShareInfo, error)
	GetOrderDetailsByUsername(ctx context.Context, tx *gorm.DB, username string, page int, num int, withAddress bool, positiveOrder bool) ([]*model.OrderDetails, error)
	GetAllocationCount(ctx context.Context, db *gorm.DB) (int64, error)
	GetAllocation(ctx context.Context, db *gorm.DB, page int, num int, positiveOrder bool) ([]*do.AllocationInfo, error)
	GetAllocationCountByUsername(ctx context.Context, db *gorm.DB, username string) (int64, error)
	GetAllocationByUsername(ctx context.Context, db *gorm.DB, username string, page int, num int, positiveOrder bool) ([]*do.AllocationInfo, error)
}

type UserServiceImpl struct {
	userInfoDao             dao.UserInfoDAO
	detailedShareInfoDao    dao.DetailedShareInfoDAO
	userShareInfoDao        dao.UserShareInfoDAO
	minedBlockInfoDao       dao.MinedBlockInfoDAO
	allocationInfoDAO       dao.AllocationInfoDAO
	metaInfoDAO             dao.MetaInfoDAO
	orderInfoDAO            dao.OrderInfoDAO
	orderTransactionInfoDAO dao.OrderTransactionInfoDAO
}

var userService UserService = &UserServiceImpl{
	userInfoDao:             dao.GetUserInfoDAOImpl(),
	detailedShareInfoDao:    dao.GetDetailedShareInfoDAOImpl(),
	userShareInfoDao:        dao.GetUserShareInfoDAOImpl(),
	minedBlockInfoDao:       dao.GetMinedBlockInfoDAOImpl(),
	allocationInfoDAO:       dao.GetAllocationInfoDAOImpl(),
	metaInfoDAO:             dao.GetMetaInfoDAOImpl(),
	orderInfoDAO:            dao.GetOrderInfoDAOImpl(),
	orderTransactionInfoDAO: dao.GetOrderTransactionInfoDAOImpl(),
}

func GetUserService() UserService {
	return userService
}

func (u *UserServiceImpl) GetUserByUsername(ctx context.Context, tx *gorm.DB, username string) (*do.UserInfo, error) {
	if utils.IsBlank(username) || len(username) > constdef.MaxUsernameLength {
		return nil, fmt.Errorf("invalid username %v: blank or exceed max length", username)
	}

	info, err := u.userInfoDao.GetByUsername(ctx, tx, username)
	return info, err
}

func (u *UserServiceImpl) GetUserByID(ctx context.Context, tx *gorm.DB, id uint64) (*do.UserInfo, error) {
	info, err := u.userInfoDao.GetByID(ctx, tx, id)
	return info, err
}

func (u *UserServiceImpl) UsernameExist(ctx context.Context, tx *gorm.DB, username string) (bool, error) {
	if utils.IsBlank(username) || len(username) > constdef.MaxUsernameLength {
		return false, fmt.Errorf("invalid username %v: blank or exceed max length", username)
	}

	return u.userInfoDao.ExistUsername(ctx, tx, username)
}

func (u *UserServiceImpl) RegisterUser(ctx context.Context, tx *gorm.DB, username string, password string, address string, email *string) (*do.UserInfo, error) {

	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	address = strings.TrimSpace(address)

	usernameValid := utils.CheckUsernameValidity(username)
	if !usernameValid {
		return nil, pooljson.ErrInvalidUsername
	}

	passwordValid := utils.CheckPasswordValidity(password)
	if !passwordValid {
		return nil, pooljson.ErrInvalidPassword
	}

	usernameExist, err := u.userInfoDao.ExistUsername(ctx, tx, username)
	if err != nil {
		return nil, err
	}
	if usernameExist {
		return nil, pooljson.ErrAlreadyRegistered
	}

	newSalt, err := utils.GenerateRandomSalt(12, 4, 4, 4)
	if err != nil {
		return nil, err
	}

	newPassword := password + newSalt
	newPasswordHash := sha256.Sum256([]byte(newPassword))

	info := do.UserInfo{
		Username: username,
		Password: hex.EncodeToString(newPasswordHash[:])[:20],
		Address:  address,
		Salt:     newSalt,
		Email:    email,
	}
	res, err := u.userInfoDao.Create(ctx, tx, &info)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (u *UserServiceImpl) Login(ctx context.Context, tx *gorm.DB, username string, password string) (bool, *do.UserInfo, error) {

	usernameValid := utils.CheckUsernameValidity(username)
	if !usernameValid {
		return false, nil, pooljson.ErrInvalidUsername
	}

	// passwordValid := utils.CheckPasswordValidity(password)
	// if !passwordValid {
	// 	return false, nil, pooljson.ErrInvalidPassword
	// }

	userInfo, err := u.userInfoDao.GetByUsername(ctx, tx, username)
	if err != nil {
		return false, nil, err
	}

	if userInfo == nil {
		return false, nil, errors.New("cannot find user")
	}

	passwordHash := sha256.Sum256([]byte(password + userInfo.Salt))
	loginPassword := hex.EncodeToString(passwordHash[:])[:20]
	if userInfo.Password == loginPassword {
		return true, userInfo, nil
	}
	return true, userInfo, nil
	// return false, nil, nil
}

func (u *UserServiceImpl) GetShare(ctx context.Context, db *gorm.DB, page int, num int, positiveOrder bool) ([]*do.UserShareInfo, error) {
	return u.userShareInfoDao.Get(ctx, db, page, num, positiveOrder)
}

func (u *UserServiceImpl) GetShareCount(ctx context.Context, db *gorm.DB) (int64, error) {
	return u.userShareInfoDao.GetNum(ctx, db)
}

func (u *UserServiceImpl) GetShareByUsername(ctx context.Context, db *gorm.DB, username string, page int, num int, positiveOrder bool) ([]*do.UserShareInfo, error) {
	return u.userShareInfoDao.GetByUsername(ctx, db, username, page, num, positiveOrder)
}

func (u *UserServiceImpl) GetShareCountByUsername(ctx context.Context, db *gorm.DB, username string) (int64, error) {
	return u.userShareInfoDao.GetNumByUsername(ctx, db, username)
}

func (u *UserServiceImpl) AddShareByUsernameOld(ctx context.Context, db *gorm.DB, shareInfo *model.ShareInfo, rewardInterval int, recordDetail bool) error {
	if shareInfo == nil {
		return errors.New("nil share info")
	}

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		height := shareInfo.Height
		startHeight := height - height%int64(rewardInterval)
		endHeight := startHeight + int64(rewardInterval)

		userShareInfo, err := u.userShareInfoDao.GetByUsernameAndHeight(ctx, tx, shareInfo.Username, startHeight, endHeight, true)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("Get by username and height error: %v", err)
			return err
		}

		if userShareInfo == nil {
			newUserShareInfo := do.UserShareInfo{
				Username:    shareInfo.Username,
				ShareCount:  shareInfo.ShareCount,
				StartHeight: startHeight,
				EndHeight:   endHeight,
			}
			err := u.userShareInfoDao.Create(ctx, tx, &newUserShareInfo)
			if err != nil {
				log.Errorf("Create user share info error: %v", err)
				return err
			}
		} else {
			err := u.userShareInfoDao.AddShareByID(ctx, tx, userShareInfo.ID, shareInfo.ShareCount)
			if err != nil {
				log.Errorf("Add share by ID error: %v", err)
				return err
			}
		}

		if recordDetail {
			shareDO := model.ConvertShareInfoToDO(shareInfo)
			_, err = u.detailedShareInfoDao.Create(ctx, tx, shareDO)
			if err != nil {
				log.Errorf("Create detailed share info error: %v", err)
				return err
			}
		}

		return nil
	})

	return err
}

func (u *UserServiceImpl) AddShareByUsername(ctx context.Context, db *gorm.DB, shareInfo *model.ShareInfo, rewardInterval int, recordDetail bool) error {
	if shareInfo == nil {
		return errors.New("nil share info")
	}

	height := shareInfo.Height
	startHeight := height - height%int64(rewardInterval)
	endHeight := startHeight + int64(rewardInterval)

	shareDesc := utils.ShareDesc{
		Username:    shareInfo.Username,
		StartHeight: startHeight,
		EndHeight:   endHeight,
	}
	id, ok := utils.ShareIDCache.Get(shareDesc)

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		if ok {
			err := u.userShareInfoDao.AddShareByID(ctx, tx, id, shareInfo.ShareCount)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Errorf("Internal error: unable to add share by id, %v", err)
				return err
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				newUserShareInfo := do.UserShareInfo{
					Username:    shareInfo.Username,
					ShareCount:  shareInfo.ShareCount,
					StartHeight: startHeight,
					EndHeight:   endHeight,
				}
				err := u.userShareInfoDao.Create(ctx, tx, &newUserShareInfo)
				if err != nil {
					log.Errorf("Create user share info error: %v", err)
					return err
				}
				utils.ShareIDCache.Set(shareDesc, newUserShareInfo.ID)
			}
		} else {
			userShareInfo, err := u.userShareInfoDao.GetByUsernameAndHeight(ctx, tx, shareInfo.Username, startHeight, endHeight, true)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Errorf("Get by username and height error: %v", err)
				return err
			}
			if errors.Is(err, gorm.ErrRecordNotFound) {
				newUserShareInfo := do.UserShareInfo{
					Username:    shareInfo.Username,
					ShareCount:  shareInfo.ShareCount,
					StartHeight: startHeight,
					EndHeight:   endHeight,
				}
				err := u.userShareInfoDao.Create(ctx, tx, &newUserShareInfo)
				if err != nil {
					log.Errorf("Create user share info error: %v", err)
					return err
				}
				utils.ShareIDCache.Set(shareDesc, newUserShareInfo.ID)
			} else {
				err := u.userShareInfoDao.AddShareByID(ctx, tx, userShareInfo.ID, shareInfo.ShareCount)
				if err != nil {
					log.Errorf("Internal error: unable to add share by id, %v", err)
					return err
				}
				utils.ShareIDCache.Set(shareDesc, userShareInfo.ID)
			}
		}

		if recordDetail {
			shareDO := model.ConvertShareInfoToDO(shareInfo)
			_, err := u.detailedShareInfoDao.Create(ctx, tx, shareDO)
			if err != nil {
				log.Errorf("Create detailed share info error: %v", err)
				return err
			}
		}

		return nil
	})

	return err
}

func (u *UserServiceImpl) AddMinedBlock(ctx context.Context, tx *gorm.DB, shareInfo *model.ShareInfo) error {
	if shareInfo == nil {
		return errors.New("nil sharInfo")
	}

	minedBlockInfo := model.ConvertShareInfoToMinedBlockDO(shareInfo)

	_, err := u.minedBlockInfoDao.Create(ctx, tx, minedBlockInfo)
	return err
}

func (u *UserServiceImpl) ConnectBlock(ctx context.Context, tx *gorm.DB, blockHash string) error {
	if utils.IsBlank(blockHash) {
		return errors.New("empty blockHash")
	}

	_, err := u.minedBlockInfoDao.ConnectByBlockHash(ctx, tx, blockHash)
	return err
}

func (u *UserServiceImpl) GetMinedBlocks(ctx context.Context, tx *gorm.DB, page int, num int, positiveOrder bool) ([]*do.MinedBlockInfo, error) {
	return u.minedBlockInfoDao.Get(ctx, tx, page, num, positiveOrder)
}

func (u *UserServiceImpl) DisconnectBlock(ctx context.Context, tx *gorm.DB, blockHash string) error {
	if utils.IsBlank(blockHash) {
		return errors.New("empty blockHash")
	}

	_, err := u.minedBlockInfoDao.DisconnectByBlockHash(ctx, tx, blockHash)
	return err
}

func (u *UserServiceImpl) RejectBlock(ctx context.Context, tx *gorm.DB, blockNotification *model.BlockNotification) error {
	if blockNotification == nil {
		return errors.New("nil blockNotification")
	}

	_, err := u.minedBlockInfoDao.RejectedByBlockHash(ctx, tx, blockNotification.BlockHash.String(), blockNotification.Info)
	return err
}

func (u *UserServiceImpl) GetUnallocatedReward(ctx context.Context, tx *gorm.DB) (int64, error) {
	metaInfo, err := u.metaInfoDAO.Get(ctx, tx)
	if err != nil {
		log.Errorf("Internal error: unable to get meta info: %v", err)
		return 0, err
	}
	lastRewardHeight := metaInfo.LastRewardHeight

	connectedBlocks, err := u.minedBlockInfoDao.GetConnectedBlocksHigherThanHeight(ctx, tx, lastRewardHeight)
	if err != nil {
		log.Errorf("Internal error: unable to get connected blocks: %v", err)
		return 0, err
	}

	var res int64 = 0
	for _, block := range connectedBlocks {
		res += int64(block.Reward)
	}
	return res, nil
}

func (u *UserServiceImpl) RegisterAdmin(ctx context.Context, tx *gorm.DB, password string) error {
	password = strings.TrimSpace(password)

	usernameExist, err := u.userInfoDao.ExistUsername(ctx, tx, "admin")
	if err != nil {
		return err
	}
	if usernameExist {
		return errors.New("admin already exists")
	}

	newSalt, err := utils.GenerateRandomSalt(12, 4, 4, 4)
	if err != nil {
		return err
	}

	newPassword := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:"+password)) + newSalt
	newPasswordHash := sha256.Sum256([]byte(newPassword))

	info := do.UserInfo{
		Username: "admin",
		Password: hex.EncodeToString(newPasswordHash[:])[:20],
		Salt:     newSalt,
		Email:    nil,
	}
	_, err = u.userInfoDao.Create(ctx, tx, &info)
	return err
}

func (u *UserServiceImpl) LoginAdmin(ctx context.Context, tx *gorm.DB, authMsg string) (bool, error) {

	userInfo, err := u.userInfoDao.GetByUsername(ctx, tx, "admin")
	if err != nil {
		return false, err
	}

	if userInfo == nil {
		return false, errors.New("cannot find user admin")
	}

	passwordHash := sha256.Sum256([]byte(authMsg + userInfo.Salt))
	loginPassword := hex.EncodeToString(passwordHash[:])[:20]
	if userInfo.Password == loginPassword {
		return true, nil
	}
	return false, nil
}

func (u *UserServiceImpl) ChangePassword(ctx context.Context, tx *gorm.DB, username string, oldPassword string, newPassword string) error {
	usernameValid := utils.CheckUsernameValidity(username)
	if !usernameValid {
		return pooljson.ErrInvalidUsername
	}

	oldPasswordValid := utils.CheckPasswordValidity(oldPassword)
	newPasswordValid := utils.CheckPasswordValidity(newPassword)
	if !oldPasswordValid || !newPasswordValid {
		return pooljson.ErrInvalidPassword
	}

	success, _, err := u.Login(ctx, tx, username, oldPassword)
	if err != nil {
		return err
	}
	if !success {
		return pooljson.ErrPasswordIncorrect
	}

	err = u.ResetPassword(ctx, tx, username, newPassword)
	if err != nil {
		return err
	}

	return nil
}

func (u *UserServiceImpl) ResetPassword(ctx context.Context, tx *gorm.DB, username string, password string) error {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	usernameValid := utils.CheckUsernameValidity(username)
	if !usernameValid {
		return pooljson.ErrInvalidUsername
	}

	passwordValid := utils.CheckPasswordValidity(password)
	if !passwordValid {
		return pooljson.ErrInvalidPassword
	}

	usernameExist, err := u.userInfoDao.ExistUsername(ctx, tx, username)
	if err != nil {
		return err
	}
	if !usernameExist {
		return errors.New("username does not exist")
	}

	newSalt, err := utils.GenerateRandomSalt(12, 4, 4, 4)
	if err != nil {
		return err
	}

	newPassword := password + newSalt
	newPasswordHash := sha256.Sum256([]byte(newPassword))

	info := do.UserInfo{
		Username: username,
		Password: hex.EncodeToString(newPasswordHash[:])[:20],
		Salt:     newSalt,
	}
	res, err := u.userInfoDao.UpdateByUsername(ctx, tx, &info)
	if err != nil {
		return err
	}
	if res == 0 {
		return errors.New("username does not exist")
	}

	return nil
}

func (u *UserServiceImpl) AllocateRewards(ctx context.Context, tx *gorm.DB, startHeight int, endHeight int, manageFeePercent int) error {
	err := tx.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// Find connected and unallocated blocks.
		connectedBlocks, err := u.minedBlockInfoDao.GetUnallocatedBetweenHeight(ctx, tx, int64(startHeight), int64(endHeight), true)
		if err != nil {
			log.Errorf("Internal error: unable to get unallocated blocks between height: %v", err)
			return err
		}

		var totalRewards uint64 = 0
		for _, block := range connectedBlocks {
			totalRewards += block.Reward
		}
		log.Infof("Find %v mined blocks in height [%v, %v), total rewards: %v Neutrino", len(connectedBlocks), startHeight, endHeight, totalRewards)

		allAllocateRewards := int64(totalRewards * uint64(100-manageFeePercent) / 100)

		// Find users with share.
		users, err := u.userShareInfoDao.GetUnallocatedByHeight(ctx, tx, int64(startHeight), int64(endHeight))
		if err != nil {
			log.Errorf("Internal error: unable to get unallocated users with share, %v", err)
			return err
		}

		var totalShareCount int64 = 0
		for _, user := range users {
			totalShareCount += user.ShareCount
		}

		// Allocate rewards to each user, update their balance,
		// also update the allocation info.
		var rewardAllocated int64 = 0
		allocationInfos := make([]*do.AllocationInfo, 0)
		for _, user := range users {
			allocated := allAllocateRewards * user.ShareCount / totalShareCount
			rewardAllocated += allocated
			allocationInfo := &do.AllocationInfo{
				Username:    user.Username,
				ShareCount:  user.ShareCount,
				Balance:     allocated,
				StartHeight: int64(startHeight),
				EndHeight:   int64(endHeight),
			}
			allocationInfos = append(allocationInfos, allocationInfo)
			_, err := u.userInfoDao.AddBalanceByUsername(ctx, tx, user.Username, allocated)
			if err != nil {
				log.Errorf("Add balance for user %v fail: %v", user.Username, err)
				return err
			}
		}
		_, err = u.allocationInfoDAO.MCreate(ctx, tx, allocationInfos)
		if err != nil {
			log.Errorf("Multi create allocation infos fail: %v", err)
			return err
		}

		manageFee := int64(totalRewards) - rewardAllocated
		log.Infof("Allocating %v Neutrino to miners, manage fee %v Neutrino", rewardAllocated, manageFee)
		_, err = u.metaInfoDAO.AddBalance(ctx, tx, manageFee)
		if err != nil {
			log.Errorf("Add balance for admin fail: %v", err)
			return err
		}
		_, err = u.allocationInfoDAO.Create(ctx, tx, &do.AllocationInfo{
			Username:    "admin",
			ShareCount:  0,
			Balance:     manageFee,
			StartHeight: int64(startHeight),
			EndHeight:   int64(endHeight),
		})

		_, err = u.metaInfoDAO.AddAllocatedBalance(ctx, tx, rewardAllocated)
		if err != nil {
			log.Errorf("Add allocated balance fail: %v", err)
			return err
		}
		metaInfo, err := u.metaInfoDAO.Get(ctx, tx)
		if err != nil {
			log.Errorf("Get meta info fail: %v", err)
			return err
		}
		if metaInfo.LastRewardHeight < int64(endHeight) {
			_, err = u.metaInfoDAO.UpdateLastRewardHeight(ctx, tx, int64(endHeight))
			if err != nil {
				log.Errorf("Update last reward height fail: %v", err)
				return err
			}
		}

		// Tag user share info as allocated.
		allocatedIDs := make([]uint64, 0)
		for _, user := range users {
			allocatedIDs = append(allocatedIDs, user.ID)
		}
		err = u.userShareInfoDao.SetAllocated(ctx, tx, allocatedIDs)
		if err != nil {
			log.Errorf("Set allocated for user share info error: %v", err)
			return err
		}

		// Tag mined block as allocated.
		allocatedBlockIDs := make([]uint64, 0)
		for _, block := range connectedBlocks {
			allocatedBlockIDs = append(allocatedBlockIDs, block.ID)
		}
		err = u.minedBlockInfoDao.SetAllocated(ctx, tx, allocatedBlockIDs)
		if err != nil {
			log.Errorf("Set allocated for mined blocks error: %v", err)
			return err
		}

		return nil
	})

	return err
}

func (u *UserServiceImpl) GetUnallocatedByHeight(ctx context.Context, tx *gorm.DB, startHeight int, endHeight int) ([]*do.MinedBlockInfo, error) {
	connectedBlocks, err := u.minedBlockInfoDao.GetUnallocatedBetweenHeight(ctx, tx, int64(startHeight), int64(endHeight), true)
	if err != nil {
		log.Errorf("Internal error: unable to get unallocated blocks between height: %v", err)
		return nil, err
	}

	return connectedBlocks, nil
}

func (u *UserServiceImpl) GetAcceptedBlocksByHeight(ctx context.Context, tx *gorm.DB, startHeight int, endHeight int) ([]*do.MinedBlockInfo, error) {
	acceptedBlocks, err := u.minedBlockInfoDao.GetAcceptedBlocksBetweenHeight(ctx, tx, int64(startHeight), int64(endHeight), true)
	if err != nil {
		log.Errorf("Internal error: unable to get accepted blocks between height: %v", err)
		return nil, err
	}

	return acceptedBlocks, nil
}

func (u *UserServiceImpl) AllocateRewardsWithBlocks(ctx context.Context, tx *gorm.DB, startHeight int, endHeight int,
	manageFeePercent int, connectedBlocks []*do.MinedBlockInfo) error {
	err := tx.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		var totalRewards uint64 = 0
		for _, block := range connectedBlocks {
			totalRewards += block.Reward
		}
		log.Infof("Find %v mined blocks in height [%v, %v), total rewards: %v Neutrino", len(connectedBlocks), startHeight, endHeight, totalRewards)

		allAllocateRewards := int64(totalRewards * uint64(100-manageFeePercent) / 100)

		// Find users with share.
		users, err := u.userShareInfoDao.GetUnallocatedByHeight(ctx, tx, int64(startHeight), int64(endHeight))
		if err != nil {
			log.Errorf("Internal error: unable to get unallocated users with share, %v", err)
			return err
		}

		var totalShareCount int64 = 0
		for _, user := range users {
			totalShareCount += user.ShareCount
		}

		userToShareCount := make(map[string]int64)
		for _, user := range users {
			count, ok := userToShareCount[user.Username]
			if ok {
				userToShareCount[user.Username] = count + user.ShareCount
			} else {
				userToShareCount[user.Username] = user.ShareCount
			}
		}

		// Allocate rewards to each user, update their balance,
		// also update the allocation info.
		var rewardAllocated int64 = 0
		allocationInfos := make([]*do.AllocationInfo, 0)
		for username, shareCount := range userToShareCount {
			allocated := int64(float64(allAllocateRewards) * (float64(shareCount) / float64(totalShareCount)))
			rewardAllocated += allocated
			allocationInfo := &do.AllocationInfo{
				Username:    username,
				ShareCount:  shareCount,
				Balance:     allocated,
				StartHeight: int64(startHeight),
				EndHeight:   int64(endHeight),
			}
			allocationInfos = append(allocationInfos, allocationInfo)
			_, err := u.userInfoDao.AddBalanceByUsername(ctx, tx, username, allocated)
			if err != nil {
				log.Errorf("Add balance for user %v fail: %v", username, err)
				return err
			}
		}
		_, err = u.allocationInfoDAO.MCreate(ctx, tx, allocationInfos)
		if err != nil {
			log.Errorf("Multi create allocation infos fail: %v", err)
			return err
		}

		manageFee := int64(totalRewards) - rewardAllocated
		log.Infof("Allocating %v Neutrino to miners, manage fee %v Neutrino", rewardAllocated, manageFee)
		_, err = u.metaInfoDAO.AddBalance(ctx, tx, manageFee)
		if err != nil {
			log.Errorf("Add balance for admin fail: %v", err)
			return err
		}
		_, err = u.allocationInfoDAO.Create(ctx, tx, &do.AllocationInfo{
			Username:    "admin",
			ShareCount:  0,
			Balance:     manageFee,
			StartHeight: int64(startHeight),
			EndHeight:   int64(endHeight),
		})

		_, err = u.metaInfoDAO.AddAllocatedBalance(ctx, tx, rewardAllocated)
		if err != nil {
			log.Errorf("Add allocated balance fail: %v", err)
			return err
		}
		metaInfo, err := u.metaInfoDAO.Get(ctx, tx)
		if err != nil {
			log.Errorf("Get meta info fail: %v", err)
			return err
		}
		if metaInfo.LastRewardHeight < int64(endHeight) {
			_, err = u.metaInfoDAO.UpdateLastRewardHeight(ctx, tx, int64(endHeight))
			if err != nil {
				log.Errorf("Update last reward height fail: %v", err)
				return err
			}
		}

		// Tag user share info as allocated.
		allocatedIDs := make([]uint64, 0)
		for _, user := range users {
			allocatedIDs = append(allocatedIDs, user.ID)
		}
		err = u.userShareInfoDao.SetAllocated(ctx, tx, allocatedIDs)
		if err != nil {
			log.Errorf("Set allocated for user share info error: %v", err)
			return err
		}

		// Tag mined block as allocated.
		allocatedBlockIDs := make([]uint64, 0)
		for _, block := range connectedBlocks {
			allocatedBlockIDs = append(allocatedBlockIDs, block.ID)
		}
		err = u.minedBlockInfoDao.SetAllocated(ctx, tx, allocatedBlockIDs)
		if err != nil {
			log.Errorf("Set allocated for mined blocks error: %v", err)
			return err
		}

		return nil
	})

	return err
}

func (u *UserServiceImpl) AllocateRewardsBeforeHeight(ctx context.Context, tx *gorm.DB, height int, manageFeePercent int) error {
	err := tx.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		log.Debugf("Looking for unallocated user share before height %v...", height)
		unallocatedShares, err := u.userShareInfoDao.GetUnallocatedBeforeHeight(ctx, tx, int64(height))
		if err != nil {
			log.Errorf("Get unallocated before height err: %v", err)
			return err
		}
		if len(unallocatedShares) == 0 {
			log.Debugf("Unallocated user share not found.")
			return nil
		}

		needAllocatedHeight := make(map[model.HeightPair]struct{})
		for _, unallocatedShare := range unallocatedShares {
			tmp := model.HeightPair{
				StartHeight: unallocatedShare.StartHeight,
				EndHeight:   unallocatedShare.EndHeight,
			}
			needAllocatedHeight[tmp] = struct{}{}
		}

		log.Infof("Find %v ranges that have not been allocated", len(needAllocatedHeight))
		for heightPair := range needAllocatedHeight {
			log.Infof("Allocating reward in height [%v, %v)", heightPair.StartHeight, heightPair.EndHeight)
			err := u.AllocateRewards(ctx, tx, int(heightPair.StartHeight), int(heightPair.EndHeight), manageFeePercent)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func (u *UserServiceImpl) GetUnallocatedRanges(ctx context.Context, tx *gorm.DB, height int) (map[model.HeightPair]struct{}, error) {
	log.Debugf("Looking for unallocated user share before height %v...", height)
	unallocatedShares, err := u.userShareInfoDao.GetUnallocatedBeforeHeight(ctx, tx, int64(height))
	if err != nil {
		log.Errorf("Get unallocated before height err: %v", err)
		return nil, err
	}
	if len(unallocatedShares) == 0 {
		log.Debugf("Unallocated user share not found.")
		return nil, err
	}

	needAllocatedHeight := make(map[model.HeightPair]struct{})
	for _, unallocatedShare := range unallocatedShares {
		tmp := model.HeightPair{
			StartHeight: unallocatedShare.StartHeight,
			EndHeight:   unallocatedShare.EndHeight,
		}
		needAllocatedHeight[tmp] = struct{}{}
	}
	log.Infof("Find %v ranges that have not been allocated", len(needAllocatedHeight))
	return needAllocatedHeight, nil
}

// CreateOrders create an order
func (u *UserServiceImpl) CreateOrders(ctx context.Context, tx *gorm.DB, unitCnt int, unit uint64, utxoList []*abejson.UTXO) error {
	// previous allocate but unfinished order i.e. utxo_hash='' and status=0 and amount = ?
	orderMapping := map[int][]*do.OrderInfo{}
	for _, multiple := range []int{1, 2, 4} {
		orderInfos, err := u.orderInfoDAO.GetUnallocatedOrdersByAmount(ctx, tx, uint64(multiple)*unit)
		if err != nil {
			return err
		}
		if len(orderInfos) != 0 {
			orderMapping[multiple] = orderInfos
			unitCnt -= multiple * len(orderInfos)
		}
	}

	//  get the information for all redeemable user i.e. balance > redeemed_balance
	userInfos, err := u.userInfoDao.GetNeedAllocate(ctx, tx)
	if err != nil {
		log.Errorf("Get unallocated user err: %v", err)
		return err
	}

	// allocate remain unit to users with available balance
	orderMapping, err = u.createOrderForUsers(ctx, tx, unitCnt, unit, userInfos, orderMapping)
	if err != nil {
		return err
	}

	// allocate the utxo to unallocated order
	return u.allocateUTXOForOrders(ctx, tx, utxoList, orderMapping)
}

func (u *UserServiceImpl) createOrderForUsers(
	ctx context.Context,
	tx *gorm.DB,
	unitCnt int,
	unit uint64,
	userInfos []*do.UserInfo,
	orderMapping map[int][]*do.OrderInfo,
) (map[int][]*do.OrderInfo, error) {
	log.Debugf("Get %d user for allocating reward", len(userInfos))
	for i := 0; i < len(userInfos); i++ {
		err := tx.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			log.Debugf("username %s: balance %d redeemed_balance %d", userInfos[i].Username, userInfos[i].Balance, userInfos[i].RedeemedBalance)
			remainBalance := userInfos[i].Balance - userInfos[i].RedeemedBalance
			currentBlockRewardQuarter := int64(unit)
			if 0 < remainBalance && remainBalance < currentBlockRewardQuarter {
				return nil
			}
			level := remainBalance / currentBlockRewardQuarter
			for unitCnt > 0 && level >= 1 {
				currentRedeemedBalance := int64(0)
				allocateLevel := 0
				switch {
				case unitCnt >= 4 && level >= 4:
					currentRedeemedBalance = currentBlockRewardQuarter * 4
					unitCnt -= 4
					allocateLevel = 4
				case unitCnt >= 2 && level >= 2:
					currentRedeemedBalance = currentBlockRewardQuarter * 2
					unitCnt -= 2
					allocateLevel = 2
				case unitCnt >= 1 && level >= 1:
					currentRedeemedBalance = currentBlockRewardQuarter
					unitCnt -= 1
					allocateLevel = 1
				}
				if currentRedeemedBalance == 0 {
					break
				}
				log.Debugf("Create order for user %d with %d for allocating reward", userInfos[i].ID, currentRedeemedBalance)
				var order *do.OrderInfo
				order, err := u.CreateOrder(ctx, tx, userInfos[i].ID, currentRedeemedBalance)
				if err != nil {
					log.Errorf("Create a order for user ID %v err: %v", userInfos[i].ID, err)
					return err
				}
				if _, ok := orderMapping[allocateLevel]; !ok {
					orderMapping[allocateLevel] = make([]*do.OrderInfo, 0, unitCnt/allocateLevel)
				}
				orderMapping[allocateLevel] = append(orderMapping[allocateLevel], order)
				remainBalance = remainBalance - currentRedeemedBalance
				level = remainBalance / currentBlockRewardQuarter
			}
			return nil
		})
		if err != nil {
			log.Errorf("Create a order for user ID %v err: %v", userInfos[i].ID, err)
			return orderMapping, err
		}
	}
	return orderMapping, nil
}

func (u *UserServiceImpl) allocateUTXOForOrders(
	ctx context.Context,
	tx *gorm.DB,
	utxoList []*abejson.UTXO,
	orderMapping map[int][]*do.OrderInfo,
) error {
	var err error
	// allocate utxo hash with corresponding orders
	currentUTXOListIndex := 0
	// [4]
	indexFour := 0
	for indexFour < len(orderMapping[4]) {
		// find allocatable utxo
		for currentUTXOListIndex < len(utxoList) && utxoList[currentUTXOListIndex].Allocated {
			currentUTXOListIndex += 1
		}
		if currentUTXOListIndex < len(utxoList) {
			err = u.orderInfoDAO.UpdateUTXOHashByID(
				ctx,
				tx,
				orderMapping[4][indexFour].ID,
				utxoList[currentUTXOListIndex].UTXOHash.String())
			if err != nil {
				log.Errorf("Allocate utxo to order ID %v err: %v", orderMapping[4][indexFour].ID, err)
				return err
			}
			log.Infof("Allocate utxo %s to order ID %v", utxoList[currentUTXOListIndex].UTXOHash.String(), orderMapping[4][indexFour].ID)

			indexFour += 1
			currentUTXOListIndex += 1
		}
		if currentUTXOListIndex == len(utxoList) {
			break
		}
	}

	// [2,2]
	indexTwo := 0
	for indexTwo+1 < len(orderMapping[2]) {
		for currentUTXOListIndex < len(utxoList) && utxoList[currentUTXOListIndex].Allocated {
			currentUTXOListIndex += 1
		}
		if currentUTXOListIndex < len(utxoList) {
			err = tx.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				for j := 0; j < 2; j++ {
					err = u.orderInfoDAO.UpdateUTXOHashByID(
						ctx,
						tx,
						orderMapping[2][indexTwo+j].ID,
						utxoList[currentUTXOListIndex].UTXOHash.String())
					if err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				log.Errorf("Allocate utxo to order ID %v,%v err: %v", orderMapping[2][indexTwo].ID, orderMapping[2][indexTwo+1].ID, err)
				return err
			}
			log.Infof("Allocate utxo %s to order ID %v,%v", utxoList[currentUTXOListIndex].UTXOHash.String(), orderMapping[2][indexTwo].ID, orderMapping[2][indexTwo+1].ID)

			indexTwo += 2
			currentUTXOListIndex += 1
		}
		if currentUTXOListIndex == len(utxoList) {
			break
		}
	}

	// [1,1,1,1]
	indexOne := 0
	for indexOne+3 < len(orderMapping[1]) {
		for currentUTXOListIndex < len(utxoList) && utxoList[currentUTXOListIndex].Allocated {
			currentUTXOListIndex += 1
		}
		if currentUTXOListIndex < len(utxoList) {
			err = tx.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				for j := 0; j < 4; j++ {
					err = u.orderInfoDAO.UpdateUTXOHashByID(
						ctx,
						tx,
						orderMapping[1][indexOne+j].ID,
						utxoList[currentUTXOListIndex].UTXOHash.String())
					if err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				log.Errorf("Allocate utxo to order ID %v,%v,%v,%v err: %v", orderMapping[1][indexOne].ID, orderMapping[1][indexOne+1].ID, orderMapping[1][indexOne+2].ID, orderMapping[1][indexOne+3].ID, err)
				return err
			}
			log.Infof("Allocate utxo %s to order ID %v,%v,%v,%v", utxoList[currentUTXOListIndex].UTXOHash.String(), orderMapping[1][indexOne].ID, orderMapping[1][indexOne+1].ID, orderMapping[1][indexOne+2].ID, orderMapping[1][indexOne+3].ID)

			indexOne += 4
			currentUTXOListIndex += 1
		}
		if currentUTXOListIndex == len(utxoList) {
			break
		}
	}

	// [2,1,1]
	for indexTwo < len(orderMapping[2]) && indexOne+1 < len(orderMapping[1]) {
		for currentUTXOListIndex < len(utxoList) && utxoList[currentUTXOListIndex].Allocated {
			currentUTXOListIndex += 1
		}
		if currentUTXOListIndex < len(utxoList) {
			err = tx.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				err = u.orderInfoDAO.UpdateUTXOHashByID(
					ctx,
					tx,
					orderMapping[2][indexTwo].ID,
					utxoList[currentUTXOListIndex].UTXOHash.String())
				if err != nil {
					return err
				}
				for j := 0; j < 2; j++ {
					err = u.orderInfoDAO.UpdateUTXOHashByID(
						ctx,
						tx,
						orderMapping[1][indexOne+j].ID,
						utxoList[currentUTXOListIndex].UTXOHash.String())
					if err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				log.Errorf("Create a order for user ID %v,%v,%v err: %v", orderMapping[2][indexTwo].ID, orderMapping[1][indexOne].ID, orderMapping[1][indexOne+1].ID, err)
				return err
			}
			log.Infof("Allocate utxo %s to order ID %v,%v,%v", utxoList[currentUTXOListIndex].UTXOHash.String(), orderMapping[2][indexTwo].ID, orderMapping[1][indexOne].ID, orderMapping[1][indexOne+1].ID)

			indexTwo += 1
			indexOne += 2
			currentUTXOListIndex += 1
		}
		if currentUTXOListIndex == len(utxoList) {
			break
		}
	}

	return nil
}

func (u *UserServiceImpl) GetAllUnfinishedOrders(ctx context.Context, tx *gorm.DB) ([]*do.OrderInfo, error) {
	orderInfos, err := u.orderInfoDAO.GetAllUnfinished(ctx, tx)
	if err != nil {
		log.Errorf("Get unallocated user err: %v", err)
		return nil, err
	}
	return orderInfos, nil
}

func (u *UserServiceImpl) GetUnfinishedOrders(ctx context.Context, tx *gorm.DB) ([]*do.OrderInfo, error) {
	orderInfos, err := u.orderInfoDAO.GetUnfinished(ctx, tx)
	if err != nil {
		log.Errorf("Get unallocated user err: %v", err)
		return nil, err
	}
	return orderInfos, nil
}
func (u *UserServiceImpl) GetUnfinishedOrderTransactions(ctx context.Context, tx *gorm.DB) ([]*do.OrderTransactionInfo, error) {
	orderTransactionInfos, err := u.orderTransactionInfoDAO.GetUnfinishedOrderTransactions(ctx, tx)
	if err != nil {
		log.Errorf("Get unallocated user err: %v", err)
		return nil, err
	}
	return orderTransactionInfos, nil
}

func (u *UserServiceImpl) GetUnfinishedOrdersByAmountUnit(ctx context.Context, tx *gorm.DB, unit int64) (map[string][]*do.OrderInfo, error) {
	res := map[string][]*do.OrderInfo{}
	for _, multiple := range []int{1, 2, 4} {
		orderInfos, err := u.orderInfoDAO.GetUnfinishedOrdersByAmount(ctx, tx, int64(multiple)*unit)
		if err != nil {
			log.Errorf("Get unallocated user err: %v", err)
			return nil, err
		}
		for i := 0; i < len(orderInfos); i++ {
			if _, ok := res[orderInfos[i].UTXOHash]; !ok {
				res[orderInfos[i].UTXOHash] = make([]*do.OrderInfo, 0, 5)
			}
			res[orderInfos[i].UTXOHash] = append(res[orderInfos[i].UTXOHash], orderInfos[i])
		}
	}

	return res, nil
}

func (u *UserServiceImpl) GetOrderTransactionInfoRange(ctx context.Context, tx *gorm.DB, lowHeight int64, highHeight int64) ([]*model.OrderTransactionStatus, error) {
	// Get all not appending order_transaction, check the
	txInfos, err := u.orderTransactionInfoDAO.GetTransactionRange(ctx, tx, lowHeight, highHeight)
	if err != nil {
		log.Errorf("Get need to check transaction hash err: %v", err)
		return nil, err
	}
	res := make([]*model.OrderTransactionStatus, len(txInfos))
	for i := 0; i < len(txInfos); i++ {
		res[i] = &model.OrderTransactionStatus{
			ID:                     txInfos[i].ID,
			OrderID:                txInfos[i].OrderID,
			Status:                 int(txInfos[i].Status),
			TransactionHash:        txInfos[i].TransactionHash,
			RequestTransactionHash: txInfos[i].TransactionRequestHash,
			Height:                 txInfos[i].Height,
			LastUpdateStatusHeight: txInfos[i].LastCheckedHeight,
		}
	}
	return res, nil
}

func (u *UserServiceImpl) GetTransactionRequestHash(ctx context.Context, tx *gorm.DB) ([]string, error) {
	transactionRequestHash, err := u.orderTransactionInfoDAO.GetTransactionRequestHashPending(ctx, tx)
	if err != nil {
		log.Errorf("Get transaction request hash err: %v", err)
		return nil, err
	}
	return transactionRequestHash, nil

}

func (u *UserServiceImpl) GetOrderTransactionByTransactionRequestHash(ctx context.Context, tx *gorm.DB, requestHash string) ([]*model.OrderTransactionInfo, error) {
	orderTransactionInfos, err := u.orderTransactionInfoDAO.GetOrderTransactionPendingWithRequestHash(ctx, tx, requestHash)
	if err != nil {
		log.Errorf("Get need to check transaction hash err: %v", err)
		return nil, err
	}
	res := make([]*model.OrderTransactionInfo, len(orderTransactionInfos))
	for i := 0; i < len(orderTransactionInfos); i++ {
		res[i] = &model.OrderTransactionInfo{
			ID:       orderTransactionInfos[i].ID,
			OrderID:  orderTransactionInfos[i].OrderID,
			Amount:   orderTransactionInfos[i].Amount,
			Address:  orderTransactionInfos[i].Address,
			Status:   int(orderTransactionInfos[i].Status),
			UserID:   orderTransactionInfos[i].OrderInfo.UserID,
			UTXOHash: orderTransactionInfos[i].OrderInfo.UTXOHash,
		}
	}
	return res, nil
}
func (u *UserServiceImpl) GetOrderTransactionPending(ctx context.Context, tx *gorm.DB) ([]*model.OrderTransactionInfo, error) {
	orderTransactionInfos, err := u.orderTransactionInfoDAO.GetTransactionPending(ctx, tx)
	if err != nil {
		log.Errorf("Get need to check transaction hash err: %v", err)
		return nil, err
	}
	res := make([]*model.OrderTransactionInfo, len(orderTransactionInfos))
	for i := 0; i < len(orderTransactionInfos); i++ {
		res[i] = &model.OrderTransactionInfo{
			ID:       orderTransactionInfos[i].ID,
			OrderID:  orderTransactionInfos[i].OrderID,
			Amount:   orderTransactionInfos[i].Amount,
			Address:  orderTransactionInfos[i].Address,
			Status:   int(orderTransactionInfos[i].Status),
			UserID:   orderTransactionInfos[i].OrderInfo.UserID,
			UTXOHash: orderTransactionInfos[i].OrderInfo.UTXOHash,
		}
	}
	return res, nil
}

func (u *UserServiceImpl) GetPendingOrderTransaction(ctx context.Context, tx *gorm.DB) ([]*model.OrderTransactionStatus, error) {
	// Get all not appending order_transaction, check the
	txInfos, err := u.orderTransactionInfoDAO.GetPendingTransactions(ctx, tx)
	if err != nil {
		log.Errorf("Get need to check transaction hash err: %v", err)
		return nil, err
	}
	res := make([]*model.OrderTransactionStatus, len(txInfos))
	for i := 0; i < len(txInfos); i++ {
		res[i] = &model.OrderTransactionStatus{
			ID:                     txInfos[i].ID,
			OrderID:                txInfos[i].OrderID,
			Status:                 int(txInfos[i].Status),
			RequestTransactionHash: txInfos[i].TransactionRequestHash,
		}
	}
	return res, nil
}

// CreateOrder create a new order which sent some balance to a user (specified by userID).
// It will create a new row in order_infos and modify redeemed_balance in user_infos.
// This function should be called under database transaction
func (u *UserServiceImpl) CreateOrder(ctx context.Context, tx *gorm.DB, userID uint64, amount int64) (*do.OrderInfo, error) {
	var res *do.OrderInfo

	newOrder := do.OrderInfo{
		UserID: userID,
		Amount: amount,
	}

	var err error
	res, err = u.orderInfoDAO.Create(ctx, tx, &newOrder)
	if err != nil {
		log.Errorf("Create order err: %v", err)
		return nil, err
	}

	_, err = u.userInfoDao.AddRedeemedBalanceByID(ctx, tx, userID, amount)
	if err != nil {
		log.Errorf("Add redeemed balance by ID err: %v", err)
		return nil, err
	}

	return res, err
}

// CreateOrderByUsername create a new order which sent some balance to a user (specified by username).
// It will create a new row in order_infos and modify redeemed_balance in user_infos.
func (u *UserServiceImpl) CreateOrderByUsername(ctx context.Context, tx *gorm.DB, username string, amount int64) (*do.OrderInfo, error) {
	var res *do.OrderInfo

	err := tx.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		userInfo, err := u.userInfoDao.GetByUsername(ctx, tx, username)
		if err != nil {
			log.Errorf("get by username err: %v", err)
			return err
		}
		newOrder := do.OrderInfo{
			UserID: userInfo.ID,
			Amount: amount,
		}

		res, err = u.orderInfoDAO.Create(ctx, tx, &newOrder)
		if err != nil {
			log.Errorf("Create order err: %v", err)
			return err
		}

		_, err = u.userInfoDao.AddRedeemedBalanceByID(ctx, tx, userInfo.ID, amount)
		if err != nil {
			log.Errorf("Create order err: %v", err)
			return err
		}
		return nil
	})

	return res, err
}

// CreateOrderForAdmin create a new order for admin
// It will create a new row in order_infos
// This function should be called under database transaction
func (u *UserServiceImpl) CreateOrderForAdmin(ctx context.Context, tx *gorm.DB, adminUserID uint64, amount int64, utxoHash string) (*do.OrderInfo, error) {
	newOrder := do.OrderInfo{
		UserID:   adminUserID,
		Amount:   amount,
		Status:   1,
		UTXOHash: utxoHash,
	}

	res, err := u.orderInfoDAO.Create(ctx, tx, &newOrder)
	if err != nil {
		log.Errorf("Create order err: %v", err)
		return nil, err
	}

	return res, err
}

// SetOrderStatus modifies the status of an order.
// 0: transaction not sent
// 1: transaction sent
// 2: unable to send transactions (manual required)
func (u *UserServiceImpl) SetOrderStatus(ctx context.Context, tx *gorm.DB, orderID uint64, newStatus int) error {
	err := u.orderInfoDAO.UpdateStatusByID(ctx, tx, orderID, newStatus)
	return err
}

func (u *UserServiceImpl) SetOrderUTXOHash(ctx context.Context, tx *gorm.DB, orderID uint64, utxoHash string) error {
	err := u.orderInfoDAO.UpdateUTXOHashByID(ctx, tx, orderID, utxoHash)
	return err
}

func (u *UserServiceImpl) CreateOrderTransaction(ctx context.Context, tx *gorm.DB, transaction *model.OrderTransaction) (*do.OrderTransactionInfo, error) {
	orderDO := do.OrderTransactionInfo{
		OrderID:                transaction.OrderID,
		Status:                 do.TransactionPending,
		Amount:                 transaction.Amount,
		Address:                transaction.Address,
		TransactionRequestHash: transaction.RequestTransactionHash,
	}
	return u.orderTransactionInfoDAO.Create(ctx, tx, &orderDO)
}

func (u *UserServiceImpl) GetTransactions(ctx context.Context, tx *gorm.DB, page int, num int, withAddress bool, positiveOrder bool) ([]*model.OrderTransactionDetails, error) {
	transactions, err := u.orderTransactionInfoDAO.Get(ctx, tx, page, num, withAddress, positiveOrder)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	res := make([]*model.OrderTransactionDetails, 0)
	for _, transaction := range transactions {
		tmp := &model.OrderTransactionDetails{
			ID:              transaction.ID,
			TransactionHash: transaction.TransactionHash,
			Amount:          transaction.Amount,
			Status:          transaction.Status,
			Height:          transaction.Height,
			Address:         transaction.Address,
			CreatedAt:       transaction.CreatedAt.Unix(),
			UpdatedAt:       transaction.UpdatedAt.Unix(),
		}
		res = append(res, tmp)
	}
	return res, nil
}

func (u *UserServiceImpl) SetOrderTransactionRequestHashByIDs(ctx context.Context, tx *gorm.DB, IDs []uint64, requestHash string) error {
	err := u.orderTransactionInfoDAO.UpdateByIDs(ctx, tx, IDs, map[string]interface{}{
		"transaction_request_hash": requestHash,
	})
	if err != nil {
		log.Errorf("update order_transaction_infos error with ids [%v] and request hash %v: %v", IDs, requestHash, err)
		return err
	}

	return nil
}
func (u *UserServiceImpl) SetOrderTransactionInfoByIDs(ctx context.Context, tx *gorm.DB, IDs []uint64, txHash string, previousStatus int, newStatus int, height int) error {
	params := map[string]interface{}{
		"transaction_hash": txHash,
	}
	if previousStatus != newStatus {
		params["status"] = newStatus
		params["height"] = height
		params["last_checked_height"] = height
	} else {
		params["last_checked_height"] = height
	}

	err := u.orderTransactionInfoDAO.UpdateByIDs(ctx, tx, IDs, params)
	if err != nil {
		log.Errorf("update order_transaction_infos error with ids [%v] and params %v: %v", IDs, params, err)
		return err
	}

	return nil
}

func (u *UserServiceImpl) SetOrderTransactionInfoByTxHash(ctx context.Context, tx *gorm.DB, txHash string, newStatus int, height int) error {
	err := tx.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := u.orderTransactionInfoDAO.UpdateStatusByTxHash(ctx, tx, txHash, height, newStatus)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("fail to update the status by transaction hash: %v", err)
			return err
		}
		return nil
	})

	return err
}

//func (u *UserServiceImpl) SetOrderTransactionStatus(ctx context.Context, tx *gorm.DB, transactionIDs []uint64, newStatus int) error {
//	return u.orderTransactionInfoDAO.UpdateStatusByIDs(ctx, tx, transactionIDs, newStatus)
//}
//
//func (u *UserServiceImpl) SetOrderTransactionHeight(ctx context.Context, tx *gorm.DB, transactionID []uint64, height int) error {
//	return u.orderTransactionInfoDAO.UpdateHeightByIDs(ctx, tx, transactionID, height)
//}

func (u *UserServiceImpl) GetUserNum(ctx context.Context, tx *gorm.DB) (int64, error) {
	return u.userInfoDao.GetUserNum(ctx, tx)
}

func (u *UserServiceImpl) GetLatestRewardInterval(ctx context.Context, tx *gorm.DB) (int, error) {
	lastUserShareInfo, err := u.userShareInfoDao.GetLatest(ctx, tx)
	if err != nil {
		log.Errorf("Get latest user share info err: %v", err)
		return 0, err
	}
	if lastUserShareInfo == nil {
		return 0, nil
	}

	return int(lastUserShareInfo.EndHeight - lastUserShareInfo.StartHeight), nil
}

func (u *UserServiceImpl) GetLatestUserShareInfo(ctx context.Context, tx *gorm.DB) (*do.UserShareInfo, error) {
	return u.userShareInfoDao.GetLatest(ctx, tx)
}

func (u *UserServiceImpl) GetUsers(ctx context.Context, tx *gorm.DB, page int, num int, withAddress bool, positiveOrder bool) ([]*do.UserInfo, error) {
	return u.userInfoDao.GetUsers(ctx, tx, page, num, withAddress, positiveOrder)
}

func (u *UserServiceImpl) GetOrdersByUsername(ctx context.Context, tx *gorm.DB, username string, page int, num int, positiveOrder bool) ([]*do.OrderInfo, error) {
	userID, err := u.userInfoDao.GetIDByUsername(ctx, tx, username)
	if err != nil {
		return nil, err
	}

	return u.orderInfoDAO.GetByUserID(ctx, tx, userID, page, num, positiveOrder)
}

func (u *UserServiceImpl) GetOrderDetailsByUsername(ctx context.Context, tx *gorm.DB, username string,
	page int, num int, withAddress bool, positiveOrder bool) ([]*model.OrderDetails, error) {
	orders, err := u.GetOrdersByUsername(ctx, tx, username, page, num, positiveOrder)
	if err != nil {
		return nil, err
	}

	res := make([]*model.OrderDetails, 0)
	for _, order := range orders {
		tmp := &model.OrderDetails{
			ID:           order.ID,
			UserID:       order.UserID,
			Amount:       order.Amount,
			Status:       order.Status,
			Transactions: make([]*model.OrderTransactionDetails, 0),
			CreatedAt:    order.CreatedAt.Unix(),
			UpdatedAt:    order.UpdatedAt.Unix(),
		}
		transactions, _ := u.orderTransactionInfoDAO.GetByOrderID(ctx, tx, order.ID, withAddress)
		for _, transaction := range transactions {
			tmp2 := &model.OrderTransactionDetails{
				ID:              transaction.ID,
				TransactionHash: transaction.TransactionHash,
				Amount:          transaction.Amount,
				Status:          transaction.Status,
				Height:          transaction.Height,
				Address:         transaction.Address,
				CreatedAt:       transaction.CreatedAt.Unix(),
				UpdatedAt:       transaction.UpdatedAt.Unix(),
			}
			tmp.Transactions = append(tmp.Transactions, tmp2)
		}
		res = append(res, tmp)
	}
	return res, nil
}

func (u *UserServiceImpl) GetOrders(ctx context.Context, tx *gorm.DB, page int, num int, withAddress bool, positiveOrder bool) ([]*model.OrderDetails, error) {
	orders, err := u.orderInfoDAO.Get(ctx, tx, page, num, positiveOrder)
	if err != nil {
		return nil, err
	}

	res := make([]*model.OrderDetails, 0)
	for _, order := range orders {
		tmp := &model.OrderDetails{
			ID:           order.ID,
			UserID:       order.UserID,
			Amount:       order.Amount,
			Status:       order.Status,
			Transactions: make([]*model.OrderTransactionDetails, 0),
			CreatedAt:    order.CreatedAt.Unix(),
			UpdatedAt:    order.UpdatedAt.Unix(),
		}
		transactions, _ := u.orderTransactionInfoDAO.GetByOrderID(ctx, tx, order.ID, withAddress)
		for _, transaction := range transactions {
			tmp2 := &model.OrderTransactionDetails{
				ID:              transaction.ID,
				TransactionHash: transaction.TransactionHash,
				Amount:          transaction.Amount,
				Status:          transaction.Status,
				Height:          transaction.Height,
				Address:         transaction.Address,
				CreatedAt:       transaction.CreatedAt.Unix(),
				UpdatedAt:       transaction.UpdatedAt.Unix(),
			}
			tmp.Transactions = append(tmp.Transactions, tmp2)
		}
		res = append(res, tmp)
	}
	return res, nil
}

func (u *UserServiceImpl) GetOrdersByIDs(ctx context.Context, tx *gorm.DB, ids []uint64) ([]*do.OrderInfo, error) {
	return u.orderInfoDAO.GetByIDs(ctx, tx, ids)
}

func (u *UserServiceImpl) GetOrderCount(ctx context.Context, tx *gorm.DB) (int64, error) {
	return u.orderInfoDAO.GetOrderNum(ctx, tx)
}

func (u *UserServiceImpl) GetOrderCountByUsername(ctx context.Context, tx *gorm.DB, username string) (int64, error) {
	userID, err := u.userInfoDao.GetIDByUsername(ctx, tx, username)
	if err != nil {
		return 0, err
	}

	return u.orderInfoDAO.GetOrderNumByUserID(ctx, tx, userID)
}

func (u *UserServiceImpl) GetTransactionNum(ctx context.Context, tx *gorm.DB) (int64, error) {
	return u.orderTransactionInfoDAO.GetTransactionNum(ctx, tx)
}

func (u *UserServiceImpl) GetMinedBlockNum(ctx context.Context, tx *gorm.DB) (int64, error) {
	return u.minedBlockInfoDao.GetBlockNum(ctx, tx)
}

func (u *UserServiceImpl) GetAllocationCount(ctx context.Context, db *gorm.DB) (int64, error) {
	return u.allocationInfoDAO.GetNum(ctx, db)
}

func (u *UserServiceImpl) GetAllocation(ctx context.Context, db *gorm.DB, page int, num int, positiveOrder bool) ([]*do.AllocationInfo, error) {
	return u.allocationInfoDAO.Get(ctx, db, page, num, positiveOrder)
}

func (u *UserServiceImpl) GetAllocationCountByUsername(ctx context.Context, db *gorm.DB, username string) (int64, error) {
	return u.allocationInfoDAO.GetNumByUsername(ctx, db, username)
}

func (u *UserServiceImpl) GetAllocationByUsername(ctx context.Context, db *gorm.DB, username string, page int, num int, positiveOrder bool) ([]*do.AllocationInfo, error) {
	return u.allocationInfoDAO.GetByUsername(ctx, db, username, page, num, positiveOrder)
}

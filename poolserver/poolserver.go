package poolserver

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abesuite/abe-miningpool-server/chaincfg"
	"github.com/abesuite/abe-miningpool-server/chainclient"
	"github.com/abesuite/abe-miningpool-server/consensus/ethash"
	"github.com/abesuite/abe-miningpool-server/minermgr"
	"github.com/abesuite/abe-miningpool-server/model"
	"github.com/abesuite/abe-miningpool-server/ordermgr"
	"github.com/abesuite/abe-miningpool-server/pooljson"
	"github.com/abesuite/abe-miningpool-server/service"
	"github.com/abesuite/abe-miningpool-server/utils"
	"github.com/gorilla/websocket"

	"gorm.io/gorm"
)

const (
	// rpcAuthTimeoutSeconds is the number of seconds a connection to the
	// RPC server is allowed to stay open without authenticating before it
	// is closed.
	rpcAuthTimeoutSeconds = 10
)

// Commands that are available to a limited user
var rpcLimited = map[string]struct{}{
	"version":          {},
	"mining.hello":     {},
	"mining.noop":      {},
	"mining.subscribe": {},
	"mining.authorize": {},
	"mining.submit":    {},
	"mining.hashrate":  {},
	"mining.get_job":   {},

	// E9 miner
	"eth_submitLogin": {},
}

type commandHandler func(*PoolServer, interface{}, <-chan struct{}) (interface{}, error)

// rpcHandlers maps RPC command strings to appropriate handler functions.
// This is set by init because help references rpcHandlers and thus causes
// a dependency loop.
var rpcHandlers map[string]commandHandler
var rpcHandlersBeforeInit = map[string]commandHandler{
	"version": handleVersion,

	"getmetainfo": handleGetMetaInfo,
	"gethashrate": handleGetHashRate,

	"getusernum":            handleGetUserNum,
	"getuserinfo":           handleGetUserInfo,
	"getuserinfobyusername": handleGetUserInfo,
	"getuserinfos":          handleGetUserInfos,

	"getminernum":                 handleGetMinerNum,
	"getminers":                   handleGetMiners,
	"getminerinfo":                handleGetMinerInfo,
	"getminerinfobyremoteaddress": handleGetMinerInfo,
	"getminerinfos":               handleGetMinerInfos,

	"getorders":           handleGetOrders,
	"getordersbyusername": handleGetOrdersByUsername,

	"gettransactions": handleGetTransactions,

	"registeruser":       handleRegisterUser,
	"checkusernameexist": handleCheckUsernameExist,
	"checkpassword":      handleCheckPassword,
	"changepassword":     handleChangePassword,
	"resetpassword":      handleResetPassword,

	"getcurrentblocktemplate": handleGetCurrentBlockTemplate,

	"getminedblocks": handleGetMinedBlocks,

	"getusershareinfobyusername": handleGetUserShareInfoByUsername,
	"getusershareinfo":           handleGetUserShareInfo,

	"getallocationinfo":           handleGetAllocationInfo,
	"getallocationinfobyusername": handleGetAllocationInfoByUsername,

	"getagentblacklist":    handleGetAgentBlacklist,
	"getagentwhitelist":    handleGetAgentWhitelist,
	"getblacklist":         handleGetBlacklist,
	"getwhitelist":         handleGetWhitelist,
	"getadminblacklist":    handleGetAdminBlacklist,
	"getadminwhitelist":    handleGetAdminWhitelist,
	"addagentblacklist":    handleAddAgentBlacklist,
	"addagentwhitelist":    handleAddAgentWhitelist,
	"addblacklist":         handleAddBlacklist,
	"addwhitelist":         handleAddWhitelist,
	"addadminblacklist":    handleAddAdminBlacklist,
	"addadminwhitelist":    handleAddAdminWhitelist,
	"deleteagentblacklist": handleDeleteAgentBlacklist,
	"deleteagentwhitelist": handleDeleteAgentWhitelist,
	"deleteblacklist":      handleDeleteBlacklist,
	"deletewhitelist":      handleDeleteWhitelist,
	"deleteadminblacklist": handleDeleteAdminBlacklist,
	"deleleadminwhitelist": handleDeleteAdminWhitelist,

	"walletpassphrase": handleWalletPassphrase,

	"getstatus": handleGetStatus,

	"setrewardinterval":   handleSetRewardInterval,
	"setrewardmaturity":   handleSetRewardMaturity,
	"setmanagefeepercent": handleSetManageFeePercent,
}

// simpleAddr implements the net.Addr interface with two struct fields
type simpleAddr struct {
	net, addr string
}

// String returns the address.
//
// This is part of the net.Addr interface.
func (a simpleAddr) String() string {
	return a.addr
}

// Network returns the network.
//
// This is part of the net.Addr interface.
func (a simpleAddr) Network() string {
	return a.net
}

// Ensure simpleAddr implements the net.Addr interface.
var _ net.Addr = simpleAddr{}

// handleVersion implements the version command.
func handleVersion(s *PoolServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	result := map[string]pooljson.VersionResult{
		"abec": {
			VersionString: chaincfg.AbecBackendVersion,
		},
		"server": {
			VersionString: chaincfg.PoolBackendVersion,
		},
	}
	return result, nil
}

// handleGetMinerNum implements the getminernum command.
func handleGetMinerNum(s *PoolServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	num := s.minerManager.GetMinerNum()

	result := pooljson.GetMinerNumResult{
		Number: num,
	}

	return &result, nil
}

func handleGetMetaInfo(s *PoolServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	db := s.minerManager.Db
	userService := service.GetUserService()
	metaService := service.GetMetaInfoService()

	metaInfo, err := metaService.Get(context.Background(), db)
	if err != nil {
		log.Errorf("Handler GetMetaInfo fail: %v", err)
		return nil, pooljson.ErrRPCInternal
	}

	unallocated, err := userService.GetUnallocatedReward(context.Background(), db)

	jobTemplate := s.minerManager.GetJob()
	if jobTemplate == nil {
		return &pooljson.GetMetaInfoResult{
			Net:                  chaincfg.ActiveNetParams.Name,
			StartTime:            s.startTime,
			Balance:              metaInfo.Balance,
			AllocatedBalance:     metaInfo.AllocatedBalance,
			UnallocatedBalance:   unallocated,
			CurrentHeight:        -1,
			LastRewardHeight:     metaInfo.LastRewardHeight,
			RewardInterval:       s.rewardCfg.RewardInterval,
			ManageFeePercent:     s.rewardCfg.ManageFeePercent,
			RewardMaturity:       s.rewardCfg.RewardMaturity,
			ConfigChangeHeight:   s.rewardCfg.ConfigChangeHeight,
			NextRewardInterval:   s.rewardCfg.RewardInterval,
			NextManageFeePercent: s.rewardCfg.ManageFeePercent,
		}, nil
	}

	rewardInterval := s.rewardCfg.RewardInterval
	rewardMaturity := s.rewardCfg.RewardMaturity
	manageFeePercent := s.rewardCfg.ManageFeePercent
	configChangeHeight := s.rewardCfg.ConfigChangeHeight
	nextRewardInterval := s.rewardCfg.RewardInterval
	nextManageFeePercent := s.rewardCfg.ManageFeePercent
	if jobTemplate.Height < configChangeHeight {
		rewardInterval = s.rewardCfg.RewardIntervalPre
		manageFeePercent = s.rewardCfg.ManageFeePercentPre
	}

	result := pooljson.GetMetaInfoResult{
		Net:                  chaincfg.ActiveNetParams.Name,
		StartTime:            s.startTime,
		Balance:              metaInfo.Balance,
		AllocatedBalance:     metaInfo.AllocatedBalance,
		UnallocatedBalance:   unallocated,
		CurrentHeight:        jobTemplate.Height,
		LastRewardHeight:     metaInfo.LastRewardHeight,
		RewardInterval:       rewardInterval,
		ManageFeePercent:     manageFeePercent,
		RewardMaturity:       rewardMaturity,
		ConfigChangeHeight:   configChangeHeight,
		NextRewardInterval:   nextRewardInterval,
		NextManageFeePercent: nextManageFeePercent,
	}
	return &result, nil
}

func handleGetHashRate(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	res := s.minerManager.GetHashRate()
	return &pooljson.GetHashRateResult{
		HashRate: res,
	}, nil
}

func handleGetUserInfo(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.GetUserInfoCmd)
	if !ok {
		log.Debugf("Receive an invalid getuserinfo command")
		return nil, pooljson.ErrRPCInvalidParams
	}

	username := cmd.Username

	db := s.minerManager.Db
	userService := service.GetUserService()
	info, err := userService.GetUserByUsername(context.Background(), db, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pooljson.ErrUsernameNotFound
		}
		return nil, err
	}
	address := ""
	if cmd.WithAddress != nil && *cmd.WithAddress == true {
		address = info.Address
	}

	return &pooljson.GetUserInfoResult{
		ID:              info.ID,
		Username:        info.Username,
		Balance:         info.Balance,
		RedeemedBalance: info.RedeemedBalance,
		Password:        info.Password,
		Salt:            info.Salt,
		Address:         address,
		RegisteredAt:    info.CreatedAt.Unix(),
		UpdatedAt:       info.UpdatedAt.Unix(),
	}, nil
}

func handleGetUserInfos(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.GetUserInfosCmd)
	if !ok {
		log.Debugf("Receive an invalid getuserinfos command")
		return nil, pooljson.ErrRPCInvalidParams
	}

	db := s.minerManager.Db
	userService := service.GetUserService()
	withAddress := true
	if cmd.WithAddress == nil || *cmd.WithAddress == false {
		withAddress = false
	}
	positiveOrder := true
	if cmd.PositiveOrder == nil || *cmd.PositiveOrder == false {
		positiveOrder = false
	}
	infos, err := userService.GetUsers(context.Background(), db, cmd.Page, cmd.Num, withAddress, positiveOrder)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	num, err := userService.GetUserNum(context.Background(), db)
	if err != nil {
		return nil, err
	}

	users := make([]*pooljson.GetUserInfoResult, 0)
	for _, user := range infos {
		tmp := pooljson.GetUserInfoResult{
			ID:              user.ID,
			Username:        user.Username,
			Balance:         user.Balance,
			RedeemedBalance: user.RedeemedBalance,
			Password:        user.Password,
			Salt:            user.Salt,
			RegisteredAt:    user.CreatedAt.Unix(),
			UpdatedAt:       user.UpdatedAt.Unix(),
			Address:         user.Address,
		}
		users = append(users, &tmp)
	}

	res := pooljson.GetUserInfosResult{
		Total: num,
		Users: users,
	}

	return &res, nil
}

// handleCheckUsernameExist implements the checkusernameexist command.
func handleCheckUsernameExist(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.CheckUsernameExistCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	if utils.IsBlank(cmd.Username) {
		return nil, pooljson.ErrRPCInvalidParams
	}

	db := s.minerManager.Db
	userService := service.GetUserService()
	isExist, err := userService.UsernameExist(context.Background(), db, cmd.Username)
	if err != nil {
		log.Errorf(err.Error())
		return nil, pooljson.ErrRPCInternal
	}

	result := pooljson.CheckUsernameExistResult{
		IsExist: isExist,
	}

	return &result, nil
}

// handleRegisterUser implements the registeruser command.
func handleRegisterUser(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.RegisterUserCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	if utils.IsBlank(cmd.Username) || utils.IsBlank(cmd.Password) {
		return nil, pooljson.ErrRPCInvalidParams
	}

	if strings.TrimSpace(cmd.Username) == "admin" {
		return nil, pooljson.ErrRPCInvalidParams
	}

	db := s.minerManager.Db
	userService := service.GetUserService()
	_, err := userService.RegisterUser(context.Background(), db, cmd.Username, cmd.Password, cmd.Address, nil)
	if err != nil {
		log.Errorf(err.Error())
		return nil, err
	}

	result := pooljson.RegisterUserResult{
		Success: true,
	}

	return &result, nil
}

// handleChangePassword implements the changepassword command.
// This command is used by user to change the password,
// need to know the original password.
func handleChangePassword(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.ChangePasswordCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	if utils.IsBlank(cmd.Username) {
		return nil, pooljson.ErrRPCInvalidParams
	}

	db := s.minerManager.Db
	userService := service.GetUserService()
	err := userService.ChangePassword(context.Background(), db, cmd.Username, cmd.OldPassword, cmd.NewPassword)
	if err != nil {
		return nil, err
	}

	result := pooljson.ChangePasswordResult{
		Success: true,
	}

	return &result, nil
}

// handleResetPassword implements the resetpassword command.
// This command is used by admin to change the password,
// No need to know the original password.
func handleResetPassword(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.ResetPasswordCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	if utils.IsBlank(cmd.Username) {
		return nil, pooljson.ErrRPCInvalidParams
	}

	db := s.minerManager.Db
	userService := service.GetUserService()
	err := userService.ResetPassword(context.Background(), db, cmd.Username, cmd.NewPassword)
	if err != nil {
		return nil, err
	}

	result := pooljson.ResetPasswordResult{
		Success: true,
	}

	return &result, nil
}

// handleWalletPassphrase implements the walletpassphrase command.
// This command is used by admin to unlock the connected wallet,
// when set successfully, the coin would be allocated automatic.
func handleWalletPassphrase(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.WalletPassphraseCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	if cmd.Password == "" {
		return nil, pooljson.ErrRPCInvalidParams
	}

	result := pooljson.WalletPassphraseResult{
		Success: false,
	}

	if s.orderManager == nil {
		result.Detail = "fail to enable due to something error"
		return result, errors.New("internal error")
	}

	if !s.orderManager.EnableAutoFlag() {
		result.Detail = "unable to enable automatic allocating, please restart the pool"
		return result, nil
	}
	if !s.orderManager.SetWalletPrivPass(cmd.Password) {
		result.Detail = "unable to enable automatic allocating, please check the authority of passphrase"
		return result, nil
	}

	result.Success = true
	return &result, nil
}

func handleGetStatus(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	_, ok := icmd.(*pooljson.GetStatusCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	result := pooljson.GetStatusResult{}

	ctx := context.Background()
	db := s.minerManager.Db
	userService := service.GetUserService()
	num, err := userService.GetUserNum(ctx, db)
	if err != nil {
		return nil, err
	}
	result.UserNum = num

	miners := s.minerManager.GetMiners()
	result.MinerNum = len(miners)

	minedBlockNum, err := userService.GetMinedBlockNum(context.Background(), db)
	if err != nil {
		return nil, err
	}
	result.MinedBlockNum = minedBlockNum

	unfinishedOrders, err := userService.GetAllUnfinishedOrders(ctx, db)
	if err != nil {
		return nil, err
	}
	result.UnfinishedOrderNum = len(unfinishedOrders)

	if s.orderManager != nil {
		result.EnableAutoFlag = s.orderManager.EnableAutoFlag()
		result.AutoFlag = s.orderManager.AutoFlag()
	}

	return &result, nil
}

// handleGetUserNum implements the getusernum command.
func handleGetUserNum(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	_, ok := icmd.(*pooljson.GetUserNumCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	db := s.minerManager.Db
	userService := service.GetUserService()
	num, err := userService.GetUserNum(context.Background(), db)
	if err != nil {
		return nil, err
	}

	result := pooljson.GetUserNumResult{
		Number: num,
	}

	return result, nil
}

func handleGetMiners(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	_, ok := icmd.(*pooljson.GetMinersCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	miners := s.minerManager.GetMiners()
	return miners, nil
}

func handleGetMinerInfo(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.GetMinerInfoCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	minerInfo := s.minerManager.GetActiveMiner(cmd.RemoteAddress)
	if minerInfo == nil {
		return nil, pooljson.ErrNotFound
	}

	username := make([]string, 0)
	for name, _ := range minerInfo.Username {
		username = append(username, name)
	}

	return &pooljson.GetMinerInfoResult{
		UserAgent:      minerInfo.UserAgent,
		Protocol:       minerInfo.Protocol,
		ConnectionType: minerInfo.ConnectionType,
		Address:        minerInfo.Address,
		SessionID:      minerInfo.SessionID,
		Username:       username,
		ExtraNonce1:    minerInfo.ExtraNonce1,
		Difficulty:     minerInfo.Difficulty,
		ShareCount:     minerInfo.ShareCount,
		HashRateUpload: minerInfo.HashRateUpload,
		ArriveAt:       minerInfo.HelloAt.Unix(),
		LastActiveAt:   minerInfo.LastActiveAt.Unix(),
		ErrorCounts:    minerInfo.ErrorCounts,
		Rejected:       minerInfo.Rejected,
		Accepted:       minerInfo.Accepted,
	}, nil
}

func handleGetMinerInfos(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	minerInfos := s.minerManager.GetAllMiners()
	res := make([]*pooljson.GetMinerInfoResult, 0)

	for _, minerInfo := range minerInfos {
		username := make([]string, 0)
		for name, _ := range minerInfo.Username {
			username = append(username, name)
		}
		tmp := &pooljson.GetMinerInfoResult{
			UserAgent:         minerInfo.UserAgent,
			Protocol:          minerInfo.Protocol,
			ConnectionType:    minerInfo.ConnectionType,
			Address:           minerInfo.Address,
			SessionID:         minerInfo.SessionID,
			Username:          username,
			ExtraNonce1:       minerInfo.ExtraNonce1,
			Difficulty:        minerInfo.Difficulty,
			ShareCount:        minerInfo.ShareCount,
			HashRateUpload:    minerInfo.HashRateUpload,
			HashRateEstimated: int64(minerInfo.HashRateEstimated),
			ArriveAt:          minerInfo.HelloAt.Unix(),
			LastActiveAt:      minerInfo.LastActiveAt.Unix(),
			ErrorCounts:       minerInfo.ErrorCounts,
			Rejected:          minerInfo.Rejected,
			Accepted:          minerInfo.Accepted,
		}
		res = append(res, tmp)
	}

	return res, nil
}

func handleGetOrders(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.GetOrdersCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	db := s.minerManager.Db
	userService := service.GetUserService()

	total, err := userService.GetOrderCount(context.Background(), db)
	if err != nil {
		return nil, err
	}

	withAddress := true
	if cmd.WithAddress == nil || *cmd.WithAddress == false {
		withAddress = false
	}
	positiveOrder := true
	if cmd.PositiveOrder == nil || *cmd.PositiveOrder == false {
		positiveOrder = false
	}
	res, err := userService.GetOrders(context.Background(), db, cmd.Page, cmd.Num, withAddress, positiveOrder)
	if err != nil {
		return nil, err
	}
	return &pooljson.GetOrdersResult{
		Total:  total,
		Orders: res,
	}, nil
}

func handleGetOrdersByUsername(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.GetOrdersByUsernameCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	db := s.minerManager.Db
	userService := service.GetUserService()
	withAddress := true
	if cmd.WithAddress == nil || *cmd.WithAddress == false {
		withAddress = false
	}
	positiveOrder := true
	if cmd.PositiveOrder == nil || *cmd.PositiveOrder == false {
		positiveOrder = false
	}
	total, err := userService.GetOrderCountByUsername(context.Background(), db, cmd.Username)
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return &pooljson.GetOrdersResult{
			Total:  0,
			Orders: make([]*model.OrderDetails, 0),
		}, nil
	}
	res, err := userService.GetOrderDetailsByUsername(context.Background(), db, cmd.Username, cmd.Page, cmd.Num, withAddress, positiveOrder)
	if err != nil {
		return nil, err
	}
	return &pooljson.GetOrdersResult{
		Total:  total,
		Orders: res,
	}, nil
}

func handleGetTransactions(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.GetTransactionsCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	db := s.minerManager.Db
	userService := service.GetUserService()

	total, err := userService.GetTransactionNum(context.Background(), db)
	if err != nil {
		return nil, err
	}

	withAddress := true
	if cmd.WithAddress == nil || *cmd.WithAddress == false {
		withAddress = false
	}
	positiveOrder := true
	if cmd.PositiveOrder == nil || *cmd.PositiveOrder == false {
		positiveOrder = false
	}

	res, err := userService.GetTransactions(context.Background(), db, cmd.Page, cmd.Num, withAddress, positiveOrder)
	if err != nil {
		return nil, err
	}
	return &pooljson.GetTransactionsResult{
		Total:        total,
		Transactions: res,
	}, nil
}

// handleCheckPassword implements the checkpassword command.
func handleCheckPassword(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.CheckPasswordCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	db := s.minerManager.Db
	userService := service.GetUserService()
	res, _, err := userService.Login(context.Background(), db, cmd.Username, cmd.Password)
	if err != nil {
		return nil, err
	}
	return &pooljson.CheckPasswordResult{
		Result: res,
	}, nil
}

func handleGetCurrentBlockTemplate(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	jobTemplate := s.minerManager.GetJob()
	if jobTemplate == nil {
		return nil, nil
	}
	txHashes := make([]string, 0)
	for _, txHash := range jobTemplate.TxHashes {
		txHashes = append(txHashes, txHash.String())
	}

	return &pooljson.GetCurrentBlockTemplateResult{
		JobId:        jobTemplate.JobId,
		PreviousHash: jobTemplate.PreviousHash,
		TxHashes:     txHashes,
		CurrTime:     jobTemplate.CurrTime.Unix(),
		ContentHash:  jobTemplate.ContentHash,
		Height:       jobTemplate.Height,
		Reward:       jobTemplate.Reward,
	}, nil
}

func handleGetMinedBlocks(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.GetMinedBlocksCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	db := s.minerManager.Db
	userService := service.GetUserService()
	positiveOrder := false
	if cmd.PositiveOrder != nil && *cmd.PositiveOrder == true {
		positiveOrder = true
	}

	total, err := userService.GetMinedBlockNum(context.Background(), db)
	if err != nil {
		return nil, err
	}

	blocks, err := userService.GetMinedBlocks(context.Background(), db, cmd.Page, cmd.Num, positiveOrder)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &pooljson.GetMinedBlocksResult{
				Total:       total,
				MinedBlocks: make([]*pooljson.MinedBlock, 0),
			}, nil
		}
		return nil, err
	}

	minedBlocks := make([]*pooljson.MinedBlock, 0)
	for _, block := range blocks {
		tmp := pooljson.MinedBlock{
			ID:           block.ID,
			Username:     block.Username,
			Height:       block.Height,
			BlockHash:    block.BlockHash,
			SealHash:     block.SealHash,
			Reward:       block.Reward,
			Disconnected: block.Disconnected,
			Connected:    block.Connected,
			Rejected:     block.Rejected,
			Allocated:    block.Allocated,
			Info:         block.Info,
			CreatedAt:    block.CreatedAt.Unix(),
			UpdatedAt:    block.UpdatedAt.Unix(),
		}
		minedBlocks = append(minedBlocks, &tmp)
	}

	return &pooljson.GetMinedBlocksResult{
		Total:       total,
		MinedBlocks: minedBlocks,
	}, nil
}

func handleGetUserShareInfoByUsername(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.GetUserShareInfoByUsernameCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	db := s.minerManager.Db
	userService := service.GetUserService()

	if utils.IsBlank(cmd.Username) {
		return nil, pooljson.ErrNotFound
	}

	res := make([]*pooljson.UserShareInfoResult, 0)
	total, err := userService.GetShareCountByUsername(context.Background(), db, cmd.Username)
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return &pooljson.GetUserShareInfoByUsernameResult{
			Total: total,
			Share: res,
		}, nil
	}

	positiveOrder := false
	if cmd.PositiveOrder != nil && *cmd.PositiveOrder == true {
		positiveOrder = true
	}
	shareInfos, err := userService.GetShareByUsername(context.Background(), db, cmd.Username, cmd.Page, cmd.Num, positiveOrder)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return res, nil
		}
		return nil, err
	}

	for _, shareInfo := range shareInfos {
		tmp := pooljson.UserShareInfoResult{
			ID:          shareInfo.ID,
			Username:    shareInfo.Username,
			ShareCount:  shareInfo.ShareCount,
			StartHeight: shareInfo.StartHeight,
			EndHeight:   shareInfo.EndHeight,
			Allocated:   shareInfo.Allocated,
			CreatedAt:   shareInfo.CreatedAt.Unix(),
			UpdatedAt:   shareInfo.UpdatedAt.Unix(),
		}
		res = append(res, &tmp)
	}
	return &pooljson.GetUserShareInfoByUsernameResult{
		Total: total,
		Share: res,
	}, nil
}

func handleGetUserShareInfo(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.GetUserShareInfoCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	db := s.minerManager.Db
	userService := service.GetUserService()

	res := make([]*pooljson.UserShareInfoResult, 0)
	total, err := userService.GetShareCount(context.Background(), db)
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return &pooljson.GetUserShareInfoResult{
			Total: total,
			Share: res,
		}, nil
	}

	positiveOrder := false
	if cmd.PositiveOrder != nil && *cmd.PositiveOrder == true {
		positiveOrder = true
	}
	shareInfos, err := userService.GetShare(context.Background(), db, cmd.Page, cmd.Num, positiveOrder)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return res, nil
		}
		return nil, err
	}

	for _, shareInfo := range shareInfos {
		tmp := pooljson.UserShareInfoResult{
			ID:          shareInfo.ID,
			Username:    shareInfo.Username,
			ShareCount:  shareInfo.ShareCount,
			StartHeight: shareInfo.StartHeight,
			EndHeight:   shareInfo.EndHeight,
			Allocated:   shareInfo.Allocated,
			CreatedAt:   shareInfo.CreatedAt.Unix(),
			UpdatedAt:   shareInfo.UpdatedAt.Unix(),
		}
		res = append(res, &tmp)
	}
	return &pooljson.GetUserShareInfoByUsernameResult{
		Total: total,
		Share: res,
	}, nil
}

func handleGetAllocationInfoByUsername(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.GetAllocationInfoByUsernameCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	db := s.minerManager.Db
	userService := service.GetUserService()

	if utils.IsBlank(cmd.Username) {
		return nil, pooljson.ErrNotFound
	}

	res := make([]*pooljson.AllocationInfoResult, 0)
	total, err := userService.GetAllocationCountByUsername(context.Background(), db, cmd.Username)
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return &pooljson.GetAllocationInfoResult{
			Total: total,
			Share: res,
		}, nil
	}

	positiveOrder := false
	if cmd.PositiveOrder != nil && *cmd.PositiveOrder == true {
		positiveOrder = true
	}
	allocationInfos, err := userService.GetAllocationByUsername(context.Background(), db, cmd.Username, cmd.Page, cmd.Num, positiveOrder)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return res, nil
		}
		return nil, err
	}

	for _, allocationInfo := range allocationInfos {
		tmp := pooljson.AllocationInfoResult{
			ID:          allocationInfo.ID,
			Username:    allocationInfo.Username,
			ShareCount:  allocationInfo.ShareCount,
			StartHeight: allocationInfo.StartHeight,
			EndHeight:   allocationInfo.EndHeight,
			Balance:     allocationInfo.Balance,
			CreatedAt:   allocationInfo.CreatedAt.Unix(),
			UpdatedAt:   allocationInfo.UpdatedAt.Unix(),
		}
		res = append(res, &tmp)
	}
	return &pooljson.GetAllocationInfoResult{
		Total: total,
		Share: res,
	}, nil
}

func handleGetAllocationInfo(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.GetAllocationInfoCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	db := s.minerManager.Db
	userService := service.GetUserService()

	res := make([]*pooljson.AllocationInfoResult, 0)
	total, err := userService.GetAllocationCount(context.Background(), db)
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return &pooljson.GetAllocationInfoResult{
			Total: total,
			Share: res,
		}, nil
	}

	positiveOrder := false
	if cmd.PositiveOrder != nil && *cmd.PositiveOrder == true {
		positiveOrder = true
	}
	allocationInfos, err := userService.GetAllocation(context.Background(), db, cmd.Page, cmd.Num, positiveOrder)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return res, nil
		}
		return nil, err
	}

	for _, allocationInfo := range allocationInfos {
		tmp := pooljson.AllocationInfoResult{
			ID:          allocationInfo.ID,
			Username:    allocationInfo.Username,
			ShareCount:  allocationInfo.ShareCount,
			StartHeight: allocationInfo.StartHeight,
			EndHeight:   allocationInfo.EndHeight,
			Balance:     allocationInfo.Balance,
			CreatedAt:   allocationInfo.CreatedAt.Unix(),
			UpdatedAt:   allocationInfo.UpdatedAt.Unix(),
		}
		res = append(res, &tmp)
	}
	return &pooljson.GetAllocationInfoResult{
		Total: total,
		Share: res,
	}, nil
}

func handleSetRewardInterval(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.SetRewardIntervalCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	if cmd.RewardInterval <= 0 {
		return nil, pooljson.ErrInvalidRequestParams
	}
	if int(chaincfg.ActiveNetParams.EthashEpochLength)%cmd.RewardInterval != 0 {
		return nil, pooljson.ErrInvalidRequestParams
	}

	currentJobTemplate := s.minerManager.GetJob()
	if currentJobTemplate == nil {
		return nil, pooljson.ErrInternal
	}

	currentHeight := currentJobTemplate.Height
	if currentHeight < s.rewardCfg.ConfigChangeHeight {
		s.rewardCfg.RewardInterval = cmd.RewardInterval
	} else {
		currentEpoch := ethash.CalculateEpoch(int32(currentHeight))
		nextEpochStartHeight := ethash.CalculateEpochStartHeight(currentEpoch + 1)
		s.rewardCfg.ConfigChangeHeight = int64(nextEpochStartHeight)
		s.rewardCfg.RewardIntervalPre = s.rewardCfg.RewardInterval
		s.rewardCfg.ManageFeePercentPre = s.rewardCfg.ManageFeePercent
		s.rewardCfg.RewardInterval = cmd.RewardInterval
	}

	return &pooljson.CommonResult{
		Success: true,
	}, nil
}

func handleSetRewardMaturity(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.SetRewardMaturityCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	if cmd.RewardMaturity < 0 {
		return nil, pooljson.ErrInvalidRequestParams
	}
	s.rewardCfg.RewardMaturity = cmd.RewardMaturity
	return &pooljson.CommonResult{
		Success: true,
	}, nil
}

func handleSetManageFeePercent(s *PoolServer, icmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	cmd, ok := icmd.(*pooljson.SetManageFeePercentCmd)
	if !ok {
		return nil, pooljson.ErrRPCInternal
	}

	if cmd.ManageFeePercent < 0 || cmd.ManageFeePercent > 100 {
		return nil, pooljson.ErrInvalidRequestParams
	}
	currentJobTemplate := s.minerManager.GetJob()
	if currentJobTemplate == nil {
		return nil, pooljson.ErrInternal
	}

	currentHeight := currentJobTemplate.Height
	if currentHeight < s.rewardCfg.ConfigChangeHeight {
		s.rewardCfg.ManageFeePercent = cmd.ManageFeePercent
	} else {
		currentEpoch := ethash.CalculateEpoch(int32(currentHeight))
		nextEpochStartHeight := ethash.CalculateEpochStartHeight(currentEpoch + 1)
		s.rewardCfg.ConfigChangeHeight = int64(nextEpochStartHeight)
		s.rewardCfg.RewardIntervalPre = s.rewardCfg.RewardInterval
		s.rewardCfg.ManageFeePercentPre = s.rewardCfg.ManageFeePercent
		s.rewardCfg.ManageFeePercent = cmd.ManageFeePercent
	}

	return &pooljson.CommonResult{
		Success: true,
	}, nil
}

// internalRPCError is a convenience function to convert an internal error to
// an RPC error with the appropriate code set.  It also logs the error to the
// RPC server subsystem since internal errors really should not occur.  The
// context parameter is only used in the log message and may be empty if it's
// not needed.
func internalRPCError(errStr, context string) *pooljson.RPCError {
	logStr := errStr
	if context != "" {
		logStr = context + ": " + errStr
	}
	log.Error(logStr)
	return pooljson.NewRPCError(pooljson.ErrRPCInternal.Code, errStr)
}

// PoolServer provides a concurrent safe RPC server to a chain server.
type PoolServer struct {
	started                int32
	startTime              int64
	shutdown               int32
	cfg                    ConfigPoolServer
	commonCfg              *CommonConfig
	rewardCfg              *model.RewardConfig
	authsha                [sha256.Size]byte
	limitauthsha           [sha256.Size]byte
	ntfnMgr                AbstractNtfnManager
	numClients             int32
	statusLines            map[int]string
	statusLock             sync.RWMutex
	wg                     sync.WaitGroup
	requestProcessShutdown chan struct{}
	quit                   chan int

	minerManager    *minermgr.MinerManager
	chainClient     *chainclient.RPCClient
	tcpSocketServer *TCPSocketServer

	orderManager *ordermgr.OrderManager

	ethash          *ethash.Ethash
	abecBackendNode string
}

// ConfigPoolServer is a descriptor containing the RPC server configuration.
type ConfigPoolServer struct {
	DisableTLS bool
	// ListenersString an array that contains ip address and port for generating
	// listeners later
	ListenersString []string

	// Listeners defines a slice of listeners for which the RPC server will
	// take ownership of and accept connections.  Since the RPC server takes
	// ownership of these listeners, they will be closed when the RPC server
	// is stopped.
	Listeners []net.Listener

	// StartupTime is the unix timestamp for when the server that is hosting
	// the RPC server started.
	StartupTime int64

	RPCUser              string
	RPCPass              string
	RPCLimitUser         string
	RPCLimitPass         string
	RPCMaxClients        int
	RPCMaxWebsockets     int
	RPCMaxConcurrentReqs int
	RPCKey               string
	RPCCert              string
	ExternalIPs          []string

	DisableConnectToAbec bool
	UseWallet            bool
	MaxErrors            int
	DetailedShareInfo    bool
}

type CommonConfig struct {
	AgentBlacklist []string
	AgentWhitelist []string
	Blacklist      []*net.IPNet
	Whitelist      []*net.IPNet
	AdminBlacklist []*net.IPNet
	AdminWhitelist []*net.IPNet
}

func (svr *PoolServer) SetMinerManager(mgr *minermgr.MinerManager) {
	svr.minerManager = mgr
}
func (svr *PoolServer) SetOrderManager(mgr *ordermgr.OrderManager) {
	svr.orderManager = mgr
}

func (svr *PoolServer) SetChainClient(cli *chainclient.RPCClient) {
	svr.chainClient = cli
}

func (svr *PoolServer) SetEthash(ethash *ethash.Ethash) {
	svr.ethash = ethash
}

func (svr *PoolServer) SetTCPSocketServer(server *TCPSocketServer) {
	svr.tcpSocketServer = server
}

func (svr *PoolServer) SetCommonConfig(cfg *CommonConfig) {
	svr.commonCfg = cfg
}

func (svr *PoolServer) SetRewardConfig(cfg *model.RewardConfig) {
	svr.rewardCfg = cfg
}

func (svr *PoolServer) GetAgentBlacklist() []string {
	return svr.commonCfg.AgentBlacklist
}

func (svr *PoolServer) GetAgentWhitelist() []string {
	return svr.commonCfg.AgentWhitelist
}

func (svr *PoolServer) GetBlacklist() []*net.IPNet {
	return svr.commonCfg.Blacklist
}

func (svr *PoolServer) GetWhitelist() []*net.IPNet {
	return svr.commonCfg.Whitelist
}

func (svr *PoolServer) GetAdminBlacklist() []*net.IPNet {
	return svr.commonCfg.AdminBlacklist
}

func (svr *PoolServer) GetAdminWhitelist() []*net.IPNet {
	return svr.commonCfg.AdminWhitelist
}

func (svr *PoolServer) AddAgentBlacklist(agentAdd string) {
	for _, agent := range svr.commonCfg.AgentBlacklist {
		if agent == agentAdd {
			return
		}
	}
	svr.commonCfg.AgentBlacklist = append(svr.commonCfg.AgentBlacklist, agentAdd)
}

func (svr *PoolServer) DeleteAgentBlacklist(agentDelete string) {
	newAgentBlacklist := make([]string, 0)
	for _, agent := range svr.commonCfg.AgentBlacklist {
		if agent != agentDelete {
			newAgentBlacklist = append(newAgentBlacklist, agent)
		}
	}
	svr.commonCfg.AgentBlacklist = newAgentBlacklist
}

func (svr *PoolServer) AddAgentWhitelist(agentAdd string) {
	for _, agent := range svr.commonCfg.AgentWhitelist {
		if agent == agentAdd {
			return
		}
	}
	svr.commonCfg.AgentWhitelist = append(svr.commonCfg.AgentWhitelist, agentAdd)
}

func (svr *PoolServer) DeleteAgentWhitelist(agentDelete string) {
	newAgentWhitelist := make([]string, 0)
	for _, agent := range svr.commonCfg.AgentWhitelist {
		if agent != agentDelete {
			newAgentWhitelist = append(newAgentWhitelist, agent)
		}
	}
	svr.commonCfg.AgentWhitelist = newAgentWhitelist
}

func (svr *PoolServer) AddBlacklist(ipAdd *net.IPNet) {
	for _, ip := range svr.commonCfg.Blacklist {
		if ipAdd.String() == ip.String() {
			return
		}
	}
	svr.commonCfg.Blacklist = append(svr.commonCfg.Blacklist, ipAdd)
}

func (svr *PoolServer) DeleteBlacklist(ipAdd *net.IPNet) {
	newBlacklist := make([]*net.IPNet, 0)
	for _, ip := range svr.commonCfg.Blacklist {
		if ipAdd.String() != ip.String() {
			newBlacklist = append(newBlacklist, ip)
		}
	}
	svr.commonCfg.Blacklist = newBlacklist
}

func (svr *PoolServer) AddWhitelist(ipAdd *net.IPNet) {
	for _, ip := range svr.commonCfg.Whitelist {
		if ipAdd.String() == ip.String() {
			return
		}
	}
	svr.commonCfg.Whitelist = append(svr.commonCfg.Whitelist, ipAdd)
}

func (svr *PoolServer) DeleteWhitelist(ipAdd *net.IPNet) {
	newWhitelist := make([]*net.IPNet, 0)
	for _, ip := range svr.commonCfg.Whitelist {
		if ipAdd.String() != ip.String() {
			newWhitelist = append(newWhitelist, ip)
		}
	}
	svr.commonCfg.Whitelist = newWhitelist
}

func (svr *PoolServer) AddAdminBlacklist(ipAdd *net.IPNet) {
	for _, ip := range svr.commonCfg.AdminBlacklist {
		if ipAdd.String() == ip.String() {
			return
		}
	}
	svr.commonCfg.AdminBlacklist = append(svr.commonCfg.AdminBlacklist, ipAdd)
}

func (svr *PoolServer) DeleteAdminBlacklist(ipAdd *net.IPNet) {
	newBlacklist := make([]*net.IPNet, 0)
	for _, ip := range svr.commonCfg.AdminBlacklist {
		if ipAdd.String() != ip.String() {
			newBlacklist = append(newBlacklist, ip)
		}
	}
	svr.commonCfg.AdminBlacklist = newBlacklist
}

func (svr *PoolServer) AddAdminWhitelist(ipAdd *net.IPNet) {
	for _, ip := range svr.commonCfg.AdminWhitelist {
		if ipAdd.String() == ip.String() {
			return
		}
	}
	svr.commonCfg.AdminWhitelist = append(svr.commonCfg.AdminWhitelist, ipAdd)
}

func (svr *PoolServer) DeleteAdminWhitelist(ipAdd *net.IPNet) {
	newWhitelist := make([]*net.IPNet, 0)
	for _, ip := range svr.commonCfg.AdminWhitelist {
		if ipAdd.String() != ip.String() {
			newWhitelist = append(newWhitelist, ip)
		}
	}
	svr.commonCfg.AdminWhitelist = newWhitelist
}

// filesExists reports whether the named file or directory exists.
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// genCertPair generates a key/cert pair to the paths provided.
func genCertPair(certFile string, keyFile string, externalIPs []string) error {
	log.Infof("Generating TLS certificates of mining pool...")

	org := "abec mining pool autogenerated cert"
	validUntil := time.Now().Add(10 * 365 * 24 * time.Hour)

	extraHosts := externalIPs
	serverIP := ""
	fmt.Print("Please enter the ip address of your server (press enter if you just test on local area network): ")
	_, err := fmt.Scanln(&serverIP)
	if serverIP != "" {
		extraHosts = append(extraHosts, serverIP)
	}

	cert, key, err := utils.NewTLSCertPair(org, validUntil, extraHosts)
	if err != nil {
		return err
	}

	// Write cert and key files.
	if err = ioutil.WriteFile(certFile, cert, 0666); err != nil {
		return err
	}
	if err = ioutil.WriteFile(keyFile, key, 0600); err != nil {
		os.Remove(certFile)
		return err
	}

	log.Infof("Done generating TLS certificates")
	return nil
}

// parseListeners determines whether each listen address is IPv4 and IPv6 and
// returns a slice of appropriate net.Addrs to listen on with TCP. It also
// properly detects addresses which apply to "all interfaces" and adds the
// address as both IPv4 and IPv6.
func parseListeners(addrs []string) ([]net.Addr, error) {
	netAddrs := make([]net.Addr, 0, len(addrs)*2)
	for _, addr := range addrs {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			// Shouldn't happen due to already being normalized.
			return nil, err
		}

		// Empty host or host of * on plan9 is both IPv4 and IPv6.
		if host == "" || (host == "*" && runtime.GOOS == "plan9") {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp4", addr: addr})
			netAddrs = append(netAddrs, simpleAddr{net: "tcp6", addr: addr})
			continue
		}

		// Strip IPv6 zone id if present since net.ParseIP does not
		// handle it.
		zoneIndex := strings.LastIndex(host, "%")
		if zoneIndex > 0 {
			host = host[:zoneIndex]
		}

		// Parse the IP.
		ip := net.ParseIP(host)
		if ip == nil {
			return nil, fmt.Errorf("'%s' is not a valid IP address", host)
		}

		// To4 returns nil when the IP is not an IPv4 address, so use
		// this determine the address type.
		if ip.To4() == nil {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp6", addr: addr})
		} else {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp4", addr: addr})
		}
	}
	return netAddrs, nil
}

// setupRPCListeners returns a slice of listeners that are configured for use
// with the RPC server depending on the configuration settings for listen
// addresses and TLS.
func setupRPCListeners(listenersString []string, RPCKey string, RPCCert string, externalIPs []string,
	disableTLS bool) ([]net.Listener, error) {
	listenFunc := net.Listen
	// Check the TLS cert and key file
	if !disableTLS {
		if !fileExists(RPCKey) && !fileExists(RPCCert) {
			return nil, errors.New("cannot find pool cert and key")
		}

		keypair, err := tls.LoadX509KeyPair(RPCCert, RPCKey)
		if err != nil {
			return nil, err
		}

		tlsConfig := tls.Config{
			Certificates: []tls.Certificate{keypair},
			MinVersion:   tls.VersionTLS12,
		}

		// Change the standard net.Listen function to the tls one.
		listenFunc = func(net string, laddr string) (net.Listener, error) {
			return tls.Listen(net, laddr, &tlsConfig)
		}
	}

	netAddrs, err := parseListeners(listenersString)
	if err != nil {
		return nil, err
	}

	listeners := make([]net.Listener, 0, len(netAddrs))
	for _, addr := range netAddrs {
		listener, err := listenFunc(addr.Network(), addr.String())
		if err != nil {
			log.Warnf("Can't listen on %s: %v", addr, err)
			continue
		}
		listeners = append(listeners, listener)
	}

	return listeners, nil
}

// NewPoolServer returns a new instance of the PoolServer struct.
func NewPoolServer(config *ConfigPoolServer) (*PoolServer, error) {
	rpcListeners, err := setupRPCListeners(config.ListenersString, config.RPCKey, config.RPCCert, config.ExternalIPs,
		config.DisableTLS)
	if err != nil {
		return nil, err
	}
	if len(rpcListeners) == 0 {
		return nil, errors.New("Pool RPCS: No valid listen address")
	}
	config.Listeners = rpcListeners
	rpc := PoolServer{
		startTime:              time.Now().Unix(),
		cfg:                    *config,
		statusLines:            make(map[int]string),
		requestProcessShutdown: make(chan struct{}),
		quit:                   make(chan int),
		abecBackendNode:        "",
	}
	if config.RPCUser != "" && config.RPCPass != "" {
		login := config.RPCUser + ":" + config.RPCPass
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
		rpc.authsha = sha256.Sum256([]byte(auth))
	}
	if config.RPCLimitUser != "" && config.RPCLimitPass != "" {
		login := config.RPCLimitUser + ":" + config.RPCLimitPass
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
		rpc.limitauthsha = sha256.Sum256([]byte(auth))
	}
	rpc.ntfnMgr = newWsNotificationManager(&rpc)

	return &rpc, nil
}

// jsonRPCRead handles reading and responding to RPC messages.
func (svr *PoolServer) jsonRPCRead(w http.ResponseWriter, r *http.Request, isAdmin bool) {
	if atomic.LoadInt32(&svr.shutdown) != 0 {
		return
	}

	// Read and close the JSON-RPC request body from the caller.
	body, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		errCode := http.StatusBadRequest
		http.Error(w, fmt.Sprintf("%d error reading JSON message: %v",
			errCode, err), errCode)
		return
	}

	// Unfortunately, the http server doesn't provide the ability to
	// change the read deadline for the new connection and having one breaks
	// long polling. However, not having a read deadline on the initial
	// connection would mean clients can connect and idle forever. Thus,
	// hijack the connection from the HTTP server, clear the read deadline,
	// and handle writing the response manually.
	hj, ok := w.(http.Hijacker)
	if !ok {
		errMsg := "webserver doesn't support hijacking"
		log.Warnf(errMsg)
		errCode := http.StatusInternalServerError
		http.Error(w, strconv.Itoa(errCode)+" "+errMsg, errCode)
		return
	}
	conn, buf, err := hj.Hijack()
	if err != nil {
		log.Warnf("Failed to hijack HTTP connection: %v", err)
		errCode := http.StatusInternalServerError
		http.Error(w, strconv.Itoa(errCode)+" "+err.Error(), errCode)
		return
	}
	defer conn.Close()
	defer buf.Flush()
	conn.SetReadDeadline(timeZeroVal)

	// Attempt to parse the raw body into a JSON-RPC request.
	var responseID interface{}
	var jsonErr error
	var result interface{}
	var request pooljson.Request
	if err := json.Unmarshal(body, &request); err != nil {
		jsonErr = &pooljson.RPCError{
			Code:    pooljson.ErrRPCParse.Code,
			Message: "Failed to parse request: " + err.Error(),
		}
	}
	if jsonErr == nil {

		if request.ID == nil {
			return
		}

		// The parse was at least successful enough to have an ID so
		// set it for the response.
		responseID = request.ID

		// Setup a close notifier.  Since the connection is hijacked,
		// the CloseNotifer on the ResponseWriter is not available.
		closeChan := make(chan struct{}, 1)
		go func() {
			_, err := conn.Read(make([]byte, 1))
			if err != nil {
				close(closeChan)
			}
		}()

		// Check if the user is limited and set error if method unauthorized
		if !isAdmin {
			if _, ok := rpcLimited[request.Method]; !ok {
				jsonErr = &pooljson.RPCError{
					Code:    pooljson.ErrRPCInvalidParams.Code,
					Message: "limited user not authorized for this method",
				}
			}
		}

		if jsonErr == nil {
			// Attempt to parse the JSON-RPC request into a known concrete
			// command.
			parsedCmd := parseCmd(&request)
			if parsedCmd.err != nil {
				jsonErr = parsedCmd.err
			} else {
				result, jsonErr = svr.standardCmdResult(parsedCmd, closeChan)
			}
		}
	}

	if result == nil && jsonErr == nil {
		jsonErr = pooljson.ErrRPCInternal
	}
	// Marshal the response.
	msg, err := createMarshalledReply(responseID, result, jsonErr)
	if err != nil {
		log.Errorf("Failed to marshal reply: %v", err)
		return
	}

	// Write the response.
	err = svr.writeHTTPResponseHeaders(r, w.Header(), http.StatusOK, buf)
	if err != nil {
		log.Error(err)
		return
	}
	if _, err := buf.Write(msg); err != nil {
		log.Errorf("Failed to write marshalled reply: %v", err)
	}

	// Terminate with newline to maintain compatibility.
	if err := buf.WriteByte('\n'); err != nil {
		log.Errorf("Failed to append terminating newline to reply: %v", err)
	}
}

// writeHTTPResponseHeaders writes the necessary response headers prior to
// writing an HTTP body given a request to use for protocol negotiation, headers
// to write, a status code, and a writer.
func (svr *PoolServer) writeHTTPResponseHeaders(req *http.Request, headers http.Header, code int, w io.Writer) error {
	_, err := io.WriteString(w, svr.httpStatusLine(req, code))
	if err != nil {
		return err
	}

	err = headers.Write(w)
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, "\r\n")
	return err
}

// httpStatusLine returns a response Status-Line (RFC 2616 Section 6.1)
// for the given request and response status code.  This function was lifted and
// adapted from the standard library HTTP server code since it's not exported.
func (svr *PoolServer) httpStatusLine(req *http.Request, code int) string {
	// Fast path:
	key := code
	proto11 := req.ProtoAtLeast(1, 1)
	if !proto11 {
		key = -key
	}
	svr.statusLock.RLock()
	line, ok := svr.statusLines[key]
	svr.statusLock.RUnlock()
	if ok {
		return line
	}

	// Slow path:
	proto := "HTTP/1.0"
	if proto11 {
		proto = "HTTP/1.1"
	}
	codeStr := strconv.Itoa(code)
	text := http.StatusText(code)
	if text != "" {
		line = proto + " " + codeStr + " " + text + "\r\n"
		svr.statusLock.Lock()
		svr.statusLines[key] = line
		svr.statusLock.Unlock()
	} else {
		text = "status code " + codeStr
		line = proto + " " + codeStr + " " + text + "\r\n"
	}

	return line
}

// Start is used by server.go to start the rpc listener.
func (svr *PoolServer) Start() {
	if atomic.AddInt32(&svr.started, 1) != 1 {
		return
	}

	log.Trace("Starting pool server...")
	rpcServeMux := http.NewServeMux()
	httpServer := &http.Server{
		Handler: rpcServeMux,

		// Timeout connections which don't complete the initial
		// handshake within the allowed timeframe.
		ReadTimeout: time.Second * rpcAuthTimeoutSeconds,
	}

	// Http endpoint.
	rpcServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		w.Header().Set("Content-Type", "application/json")
		r.Close = true

		// Limit the number of connections to max allowed.
		if svr.limitConnections(w, r.RemoteAddr) {
			return
		}

		// Keep track of the number of connected clients.
		svr.incrementClients()
		defer svr.decrementClients()
		_, isAdmin, err := svr.checkAuth(r, true)
		if err != nil {
			jsonAuthFail(w)
			return
		}
		// We do not allow no admin user to use http
		if !isAdmin {
			jsonAuthFail(w)
			return
		}

		// Read and respond to the request.
		svr.jsonRPCRead(w, r, isAdmin)
	})

	// Websocket endpoint.
	rpcServeMux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		if isUndesiredIP(r.RemoteAddr, svr.commonCfg.Blacklist, svr.commonCfg.Whitelist) {
			jsonIPForbidden(w)
			return
		}

		authenticated, isAdmin, err := svr.checkAuth(r, false)
		if err != nil {
			jsonAuthFail(w)
			return
		}
		// If run in single mode, do not allow non-admin users to connect
		if svr.cfg.DisableConnectToAbec && !isAdmin {
			jsonAuthFail(w)
			return
		}

		if isAdmin && isUndesiredIP(r.RemoteAddr, svr.commonCfg.AdminBlacklist, svr.commonCfg.AdminWhitelist) {
			jsonIPForbidden(w)
			return
		}

		// Attempt to upgrade the connection to a websocket connection
		// using the default size for read/write buffers.
		ws, err := websocket.Upgrade(w, r, nil, 0, 0)
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				log.Errorf("Unexpected websocket error: %v",
					err)
			}
			http.Error(w, "400 Bad Request.", http.StatusBadRequest)
			return
		}
		svr.WebsocketHandler(ws, r.RemoteAddr, authenticated, isAdmin)
	})

	for _, listener := range svr.cfg.Listeners {
		svr.wg.Add(1)
		go func(listener net.Listener) {
			tlsState := "on"
			if svr.cfg.DisableTLS {
				tlsState = "off"
			}
			log.Infof("Pool websocket RPC server listening on %s (TLS %s)", listener.Addr(), tlsState)
			httpServer.Serve(listener)
			log.Tracef("Pool websocket RPC listener done for %s", listener.Addr())
			svr.wg.Done()
		}(listener)
	}

	svr.ntfnMgr.Start()
}

// Stop is used by server.go to stop the pool rpc listener.
func (svr *PoolServer) Stop() error {
	if atomic.AddInt32(&svr.shutdown, 1) != 1 {
		log.Infof("Pool websocket RPC server is already in the process of shutting down")
		return nil
	}
	log.Warnf("Pool websocket RPC server shutting down...")
	for _, listener := range svr.cfg.Listeners {
		err := listener.Close()
		if err != nil {
			log.Errorf("Problem shutting down pool websocket RPC server: %v", err)
			return err
		}
	}
	svr.ntfnMgr.Shutdown()
	svr.ntfnMgr.WaitForShutdown()
	close(svr.quit)
	svr.wg.Wait()
	log.Infof("Pool websocket RPC server shutdown complete")
	return nil
}

// limitConnections responds with a 503 service unavailable and returns true if
// adding another client would exceed the maximum allow RPC clients.
//
// This function is safe for concurrent access.
func (svr *PoolServer) limitConnections(w http.ResponseWriter, remoteAddr string) bool {
	if int(atomic.LoadInt32(&svr.numClients)+1) > svr.cfg.RPCMaxClients {
		log.Infof("Max Pool RPC clients exceeded [%d] - "+
			"disconnecting client %s", svr.cfg.RPCMaxClients,
			remoteAddr)
		http.Error(w, "503 Too busy.  Try again later.",
			http.StatusServiceUnavailable)
		return true
	}
	return false
}

// incrementClients adds one to the number of connected RPC clients.  Note
// this only applies to standard clients.  Websocket clients have their own
// limits and are tracked separately.
//
// This function is safe for concurrent access.
func (svr *PoolServer) incrementClients() {
	atomic.AddInt32(&svr.numClients, 1)
}

// decrementClients subtracts one from the number of connected RPC clients.
// Note this only applies to standard clients.  Websocket clients have their own
// limits and are tracked separately.
//
// This function is safe for concurrent access.
func (svr *PoolServer) decrementClients() {
	atomic.AddInt32(&svr.numClients, -1)
}

// checkAuth checks the HTTP Basic authentication. If the supplied authentication
// does not match the username and password expected, a non-nil error is
// returned.
//
// This check is time-constant.
//
// The first bool return value signifies auth success (true if successful) and
// the second bool return value specifies whether the user can change the state
// of the server (true) or whether the user is limited (false). The second is
// always false if the first is.
func (svr *PoolServer) checkAuth(r *http.Request, require bool) (bool, bool, error) {
	authhdr := r.Header["Authorization"]
	if len(authhdr) <= 0 {
		if require {
			log.Warnf("RPC authentication failure from %s",
				r.RemoteAddr)
			return false, false, errors.New("auth failure")
		}

		return false, false, nil
	}

	authsha := sha256.Sum256([]byte(authhdr[0]))

	// Check for limited auth first as in environments with limited users, those
	// are probably expected to have a higher volume of calls
	limitcmp := subtle.ConstantTimeCompare(authsha[:], svr.limitauthsha[:])
	if limitcmp == 1 {
		return true, false, nil
	}

	db := svr.minerManager.Db
	userService := service.GetUserService()
	success, err := userService.LoginAdmin(context.Background(), db, authhdr[0])
	if err != nil || !success {
		return false, false, errors.New("auth failure")
	}
	return true, true, nil
}

// jsonAuthFail sends a message back to the client if the http auth is rejected.
func jsonAuthFail(w http.ResponseWriter) {
	w.Header().Add("WWW-Authenticate", `Basic realm="abe mining pool"`)
	http.Error(w, "401 Unauthorized.", http.StatusUnauthorized)
}

func jsonIPForbidden(w http.ResponseWriter) {
	http.Error(w, "403 Forbidden.", http.StatusForbidden)
}

// WebsocketHandler handles a new websocket client by creating a new wsClient,
// starting it, and blocking until the connection closes.  Since it blocks, it
// must be run in a separate goroutine.  It should be invoked from the websocket
// server handler which runs each new connection in a new goroutine thereby
// satisfying the requirement.
func (svr *PoolServer) WebsocketHandler(conn *websocket.Conn, remoteAddr string,
	authenticated bool, isAdmin bool) {

	// Clear the read deadline that was set before the websocket hijacked
	// the connection.
	conn.SetReadDeadline(timeZeroVal)

	// Limit max number of websocket clients.
	log.Infof("New websocket client %s", remoteAddr)
	if svr.ntfnMgr.NumClients()+1 > svr.cfg.RPCMaxWebsockets {
		log.Infof("Max websocket clients exceeded [%d] - "+
			"disconnecting client %s", svr.cfg.RPCMaxWebsockets,
			remoteAddr)
		conn.Close()
		return
	}

	// Create a new websocket client to handle the new websocket connection
	// and wait for it to shutdown.  Once it has shutdown (and hence
	// disconnected), remove it and any notifications it registered for.
	client, err := newWebsocketClient(svr, conn, remoteAddr, authenticated, isAdmin)
	if err != nil {
		log.Errorf("Failed to serve client %s: %v", remoteAddr, err)
		conn.Close()
		return
	}
	svr.ntfnMgr.AddClient(client)
	client.Start()
	client.WaitForShutdown()
	svr.ntfnMgr.RemoveClient(client)
	log.Infof("Disconnected websocket client %s", remoteAddr)
}

// Start begins processing input and output messages.
func (c *wsClient) Start() {
	log.Tracef("Starting websocket client %s", c.addr)

	// Start processing input and output.
	c.wg.Add(3)
	go c.inHandler()
	go c.notificationQueueHandler()
	go c.outHandler()
}

// WaitForShutdown blocks until the websocket client goroutines are stopped
// and the connection is closed.
func (c *wsClient) WaitForShutdown() {
	c.wg.Wait()
}

// createMarshalledReply returns a new marshalled JSON-RPC response given the
// passed parameters.  It will automatically convert errors that are not of
// the type *pooljson.RPCError to the appropriate type as needed.
func createMarshalledReply(id, result interface{}, replyErr error) ([]byte, error) {
	var jsonErr *pooljson.RPCError
	if replyErr != nil {
		if jErr, ok := replyErr.(*pooljson.RPCError); ok {
			jsonErr = jErr
		} else {
			jsonErr = internalRPCError(replyErr.Error(), "")
		}
	}

	return pooljson.MarshalResponse(id, result, jsonErr)
}

// parsedRPCCmd represents a JSON-RPC request object that has been parsed into
// a known concrete command along with any error that might have happened while
// parsing it.
type parsedRPCCmd struct {
	id     interface{}
	method string
	cmd    interface{}
	err    *pooljson.RPCError
}

// parseCmd parses a JSON-RPC request object into known concrete command.  The
// err field of the returned parsedRPCCmd struct will contain an RPC error that
// is suitable for use in replies if the command is invalid in some way such as
// an unregistered command or invalid parameters.
func parseCmd(request *pooljson.Request) *parsedRPCCmd {
	var parsedCmd parsedRPCCmd
	parsedCmd.id = request.ID
	parsedCmd.method = request.Method

	fmt.Printf("method %v\n", parsedCmd.method)
	fmt.Printf("id %v\n", parsedCmd.id)
	fmt.Printf("request Jsonrpc %v\n", request.Jsonrpc)
	fmt.Printf("request Params %v\n", request.Params)
	// fmt.Println(string(request))

	// request &{2.0 eth_submitLogin ["0xE3E7f26a22f5227cDaa643Bc9aE458b3114301D1" "x"] %!s(float64=2)}

	cmd, err := pooljson.UnmarshalCmd(request)
	if err != nil {
		// When the error is because the method is not registered,
		// produce a method not found RPC error.
		fmt.Printf("request %v\n", err)

		if jerr, ok := err.(pooljson.Error); ok &&
			jerr.ErrorCode == pooljson.ErrUnregisteredMethod {

			parsedCmd.err = pooljson.ErrRPCMethodNotFound
			return &parsedCmd
		}

		// Otherwise, some type of invalid parameters is the
		// cause, so produce the equivalent RPC error.
		parsedCmd.err = pooljson.NewRPCError(
			pooljson.ErrRPCInvalidParams.Code, err.Error())
		return &parsedCmd
	}

	parsedCmd.cmd = cmd
	return &parsedCmd
}

// parseSimplifiedCmd parses a JSON-RPC request object into known concrete command.  The
// err field of the returned parsedRPCCmd struct will contain an RPC error that
// is suitable for use in replies if the command is invalid in some way such as
// an unregistered command or invalid parameters.
func parseSimplifiedCmd(request *pooljson.SimplifiedRequest) *parsedRPCCmd {
	var parsedCmd parsedRPCCmd
	parsedCmd.id = request.ID
	parsedCmd.method = request.Method

	cmd, err := pooljson.UnmarshalCmd2(request)
	if err != nil {
		// When the error is because the method is not registered,
		// produce a method not found RPC error.
		if jerr, ok := err.(pooljson.Error); ok &&
			jerr.ErrorCode == pooljson.ErrUnregisteredMethod {

			parsedCmd.err = pooljson.ErrRPCMethodNotFound
			return &parsedCmd
		}

		// Otherwise, some type of invalid parameters is the
		// cause, so produce the equivalent RPC error.
		parsedCmd.err = pooljson.NewRPCError(
			pooljson.ErrRPCInvalidParams.Code, err.Error())
		return &parsedCmd
	}

	parsedCmd.cmd = cmd
	return &parsedCmd
}

// standardCmdResult checks that a parsed command is a standard pool JSON-RPC
// command and runs the appropriate handler to reply to the command.  Any
// commands which are not recognized or not implemented will return an error
// suitable for use in replies.
func (svr *PoolServer) standardCmdResult(cmd *parsedRPCCmd, closeChan <-chan struct{}) (interface{}, error) {
	// Recovery
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Panic from %v handler: %v\n", cmd.method, err)
			var buf [4096]byte
			n := runtime.Stack(buf[:], false)
			log.Errorf("Stack Trace ==>\n %s\n", string(buf[:n]))
			log.Infof("Recovering...")
			// Dump panic file
			_ = utils.DumpPanicInfo(fmt.Sprintf("%v", err) + "\n" + string(buf[:]))
		}
	}()

	handler, ok := rpcHandlers[cmd.method]
	if ok {
		goto handled
	}
	return nil, pooljson.ErrRPCMethodNotFound
handled:

	return handler(svr, cmd.cmd, closeChan)
}

// HandleChainClientNotification handles notifications from chain client.  It does
// things such as notify template changed and so on.
func (svr *PoolServer) HandleChainClientNotification(notification *chainclient.Notification) {
	switch notification.Type {

	}
}

// HandleMinerManagerNotification handles notifications from miner manager. It does
// things such as notify share difficulty changed and so on.
func (svr *PoolServer) HandleMinerManagerNotification(notification *minermgr.Notification) {
	switch notification.Type {

	case minermgr.NTNewJobReady:
		newJob, ok := notification.Data.(*model.JobTemplate)
		if !ok {
			log.Warnf("The NTNewJobReady notification is not a job template!")
			break
		}

		svr.ntfnMgr.NotifyNewJob(newJob)

	case minermgr.NTEpochChange:
		newEpoch, ok := notification.Data.(int64)
		if !ok {
			log.Warnf("The NTEpochChange notification is not a int64 value!")
			break
		}

		svr.ntfnMgr.NotifyNewEpoch(newEpoch)
	}

}

// isUndesiredIP determines whether the server should continue to pursue
// a connection with this peer based on its ip address. It performs
// the following steps:
// 1) Reject the peer if it contains a blacklisted ip.
// 2) If no whitelist is provided, accept all ip.
// 3) Accept the peer if it contains a whitelisted ip.
// 4) Reject all other peers.
func isUndesiredIP(remoteAddress string, blacklistedIPs, whitelistedIPs []*net.IPNet) bool {

	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		log.Warnf("Unable to SplitHostPort on '%s': %v", remoteAddress, err)
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		log.Warnf("Unable to parse IP '%s'", remoteAddress)
		return false
	}

	for _, blacklistedIP := range blacklistedIPs {
		if blacklistedIP.Contains(ip) {
			log.Debugf("Ignoring peer %s because it contains blacklisted ip: %v", remoteAddress, blacklistedIP)
			return true
		}
	}

	// If no whitelist is provided, we will accept all peers.
	if len(whitelistedIPs) == 0 {
		return false
	}

	// Check to see if it contains one of our whitelisted ip, if so accept.
	for _, whitelistedIP := range whitelistedIPs {
		if whitelistedIP.Contains(ip) {
			return false
		}
	}

	// Otherwise, the peer's ip was not included in our whitelist.
	log.Debugf("Ignoring peer %s because it is not in whitelist", remoteAddress)

	return true
}

func init() {
	rpcHandlers = rpcHandlersBeforeInit
	rand.Seed(time.Now().UnixNano())
}

package ordermgr

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abesuite/abe-miningpool-server/abejson"
	"github.com/abesuite/abe-miningpool-server/chainclient"
	"github.com/abesuite/abe-miningpool-server/dal"
	"github.com/abesuite/abe-miningpool-server/dal/do"
	"github.com/abesuite/abe-miningpool-server/model"
	"github.com/abesuite/abe-miningpool-server/service"
	"github.com/abesuite/abe-miningpool-server/utils"
	"github.com/abesuite/abe-miningpool-server/walletclient"

	"gorm.io/gorm"
)

type OrderManager struct {
	rewardConfig *model.RewardConfig
	adminUserID  uint64

	chain                 *chainclient.RPCClient
	currentChainHeightMtx sync.Mutex
	currentChainHeight    int64
	redeemingFlag         atomic.Value

	changeAddress      []string
	txFeeSpecified     float64
	txFeeSpecifiedLock sync.Mutex

	db *gorm.DB

	enableAutoFlag bool // control the access of poolctl
	autoFlag       bool // control whether the wallet task work

	walletClient           *walletclient.RPCClient
	currentWalletHeightMtx sync.Mutex
	currentWalletHeight    int64

	checkedHeight int64
	checkingFlag  atomic.Value

	wg       sync.WaitGroup
	shutdown int32
	quit     chan struct{}
}

func NewOrderManager(config *model.RewardConfig, changeAddress []string, txFeeSpecified float64,
	chainClient *chainclient.RPCClient, walletClient *walletclient.RPCClient) *OrderManager {
	ctx := context.Background()
	adminUser, err := service.GetUserService().GetUserByUsername(ctx, dal.GlobalDBClient, "admin")
	if err != nil {
		return nil
	}
	metaInfo, err := service.GetMetaInfoService().Get(ctx, dal.GlobalDBClient)
	if err != nil {
		return nil
	}
	res := &OrderManager{
		rewardConfig:        config,
		adminUserID:         adminUser.ID,
		currentChainHeight:  -1,
		currentWalletHeight: -1,
		changeAddress:       changeAddress,
		txFeeSpecified:      txFeeSpecified,
		db:                  dal.GlobalDBClient,
		enableAutoFlag:      true,
		chain:               chainClient,
		walletClient:        walletClient,
		checkingFlag:        atomic.Value{},
		redeemingFlag:       atomic.Value{},
		checkedHeight:       metaInfo.LastTxCheckHeight,
		quit:                make(chan struct{}),
	}
	res.redeemingFlag.Store(false)
	res.checkingFlag.Store(false)
	return res
}

// HandleChainClientNotification handles notifications from chain client.
func (m *OrderManager) HandleChainClientNotification(notification *chainclient.Notification) {
	switch notification.Type {

	case chainclient.NTBlockTemplateChanged:
		log.Trace("Order manager receives a new block template from chain client, updating...")
		blockTemplate, ok := notification.Data.(*model.BlockTemplate)
		if !ok {
			log.Errorf("Order manager accepted notification is not a block template.")
			break
		}
		if blockTemplate == nil {
			log.Errorf("Received block template is nil")
			break
		}
		m.currentChainHeightMtx.Lock()
		m.currentChainHeight = blockTemplate.Height
		m.currentChainHeightMtx.Unlock()

	case chainclient.NTBlockConnected:
		log.Trace("Order manager receives a new block connect to chain, check and allocate reward...")
		blockNotification, ok := notification.Data.(*model.BlockNotification)
		if !ok {
			log.Errorf("Order manager accepted notification is not a block connect.")
			break
		}
		if blockNotification == nil {
			log.Errorf("Received block connect notification is nil")
			break
		}
		// only allocate the reward at height % 100 = 0
		// TODO 52000 52100 52200
		//triggerPoint := m.rewardConfig.RewardMaturity%m.rewardConfig.RewardInterval + m.rewardConfig.DelayInterval
		m.currentChainHeightMtx.Lock()
		m.currentChainHeight = int64(blockNotification.Height)
		m.currentChainHeightMtx.Unlock()
		//log.Infof("Checking allocating reward with wallet at height %d(rewardInterval = %d, triggerPoint = %d)...", height, m.rewardConfig.RewardInterval, triggerPoint)

	}
}
func (m *OrderManager) redeemBalanceHandler() {
	log.Infof("redeemBalanceHandler working")
	defer m.wg.Done()
	duration := 20 * time.Minute
	timer := time.NewTicker(duration)
	for {
		select {
		case <-timer.C:
			if m.walletClient == nil || m.walletClient.Client == nil {
				log.Errorf("Error when creating transactions: wallet client is nil")
				continue
			}
			if m.walletClient.Disconnected() {
				log.Errorf("Error when creating transactions: wallet is disconnected")
				continue
			}

			if !m.enableAutoFlag || !m.autoFlag {
				log.Infof("wallet auto allocate flag need to be set active")
				continue
			}

			m.currentChainHeightMtx.Lock()
			height := m.currentChainHeight
			m.currentChainHeightMtx.Unlock()
			ctx := context.Background()
			config, err := service.GetConfigInfoService().GetByHeight(ctx, m.db, height)
			if err != nil || config == nil {
				log.Errorf("Unable to get config info for height %d due %v", height, err)
				continue
			}
			metaInfo, err := service.GetMetaInfoService().Get(ctx, m.db)
			if err != nil || metaInfo == nil {
				log.Errorf("Unable to get meta info")
				continue
			}
			triggerHeight := config.StartHeight + (height-config.StartHeight)/config.RewardInterval*config.RewardInterval + int64(m.rewardConfig.RewardMaturity) + int64(m.rewardConfig.DelayInterval)
			if !(height >= triggerHeight && triggerHeight > metaInfo.LastRedeemedHeight) {
				log.Infof("config %#v", config)
				log.Infof("reward config %#v", m.rewardConfig)
				log.Infof("current height %d, triggerHeight %d, lastRedeemedHeight %d", height, triggerHeight, metaInfo.LastRedeemedHeight)
				continue
			}

			log.Infof("Allocating reward with wallet at height %d(rewardInterval = %d, triggerHeight = %d, lastRewardHeight = %d)...", height, m.rewardConfig.RewardInterval, triggerHeight, metaInfo.LastRedeemedHeight)

			// check the status
			if m.redeemingFlag.Load().(bool) {
				log.Debug("checkingFlag is set true, return...")
				break
			}
			if !m.redeemingFlag.CompareAndSwap(false, true) {
				log.Debug("fail to set checkingFlag true, return...")
				break
			}
			m.redeemBalance(ctx, triggerHeight)
			m.redeemingFlag.CompareAndSwap(true, false)
			timer.Reset(duration)
		case <-m.quit:
			return
		default:

		}
	}
}
func (m *OrderManager) redeemBalance(ctx context.Context, triggerHeight int64) {
	taskChan := m.walletClient.GetTaskChan()
	if taskChan == nil {
		log.Errorf("unable to get wallet task chan")
		return
	}

	userService := service.GetUserService()
	log.Infof("Creating order for allocate reward with wallet...")

	// filter allocated utxo hash
	// use the unfinished order ( utxo_hash!='',status = 0)
	unfinishedAllocatedOrders, err := userService.GetUnfinishedOrders(ctx, m.db)
	if err != nil {
		log.Errorf("Fai to acquire the unfinished orders")
		return
	}
	// use the unfinished order transaction filter(status=-1 or 3)
	unfinishedOrderTransactionInfos, err := userService.GetUnfinishedOrderTransactions(ctx, m.db)
	if err != nil {
		log.Errorf("Fai to acquire the unfinished order transactions")
		return
	}
	allocatedUTXOHashMapping := map[string]struct{}{}
	for _, orderInfo := range unfinishedAllocatedOrders {
		allocatedUTXOHashMapping[orderInfo.UTXOHash] = struct{}{}
	}
	for _, orderTransactionInfo := range unfinishedOrderTransactionInfos {
		allocatedUTXOHashMapping[orderTransactionInfo.OrderInfo.UTXOHash] = struct{}{}
	}

	// Before creating order, acquire and split the UTXO list for help to decide the manage fee
	utxoList := m.walletClient.GetUTXOs()
	utxoMapping, epochMapping, epochs := m.generateEpochMapping(ctx, utxoList, allocatedUTXOHashMapping)
	// iterate with epoch with it order
	for _, epoch := range epochs {
		utxos := epochMapping[epoch]
		// map : epoch number -> utxo list
		log.Debugf("Epoch %d with list:", epoch)
		for _, utxo := range utxos {
			log.Debugf("\t Height %d, Amount %d, UTXOHash %s", utxo.Height, utxo.Amount, utxo.UTXOHash)
		}

		configInfo, err := service.GetConfigInfoService().GetByEpoch(ctx, m.db, epoch)
		if err != nil || configInfo == nil {
			log.Errorf("Query config with epoch %d error: %v", epoch, err)
			break
		}
		epochUnit := utils.CalculateCoinbaseRewardByEpoch(epoch) / 400 * (100 - uint64(configInfo.ManageFeePercent))
		// calculate number of allocatable units and it unit from epoch
		unitCnt, err := m.generateUnits(ctx, epochUnit, utxos)
		if err != nil {
			log.Errorf("Check %d utxo in epoch %d, it units is not unify", len(utxos), epoch)
			break
		}
		// create order for all redeemable user
		err = userService.CreateOrders(ctx, m.db, unitCnt, epochUnit, utxos)
		if err != nil {
			log.Errorf("Create order error: %v", err)
			break
		}
		log.Infof("Done order for allocate reward with wallet...")

		// unfinished orders
		// status = 0 and amount = unit *1 2 4 and utxo_hash!=''
		unfinishedOrders, err := userService.GetUnfinishedOrdersByAmountUnit(ctx, m.db, int64(epochUnit))
		if err != nil {
			log.Errorf("Get unfinished order error: %v", err)
			break
		}
		log.Debugf("Unfinished Orders:")
		for utxoHashStr, infos := range unfinishedOrders {
			log.Debugf("\tutxo hash %s", utxoHashStr)
			for i := 0; i < len(infos); i++ {
				log.Debugf("\t\t ID %d, UserID %d, Amount %d ", infos[i].ID, infos[i].UserID, infos[i].Amount)
			}
		}

		log.Infof("Allocating reward with wallet： currentBlockRewardQuarter= %d...", epochUnit)
		var ok bool
		m.txFeeSpecifiedLock.Lock()
		feeSpecified := m.txFeeSpecified // transaction fee
		m.txFeeSpecifiedLock.Unlock()
		// user info cache, avoid query to database if possible
		log.Infof("Create order_transaction_info entry for unfinished order...")
		userInfos := map[uint64]*do.UserInfo{}
		for utxoHashStr, infos := range unfinishedOrders {
			orderIds := make([]uint64, 0, 5)
			flag := true
			for i := 0; i < len(infos); i++ {
				if userInfo, ok := userInfos[infos[i].UserID]; !ok {
					userInfo, err = userService.GetUserByID(ctx, m.db, infos[i].UserID)
					if err != nil {
						log.Errorf("Get user info error: %v", err)
						flag = false
						break
					}
					userInfos[infos[i].UserID] = userInfo
				}
				orderIds = append(orderIds, infos[i].ID)
				log.Infof("User ID %d: Amount = %#v, UTXOHash = %s", infos[i].UserID, infos[i].Amount, infos[i].UTXOHash)
			}
			if !flag {
				continue
			}
			utxoSpecified := utxoHashStr
			log.Infof("utxo hash %s", utxoSpecified)
			if _, ok = utxoMapping[utxoSpecified]; !ok {
				log.Errorf("The utxo hash %s is not exist in the available list for order ids [%v]", utxoSpecified, orderIds)
				continue
			}
			requestContentBuff := &bytes.Buffer{}
			var requestTransactionHashStr string
			err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				totalAmount := int64(0)
				// compute transaction request hash
				for i := 0; i < len(infos); i++ {
					totalAmount += infos[i].Amount
					requestContentBuff.WriteString(userInfos[infos[i].UserID].Address)
					utils.WriteVarInt(requestContentBuff, 0, uint64(infos[i].Amount))
				}
				amount := float64(utxoMapping[utxoSpecified].Amount) - float64(totalAmount) - feeSpecified
				randomIndex := rand.Intn(len(m.changeAddress))
				requestContentBuff.WriteString(m.changeAddress[randomIndex])
				utils.WriteVarInt(requestContentBuff, 0, uint64(amount))
				requestContentBuff.WriteString(utxoSpecified)
				utils.WriteVarInt(requestContentBuff, 0, uint64(feeSpecified))
				requestTransactionHashStr = utils.HashH(requestContentBuff.Bytes()).String()

				for i := 0; i < len(infos); i++ {
					err = userService.SetOrderStatus(ctx, tx, orderIds[i], 1)
					if err != nil {
						return err
					}
					_, err = userService.CreateOrderTransaction(ctx, tx, &model.OrderTransaction{
						OrderID:                orderIds[i],
						Amount:                 infos[i].Amount,
						Address:                userInfos[infos[i].UserID].Address,
						RequestTransactionHash: requestTransactionHashStr,
					})
					if err != nil {
						return err
					}
				}
				// creat an order / order_transaction entry for manage fee with user 'admin'
				orderForAdmin, err := userService.CreateOrderForAdmin(ctx, tx, m.adminUserID, int64(amount), utxoSpecified)
				if err != nil {
					return err
				}
				_, err = userService.CreateOrderTransaction(ctx, tx, &model.OrderTransaction{
					OrderID:                orderForAdmin.ID,
					Amount:                 orderForAdmin.Amount,
					Address:                m.changeAddress[randomIndex],
					RequestTransactionHash: requestTransactionHashStr,
				})
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				log.Warnf("Fail to creating transaction for order IDs %v, computed request_hash %s with error %s", orderIds, requestTransactionHashStr, err)
			}
		}
		log.Infof("Done create order_transaction_info entry for unfinished order...")
	}

	// double-check the channel available
	taskChan = m.walletClient.GetTaskChan()
	if taskChan == nil {
		log.Errorf("unable to get wallet task chan")
		return
	}
	err = m.issueTransactionRequest(ctx, utxoMapping, taskChan)
	if err != nil {
		log.Errorf("Error issue transaction request for %v,err")
		return
	}
	log.Infof("Done allocate reward with wallet...")
	_, err = service.GetMetaInfoService().UpdateLastRedeemedHeight(ctx, m.db, triggerHeight)
	if err != nil {
		log.Errorf("unable to update the last redeemed height to %d", triggerHeight)
		return
	}
}

func (m *OrderManager) issueTransactionRequest(ctx context.Context, utxoMapping map[string]*abejson.UTXO, taskChan chan<- *abejson.BatchOrderInfo) error {
	// To query the order_transaction_info with status == -1
	// REPLACE BY the follow logic to avoid OOM in some condition
	// 1. query all different transaction_request_hash with status = -1
	// 2. loop transaction_request_hash list:
	// 2.1 query all order_transaction_info entry with current transaction_request_hash and status = -1
	// 2.2 assert the total amount and the utxo amount
	// 2.3 create the request command to abewallet rpc client
	// 3. wait the response and handle the transaction_hash

	userService := service.GetUserService()
	transactionRequestHashList, err := userService.GetTransactionRequestHash(ctx, m.db)
	if err != nil {
		return nil
	}
	log.Infof("Start send transaction request...")
	zero := 0
	fzero := float64(1)
	m.txFeeSpecifiedLock.Lock()
	feeSpecified := m.txFeeSpecified
	m.txFeeSpecifiedLock.Unlock()
	wg := sync.WaitGroup{}
	for _, transactionRequestHash := range transactionRequestHashList {
		log.Infof("request hash %s", transactionRequestHash)
		infos, err := userService.GetOrderTransactionByTransactionRequestHash(ctx, m.db, transactionRequestHash)
		if err != nil {
			continue
		}
		// check the number of infos with the automatic allocate logic
		if len(infos) < 2 || len(infos) > 5 {
			log.Warnf("something error with transaction request hash %s, please check the database", transactionRequestHash)
			continue
		}

		// copy the specified utxo hash to avoid reference the whole order_transaction_info
		// because it would be sent by channel
		utxoSpecifiedCopyed := infos[0].UTXOHash
		response := make(chan *abejson.TxStatus)
		// create the command to send transaction
		batchOrderInfo := &abejson.BatchOrderInfo{
			ResponseChan: response,
			BatchOrderInfoCmd: &abejson.BatchOrderInfoCmd{
				Pairs:              make([]*abejson.OrderInfo, 0, 5),
				MinConf:            &zero,
				ScaleToFeeSatPerKb: &fzero,
				FeeSpecified:       &feeSpecified, // the fixed transaction fee
			},
		}
		orderTransactionInfoIDs := make([]uint64, 0, len(infos))
		totalAmount := int64(0)
		for i := 0; i < len(infos); i++ {
			orderTransactionInfoIDs = append(orderTransactionInfoIDs, infos[i].ID)
			totalAmount += infos[i].Amount
			batchOrderInfo.Pairs = append(batchOrderInfo.BatchOrderInfoCmd.Pairs, &abejson.OrderInfo{
				OrderID: infos[i].ID,
				Address: infos[i].Address,
				Amount:  float64(infos[i].Amount),
			})
		}
		if _, ok := utxoMapping[utxoSpecifiedCopyed]; !ok {
			log.Errorf("The specified utxo %s is not exist in the utxo list of current epoch for ids [%v]", utxoSpecifiedCopyed, orderTransactionInfoIDs)
			continue
		}
		// check the equality of amount and fee
		if totalAmount+int64(feeSpecified) != int64(utxoMapping[utxoSpecifiedCopyed].Amount) {
			log.Errorf("The total amount of allocated order_transaction_info ids [%v] not equal for utxo %s, may the fee is changed after the order_transaction_info created", orderTransactionInfoIDs, utxoSpecifiedCopyed)
			continue
		}

		// use the copyed utxo specified because it would be sent to channel
		batchOrderInfo.BatchOrderInfoCmd.UTXOSpecified = &utxoSpecifiedCopyed
		// send to command
		taskChan <- batchOrderInfo
		wg.Add(1)
		go func(orderTransactionInfoIDs []uint64) {
			defer wg.Done()
			select {
			case resp := <-response:
				if resp == nil {
					log.Debugf("Receive nil  for order transaction info IDs %v with nil", orderTransactionInfoIDs)
					return
				}
				log.Debugf("Receive transaction hash after creating transaction for order ID %v with hash %s and status %d", orderTransactionInfoIDs, resp.TxHash, resp.Status)
				if resp.Status == do.TransactionPending { // time out for this request
					return
				}

				txHashStr := ""
				if resp.TxHash != nil {
					txHashStr = resp.TxHash.String()
				}
				err := userService.SetOrderTransactionInfoByIDs(ctx, m.db, orderTransactionInfoIDs, txHashStr, do.TransactionPending, resp.Status, int(m.currentChainHeight))
				if err != nil {
					log.Warnf("Fail to update order transaction info IDs %v with error %s, but the transaction is successful", orderTransactionInfoIDs, err)
				}
			}
		}(orderTransactionInfoIDs)
	}
	wg.Wait()
	log.Infof("Done send transaction request...")
	return nil
}

func (m *OrderManager) generateEpochMapping(
	ctx context.Context,
	utxoList []*abejson.UTXO,
	allocatedUTXOHashMapping map[string]struct{},
) (map[string]*abejson.UTXO, map[int64][]*abejson.UTXO, []int64) {
	// establish mapping from utxo string to struct for all utxo in utxoList
	utxoMapping := map[string]*abejson.UTXO{}
	// establish mapping from epoch num to all utxo in that epoch for all utxo in utxoList
	epochMapping := map[int64][]*abejson.UTXO{}
	epochs := make([]int64, 0)
	configInfoService := service.GetConfigInfoService()
	for i := 0; i < len(utxoList); i++ {
		utxoHashStr := utxoList[i].UTXOHash.String()
		utxoMapping[utxoHashStr] = utxoList[i]

		// the utxo is set to allocated when it is in allocatedUTXOHashMapping,
		// or it has invalid config information
		if _, ok := allocatedUTXOHashMapping[utxoHashStr]; ok {
			utxoList[i].Allocated = true
		}
		configInfo, err := configInfoService.GetByHeight(ctx, m.db, utxoList[i].Height)
		if err != nil {
			log.Errorf("Query config with height %d error: %v", utxoList[i].Height, err)
			utxoList[i].Allocated = true
		}
		if configInfo == nil {
			log.Errorf("Query config with height %d error: %v", utxoList[i].Height, err)
			utxoList[i].Allocated = true
			continue
		}
		if _, ok := epochMapping[configInfo.Epoch]; !ok {
			epochMapping[configInfo.Epoch] = make([]*abejson.UTXO, 0, len(utxoList))
			epochs = append(epochs, configInfo.Epoch)
		}
		epochMapping[configInfo.Epoch] = append(epochMapping[configInfo.Epoch], utxoList[i])
	}
	sort.Slice(epochs, func(i, j int) bool {
		return epochs[i] < epochs[j]
	})
	for epoch := range epochMapping {
		sort.Slice(epochMapping[epoch], func(i, j int) bool {
			return epochMapping[epoch][i].Height < epochMapping[epoch][j].Height
		})
	}
	return utxoMapping, epochMapping, epochs
}
func (m *OrderManager) generateUnits(ctx context.Context, epochUnit uint64, utxoList []*abejson.UTXO) (int, error) {
	unitCnt := 0
	configInfoService := service.GetConfigInfoService()
	// divide the utxo list by epoch
	for i := 0; i < len(utxoList); i++ {
		if utxoList[i].Allocated {
			continue
		}

		configInfo, err := configInfoService.GetByHeight(ctx, m.db, utxoList[i].Height)
		if err != nil || configInfo == nil {
			log.Errorf("Query config with height %d error: %v", utxoList[i].Height, err)
			utxoList[i].Allocated = true
			continue
		}
		// 256 ABE
		// 256000000 Neutrino
		// when the subsidy of block is 0.5ABE = 0.5*10^7 Neutrino which mods 400 == 0
		// basicUnit := utils.CalculateCoinbaseRewardByBlockHeight(utxoList[i].Height) * (1 - uint64(configInfo.ManageFeePercent) / 100 ) / 4
		basicUnit := utils.CalculateCoinbaseRewardByBlockHeight(utxoList[i].Height) / 400 * (100 - uint64(configInfo.ManageFeePercent))
		if basicUnit != epochUnit {
			return 0, fmt.Errorf("the unit %d computed from utxo list is not equal to epoch unit %d", basicUnit, epochUnit)
		}
		unitCnt += 4
	}

	return unitCnt, nil
}
func (m *OrderManager) checkTransactionHandler() {
	log.Infof("checkTransactionStatus working")
	defer m.wg.Done()
	duration := 40 * time.Minute
	timer := time.NewTicker(duration)
	for {
		select {
		case <-timer.C:
			if m.walletClient == nil || m.walletClient.Client == nil {
				log.Errorf("Error when checking transactions status: wallet client is nil")
				continue
			}
			if m.walletClient.Disconnected() {
				log.Errorf("Error when checking transactions status: wallet is disconnected")
				continue
			}

			m.currentWalletHeightMtx.Lock()
			currentWalletHeight := m.currentWalletHeight
			m.currentWalletHeightMtx.Unlock()

			// compare the synced height with the floorHeight
			checkedHeight := m.checkedHeight
			log.Infof("current checked height %d, current wallet height %d", checkedHeight, currentWalletHeight)
			if currentWalletHeight < checkedHeight+2*walletclient.CheckHeightDiff {
				break
			}
			log.Infof("%d > %d + 2 * % d, checking the transaction status", currentWalletHeight, checkedHeight, walletclient.CheckHeightDiff)

			// check the status
			if m.checkingFlag.Load().(bool) {
				log.Debug("checkingFlag is set true, return...")
				break
			}
			if !m.checkingFlag.CompareAndSwap(false, true) {
				log.Debug("fail to set checkingFlag true, return...")
				break
			}

			ctx := context.Background()
			finished, nextCheckedHeight := m.checkTransactionStatus(ctx, currentWalletHeight, checkedHeight)

			if finished {
				// update the latest synced height into the database
				metaService := service.GetMetaInfoService()
				_, err := metaService.UpdateLastTxCheckHeight(ctx, m.db, checkedHeight)
				if err != nil {
					log.Debugf("fail to update database with last transaction check height with %d: %s", checkedHeight, err)
					return
				}
				m.checkedHeight = nextCheckedHeight
				log.Infof("update the checked height in order manager with height %d from origin height %d", nextCheckedHeight, currentWalletHeight)
			}

			m.checkTransactionPendingStatus(ctx, currentWalletHeight)

			m.checkingFlag.CompareAndSwap(true, false)
			timer.Reset(duration)
		case <-m.quit:
			return
		default:
		}
	}
}

func (m *OrderManager) checkTransactionStatus(ctx context.Context, currentWalletHeight int64, checkedHeight int64) (bool, int64) {
	checkChan := m.walletClient.CheckChan()
	if checkChan == nil {
		log.Errorf("fail to acquire the check channel for updating transaction status, please check the pool or wallet status")
		return false, 0
	}

	userService := service.GetUserService()
	wg := sync.WaitGroup{}
	nextCheckedHeight := int64(currentWalletHeight)/walletclient.CheckHeightDiff*walletclient.CheckHeightDiff - walletclient.CheckHeightDiff
	txInfos, err := userService.GetOrderTransactionInfoRange(ctx, m.db, checkedHeight, nextCheckedHeight)
	if err != nil {
		log.Errorf("fail to acquire transaction status from database error : %s", err)
		return false, 0
	}
	log.Debugf("Check the transaction infos:")
	for i := 0; i < len(txInfos); i++ {
		log.Debugf("\t ID: %d, status: %d, transaction_hash: %s,request_hash: %s", txInfos[i].ID, txInfos[i].Status, txInfos[i].RequestTransactionHash, txInfos[i].TransactionHash)
	}
	// divide all order_transaction_info into group with the same request_hash and transaction_hash
	txInfoIDMapping := make(map[uint64]*model.OrderTransactionStatus, len(txInfos))
	requestHashIDMapping := make(map[string][]uint64, len(txInfos))
	txHashIDMapping := make(map[string][]uint64, len(txInfos))
	for i := 0; i < len(txInfos); i++ {
		txInfoIDMapping[txInfos[i].ID] = txInfos[i]

		if len(txInfos[i].RequestTransactionHash) != 0 && len(txInfos[i].TransactionHash) == 0 {
			log.Errorf("the request hash is zero hash and transaction hash is zero with order_transaction_info.ID = %v", txInfos[i].TransactionHash, txInfos[i].ID)
			continue
		}

		// the status is kept for long time, not update it
		if txInfos[i].LastUpdateStatusHeight-txInfos[i].Height > walletclient.CheckHeightDiff {
			continue
		}

		if len(txInfos[i].RequestTransactionHash) != 0 {
			if _, ok := requestHashIDMapping[txInfos[i].RequestTransactionHash]; !ok {
				requestHashIDMapping[txInfos[i].RequestTransactionHash] = make([]uint64, 0, 5)
			}
			requestHashIDMapping[txInfos[i].RequestTransactionHash] = append(requestHashIDMapping[txInfos[i].RequestTransactionHash], txInfos[i].ID)
		}

		if len(txInfos[i].TransactionHash) != 0 {
			if _, ok := txHashIDMapping[txInfos[i].TransactionHash]; !ok {
				txHashIDMapping[txInfos[i].TransactionHash] = make([]uint64, 0, 5)
			}
			txHashIDMapping[txInfos[i].TransactionHash] = append(txHashIDMapping[txInfos[i].TransactionHash], txInfos[i].ID)
		}
	}

	// mark all updated order_transaction_info with IDs
	flags := make(map[uint64]bool, len(txInfos))

	// update by request hash firstly
	for requestHash, otiIDs := range requestHashIDMapping {
		response := make(chan *abejson.TxStatus, 1)
		requestTxHash, err := utils.NewHashFromStr(requestHash)
		if err != nil {
			log.Errorf("transaction_request_hash in IDs %v is wrong string for hash", otiIDs)
			continue
		}
		checkChan <- &abejson.TxStatusInfo{
			TxHash:        nil,
			RequestTxHash: requestTxHash,
			ResponseChan:  response,
		}
		wg.Add(1)
		go func(IDs []uint64, previousStatus int, previousTxHash string, requestHash string) {
			defer wg.Done()
			resp := <-response
			if resp.Status == -2 {
				log.Warnf("fail to query for IDs [%v], waiting for next time", IDs)
				return
			}

			if resp.TxHash == nil || resp.TxHash.IsEqual(&utils.ZeroHash) {
				log.Errorf("Error for updating the order_transaction_infos for request hash %s with ids %v: %v", requestHash, IDs, err)
				return
			}

			if resp.Status == do.TransactionManual {
				log.Errorf("Please manual handle transaction for IDs [%v]", IDs)
			}

			txHashStr := resp.TxHash.String()
			status := resp.Status
			if previousTxHash != "" && previousTxHash != txHashStr {
				log.Errorf("Error for updating the order_transaction_infos for request hash %s with ids %v: resp.TxHash %s unmatched with previous tx hash %s", requestHash, IDs, txHashStr, previousStatus)
				txHashStr = previousTxHash
				status = do.TransactionManual
			}

			err = userService.SetOrderTransactionInfoByIDs(ctx, m.db, IDs, txHashStr, previousStatus, status, int(currentWalletHeight))
			if err != nil {
				log.Errorf("Error for updating the order_transaction_infos for request hash %s with ids %v: %v", requestHash, IDs, err)
				return
			}
			log.Infof("transaction hash %s update its status %d with request hash %s", resp.TxHash.String(), resp.Status, requestHash)
			for j := 0; j < len(IDs); j++ {
				if !flags[IDs[j]] {
					flags[IDs[j]] = true
				}
			}
		}(otiIDs, txInfoIDMapping[otiIDs[0]].Status, txInfoIDMapping[otiIDs[0]].TransactionHash, requestHash)
	}
	wg.Wait()

	finished := true
	for _, flag := range flags {
		if !flag {
			finished = false
			break
		}
	}
	if finished {
		return true, nextCheckedHeight
	}

	// update by transaction hash
	for txHashStr, otiIDs := range txHashIDMapping {
		// check it ok?
		finished = true
		for j := 0; j < len(otiIDs); j++ {
			if !flags[otiIDs[j]] {
				finished = false
			}
		}
		if finished {
			continue
		}
		response := make(chan *abejson.TxStatus, 1)
		txHash, _ := utils.NewHashFromStr(txHashStr)
		checkChan <- &abejson.TxStatusInfo{
			TxHash:        txHash,
			RequestTxHash: nil,
			ResponseChan:  response,
		}
		wg.Add(1)
		go func(IDs []uint64, previousStatus int) {
			defer wg.Done()
			resp := <-response
			if resp.Status == -2 {
				log.Warnf("fail to query for IDs [%v], waiting for next time", IDs)
				return
			}

			if resp.Status == do.TransactionManual {
				log.Errorf("Please manual handle transaction for IDs [%v]", IDs)
			}

			err = userService.SetOrderTransactionInfoByIDs(ctx, m.db, IDs, txHash.String(), previousStatus, resp.Status, int(currentWalletHeight))
			if err != nil {
				log.Errorf("Error for updating the order_transaction_infos for tx hash %s with ids %v: %s", txHash.String(), IDs, err)
				return
			}
			log.Infof("transaction hash %s update its status %d with tx hash %s for ids %v", resp.TxHash.String(), resp.Status, txHash.String(), IDs)
			for j := 0; j < len(IDs); j++ {
				if !flags[IDs[j]] {
					flags[IDs[j]] = true
				}
			}
		}(otiIDs, txInfoIDMapping[otiIDs[0]].Status)
	}
	wg.Wait()
	finished = true
	for id, flag := range flags {
		if !flag {
			log.Warnf("update order_transaction_infos.id %d failed", id)
			finished = false
		}
	}
	if !finished {
		log.Infof("fail to update the checked height in order manager with height %d from origin height %d", nextCheckedHeight, currentWalletHeight)
		return false, 0
	}
	return true, nextCheckedHeight
}

func (m *OrderManager) checkTransactionPendingStatus(ctx context.Context, currentWalletHeight int64) {
	// if it is the time issuing transaction, return
	if m.redeemingFlag.Load().(bool) {
		return
	}

	checkChan := m.walletClient.CheckChan()
	if checkChan == nil {
		log.Errorf("fail to acquire the check channel for updating transaction status, please check the pool or wallet status")
		return
	}

	userService := service.GetUserService()
	wg := sync.WaitGroup{}

	txInfos, err := userService.GetPendingOrderTransaction(ctx, m.db)
	if err != nil {
		log.Errorf("fail to acquire pending transaction request hash from database error : %s", err)

	}
	txInfoIDMapping := make(map[uint64]*model.OrderTransactionStatus, len(txInfos))
	requestHashIDMapping := make(map[string][]uint64, len(txInfos))
	for i := 0; i < len(txInfos); i++ {
		txInfoIDMapping[txInfos[i].ID] = txInfos[i]
		if _, ok := requestHashIDMapping[txInfos[i].RequestTransactionHash]; !ok {
			requestHashIDMapping[txInfos[i].RequestTransactionHash] = make([]uint64, 0, 5)
		}
		requestHashIDMapping[txInfos[i].RequestTransactionHash] = append(requestHashIDMapping[txInfos[i].RequestTransactionHash], txInfos[i].ID)
	}
	// update by request hash firstly
	for requestHash, otiIDs := range requestHashIDMapping {
		response := make(chan *abejson.TxStatus, 1)
		requestTxHash, err := utils.NewHashFromStr(requestHash)
		if err != nil {
			log.Errorf("transaction_request_hash in IDs %v is wrong string for hash", otiIDs)
			continue
		}
		checkChan <- &abejson.TxStatusInfo{
			TxHash:        nil,
			RequestTxHash: requestTxHash,
			ResponseChan:  response,
		}
		wg.Add(1)
		go func(IDs []uint64, previousStatus int, previousTxHash string, requestHash string) {
			defer wg.Done()
			resp := <-response
			if resp.Status == -2 {
				log.Warnf("fail to query for IDs [%v], waiting for next time", IDs)
				return
			}

			if resp.TxHash == nil || resp.TxHash.IsEqual(&utils.ZeroHash) {
				log.Errorf("Error for updating the order_transaction_infos for request hash %s with ids %v: %v", requestHash, IDs, err)
				return
			}

			if resp.Status == do.TransactionManual {
				log.Errorf("Please manual handle transaction for IDs [%v]", IDs)
			}

			txHashStr := resp.TxHash.String()
			status := resp.Status
			if previousTxHash != "" && previousTxHash != txHashStr {
				log.Errorf("Error for updating the order_transaction_infos for request hash %s with ids %v: resp.TxHash %s unmatched with previous tx hash %s", requestHash, IDs, txHashStr, previousStatus)
				txHashStr = previousTxHash
				status = do.TransactionManual
			}

			err = userService.SetOrderTransactionInfoByIDs(ctx, m.db, IDs, txHashStr, previousStatus, status, int(currentWalletHeight))
			if err != nil {
				log.Errorf("Error for updating the order_transaction_infos for request hash %s with ids %v: %v", requestHash, IDs, err)
				return
			}
			log.Infof("transaction hash %s update its status %d with request hash %s", resp.TxHash.String(), resp.Status, requestHash)
		}(otiIDs, txInfoIDMapping[otiIDs[0]].Status, txInfoIDMapping[otiIDs[0]].TransactionHash, requestHash)
	}
	wg.Wait()
}

func (m *OrderManager) HandleWalletClientNotification(notification *walletclient.Notification) {
	userService := service.GetUserService()
	var originHeight int64
	switch notification.Type {
	case walletclient.NTTxAccepted:
		log.Trace("Order manager receives a accepted transaction notification from wallet client, updating...")
		txInfo, ok := notification.Data.(*walletclient.TransactionInfo)
		if !ok {
			log.Errorf("Order manager accepted notification is not a transaction hash.")
			break
		}
		if txInfo == nil {
			log.Errorf("Received transaction notification is nil")
			break
		}
		originHeight = int64(txInfo.Height)
		err := userService.SetOrderTransactionInfoByTxHash(context.Background(), m.db, txInfo.TxHash, do.TransactionInChain, int(txInfo.Height))
		if err != nil {
			log.Errorf("Error for updating the order_transaction_infos status:%s", err)
			break
		}
		log.Infof("tx hash accepted %s", txInfo.TxHash)
		m.currentWalletHeightMtx.Lock()
		m.currentWalletHeight = originHeight
		m.currentWalletHeightMtx.Unlock()
		log.Infof("send height %d to check handler", originHeight)

	case walletclient.NTTxRollback:
		log.Trace("Order manager receives a rollback transaction notification from wallet client, updating...")
		txInfo, ok := notification.Data.(*walletclient.TransactionInfo)
		if !ok {
			log.Errorf("Order manager accepted notification is not a transaction hash.")
			break
		}
		if txInfo == nil {
			log.Errorf("Received transaction notification is nil")
			break
		}
		originHeight = int64(txInfo.Height)
		err := userService.SetOrderTransactionInfoByTxHash(context.Background(), m.db, txInfo.TxHash, do.TransactionWaiting, int(txInfo.Height))
		if err != nil {
			log.Errorf("Error for updating the order_transaction_infos status:%s", err)
			break
		}
		log.Infof("tx hash rollback %s", txInfo.TxHash)
		m.currentWalletHeightMtx.Lock()
		m.currentWalletHeight = originHeight
		m.currentWalletHeightMtx.Unlock()
		log.Infof("send height %d to check handler", originHeight)

	case walletclient.NTTxInvalid:
		log.Trace("Order manager receives a invalid transaction notification from wallet client, updating...")
		txInfo, ok := notification.Data.(*walletclient.TransactionInfo)
		if !ok {
			log.Errorf("Order manager accepted notification is not a transaction hash.")
			break
		}
		if txInfo == nil {
			log.Errorf("Received transaction notification is nil")
			break
		}
		originHeight = int64(txInfo.Height)
		err := userService.SetOrderTransactionInfoByTxHash(context.Background(), m.db, txInfo.TxHash, do.TransactionInvalid, int(txInfo.Height))
		if err != nil {
			log.Errorf("Error for updating the order_transaction_infos status:%s", err)
			break
		}
		log.Infof("tx hash invalid %s", txInfo.TxHash)
		m.currentWalletHeightMtx.Lock()
		m.currentWalletHeight = originHeight
		m.currentWalletHeightMtx.Unlock()
		log.Infof("send height %d to check handler", originHeight)

	default:
		log.Warnf("Unknown notification from wallet")
	}
}

func (m *OrderManager) SetWalletClient(cli *walletclient.RPCClient) {
	m.walletClient = cli
}

func (m *OrderManager) EnableAutoFlag() bool {
	return m.enableAutoFlag
}

func (m *OrderManager) AutoFlag() bool {
	return m.autoFlag
}

func (m *OrderManager) SetWalletPrivPass(walletPrivPass string) bool {
	successed := m.walletClient.SetWalletPrivPass(walletPrivPass)
	if successed {
		m.autoFlag = true
	}
	return successed
}

func (m *OrderManager) Stop() error {
	if atomic.AddInt32(&m.shutdown, 1) != 1 {
		log.Infof("Pool TCP socket RPC server is already in the process of shutting down")
		return nil
	}
	log.Warnf("Pool TCP socket RPC server shutting down...")
	close(m.quit)
	m.wg.Wait()
	log.Infof("Pool TCP socket RPC server shutdown complete")
	return nil
}

func (m *OrderManager) Start() {
	m.wg.Add(2)

	go m.checkTransactionHandler()
	go m.redeemBalanceHandler()
}
